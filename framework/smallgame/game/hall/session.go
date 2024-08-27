package hall

import (
	"context"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/actor"
	"github.com/Lyndon-Zhang/gira/codes"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/game"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gen/service/hallpb"
)

// 锁是和session绑定的，因此由session来抢占会释放

// session 被动从stream接收消息
type hall_sesssion struct {
	*actor.Actor
	hall             *hall_service
	ctx              context.Context
	cancelFunc       context.CancelFunc
	sessionId        uint64
	memberId         string
	userId           string
	avatar           game.UserAvatar
	player           game.Player
	chClientResponse chan *hallpb.ClientMessageResponse // 由stream负责关闭
	chClientRequest  chan *hallpb.ClientMessageRequest  // 由stream负责关闭
	chPeerPush       chan gira.ProtoPush                // 其他节点，或者自己节点转发来的的push消息，由session负责关闭
	clientCancelFunc context.CancelFunc
	isClosed         int32
	mu               sync.Mutex
}

func newSession(hall *hall_service, sessionId uint64, memberId string) (session *hall_sesssion, err error) {
	var userId string
	var avatar game.UserAvatar
	ctx := hall.backgroundCtx
	avatar, err = hall.hallHandler.NewUser(ctx, memberId)
	if err != nil {
		return
	}
	userId = avatar.GetUserId().Hex()
	// 抢占锁
	var peer *gira.Peer
	peer, err = facade.LockLocalUser(userId)
	log.Infow("lock local user return", "session_id", sessionId, "peer", peer, "err", err)
	if err != nil {
		if peer == nil {
			return nil, err
		} else {
			// 顶号下线
			log.Infow("user instead", "session_id", userId)
			var resp *hallpb.UserInsteadResponse
			if resp, err = hallpb.DefaultHallClients.Unicast().WherePeer(peer).UserInstead(ctx, &hallpb.UserInsteadRequest{
				UserId:  userId,
				Address: "其他地方",
				// Address: fmt.Sprintf("%d服", facade.GetAppId()),
			}); err != nil {
				log.Infow("user instead fail", "session_id", sessionId, "error", err)
				return
			} else if resp.ErrorCode != 0 {
				err = codes.New(resp.ErrorCode, resp.ErrorMsg)
				log.Infow("user instead fail", "session_id", sessionId, "error", err)
				return
			} else {
				log.Infow("user instead success", "session_id", sessionId)
			}
		}
		// 再次抢占锁
		peer, err = facade.LockLocalUser(userId)
		log.Infow("lock local user return", "session_id", sessionId, "peer", peer, "err", err)
		if err != nil {
			return
		}
	}
	defer func() {
		if err != nil {
			peer, err := facade.UnlockLocalUser(userId)
			log.Infow("unlock local user return", "session_id", sessionId, "peer", peer, "err", err)
		}
	}()

	// 如果还没释放完成，则失败
	if _, ok := hall.sessionDict.Load(userId); ok {
		log.Warnw("unexpect session", "session_id", sessionId)
		err = errors.ErrUserInstead
		return
	}
	// 创建会话
	session = &hall_sesssion{
		hall:      hall,
		sessionId: sessionId,
		userId:    userId,
		memberId:  memberId,
		avatar:    avatar,
		Actor:     actor.NewActor(hall.config.SessionActorBuffSize),
	}
	session.ctx, session.cancelFunc = context.WithCancel(ctx)

	if _, loaded := hall.sessionDict.LoadOrStore(userId, session); loaded {
		log.Warnw("session store fail", "session_id", sessionId)
		err = errors.ErrUserInstead
		return
	}
	atomic.AddInt64(&hall.sessionCount, 1)
	var player game.Player
	// 加载数据
	if player, err = session.load(); err != nil {
		// 数据加载失败，释放锁
		hall.sessionDict.Delete(userId)
		atomic.AddInt64(&hall.sessionCount, -1)
		return
	} else {
		session.player = player
	}
	return
}

func (session *hall_sesssion) Yield(f func() error) error {
	session.mu.Unlock()
	defer func() {
		session.mu.Lock()
	}()
	return f()
}

func (session *hall_sesssion) Go(f func() error) error {
	session.mu.Lock()
	defer func() {
		session.mu.Unlock()
	}()
	return f()
}

// 主要的业务逻辑的主协程
// 协程1：处理定时器,客户端消息
// 协程2: 处理grpc请求
// 协程3: 处理节点client proto push
func (session *hall_sesssion) serve() {
	ticker := time.NewTicker(1 * time.Second)
	hall := session.hall
	saveTicker := time.NewTicker(time.Duration(session.hall.config.BgSaveInterval) * time.Second)
	session.chPeerPush = make(chan gira.ProtoPush, hall.config.PushBufferSize)
	defer func() {
		ticker.Stop()
		saveTicker.Stop()
		close(session.chPeerPush)
	}()
	sessionId := session.sessionId
	go func() {
		for {
			select {
			case req := <-session.chPeerPush:
				if req == nil {
					continue
				}
				if session.isClosed == 0 {
					if err := session.processPeerPush(req); err != nil {
						log.Infow("push fail", "session_id", sessionId, "error", err, "name", req.GetPushName())
					} else {
						log.Infow("push fail", "session_id", sessionId, "error", "already closed", "name", req.GetPushName())
					}
				}
			case <-session.ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case r := <-session.Inbox():
				session.processActorRequest(r)
			case <-session.ctx.Done():
				return
			}
		}
	}()
	for {
		select {
		// 定时保存数据
		case <-saveTicker.C:
			if session.isClosed == 0 {
				session.bgSave()
			}
		// 逻辑定时器
		case <-ticker.C:
			if session.isClosed == 0 {
				session.update()
			}
		// 处理客户端消息
		case req := <-session.chClientRequest:
			if req == nil {
				continue
			}
			if session.isClosed == 0 {
				if err := session.processClientMessage(req); err != nil {
					log.Infow("request fail", "session_id", sessionId, "error", err, "name")
				}
			} else {
				log.Infow("request fail", "session_id", sessionId, "error", "already closed")
			}
		case <-session.ctx.Done():
			log.Infow("session exit", "session_id", sessionId)
			// session.Close(context.Background())
			return
		}
	}
}

// session加锁
func (session *hall_sesssion) update() {
	defer func() {
		session.mu.Unlock()
		if e := recover(); e != nil {
			log.Error(e)
			debug.PrintStack()
		}
	}()
	session.mu.Lock()
	session.player.Update()
}

// session加锁
func (session *hall_sesssion) bgSave() {
	defer func() {
		session.mu.Unlock()
		if e := recover(); e != nil {
			log.Error(e)
			debug.PrintStack()
		}
	}()
	session.mu.Lock()
	session.player.BgSave(session.ctx)
}

// 加载数据
func (session *hall_sesssion) load() (player game.Player, err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Error(e)
			debug.PrintStack()
			err = e.(error)
		}
	}()
	// player的ctx和session的ctx平级，player并不和session绑定，player可以将自己缓存起来，下次相同玩家登录的时候再复用
	player, err = session.hall.hallHandler.NewPlayer(session.hall.backgroundCtx, session, session.memberId, session.avatar)
	if err != nil {
		return
	}
	err = player.Load(session.hall.backgroundCtx, session.memberId, session.userId)
	return
}

// 发送控制消息，并关闭会话
// 协程安全
func (self *hall_sesssion) sendPacketAndClose(ctx context.Context, typ hallpb.PacketType, reason string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			log.Errorw("sendPacketAndClose", "type", typ, "reason", reason)
			log.Error(e)
			debug.PrintStack()
			err = e.(error)
		}
	}()
	resp := &hallpb.ClientMessageResponse{}
	resp.Type = typ
	resp.SessionId = self.sessionId
	resp.ReqId = 0
	resp.Data = []byte(reason)
	self.chClientResponse <- resp
	err = self.Close(ctx)
	return nil
}

// 处理actor的request
func (session *hall_sesssion) processActorRequest(req actor.Request) {
	defer func() {
		session.mu.Unlock()
	}()
	session.mu.Lock()
	req.Next()
}

// 处理peer的push消息
func (session *hall_sesssion) processPeerPush(req gira.ProtoPush) (err error) {
	sessionId := session.sessionId
	var name string = req.GetPushName()
	log.Infow("peer push", "session_id", sessionId, "name", name)
	timeoutCtx, timeoutFunc := context.WithTimeout(session.ctx, 10*time.Second)
	defer func() {
		session.mu.Unlock()
		timeoutFunc()
		if e := recover(); e != nil {
			log.Error(e)
			debug.PrintStack()
			err = e.(error)
		}
	}()
	session.mu.Lock()
	err = session.hall.proto.PushDispatch(timeoutCtx, session.hall.playerHandler, session.player, name, req)
	if err != nil {
		log.Errorw("push dispatch fail", "session_id", sessionId, "error", err, "name", name)
		return
	}
	return
}

// 处理客户端的消息
// session加锁
func (session *hall_sesssion) processClientMessage(message *hallpb.ClientMessageRequest) (err error) {
	sessionId := session.sessionId
	var name string
	var reqId int32
	var req interface{}
	var resp []byte
	var pushArr []gira.ProtoPush
	name, reqId, req, err = session.hall.proto.RequestDecode(message.Data)
	if err != nil {
		log.Errorw("request decode fail", "session_id", sessionId, "error", err)
		return
	}
	log.Infow("request ", "session_id", sessionId, "name", name, "req_id", reqId, "data", message.Data)

	timeoutCtx, timeoutFunc := context.WithTimeout(session.ctx, 10*time.Second)
	defer func() {
		session.mu.Unlock()
		timeoutFunc()
		if e := recover(); e != nil {
			// err = gira.ErrBrokenChannel
			log.Error(e)
			debug.PrintStack()
			err = e.(error)
		}
	}()
	session.mu.Lock()
	resp, pushArr, err = session.hall.proto.RequestDispatch(timeoutCtx, session.hall.playerHandler, session.player, name, reqId, req, session.hall.config.TraceProtoDebugMsg)
	if err != nil {
		log.Errorw("request dispatch fail", "session_id", sessionId, "error", err)
		return
	}
	if session.isClosed != 0 {
		return nil
	}
	// log.Debugw("data response", "session_id", sessionId, "data response", len(dataResp))
	response := &hallpb.ClientMessageResponse{}
	response.Type = hallpb.PacketType_DATA
	response.SessionId = message.SessionId
	response.ReqId = message.ReqId
	response.Data = resp
	response.Route = name
	session.chClientResponse <- response
	for _, push := range pushArr {
		if dataPush, err := session.hall.proto.PushEncode(push); err != nil {
			log.Errorw("push fail", "session_id", sessionId, "error", err)
		} else {
			log.Infow("push to client", "session_id", sessionId, "name", push.GetPushName(), "data", len(dataPush))
			response := &hallpb.ClientMessageResponse{}
			response.Type = hallpb.PacketType_DATA
			response.SessionId = message.SessionId
			response.ReqId = 0
			response.Data = dataPush
			response.Route = push.GetPushName()
			session.chClientResponse <- response
		}
	}
	return
}

// 关闭session
// 协程安全
func (session *hall_sesssion) Close(ctx context.Context) (err error) {
	if !atomic.CompareAndSwapInt32(&session.isClosed, 0, 1) {
		err = errors.ErrSessionClosed
		return
	}
	userId := session.userId
	sessionId := session.sessionId
	log.Infow("session close", "session_id", sessionId)
	// 断开连接
	session.clientCancelFunc()
	defer func() {
		if e := recover(); e != nil {
			log.Error(err)
			debug.PrintStack()
			var ok bool
			var s string
			if err, ok = e.(error); ok {
			} else if s, ok = e.(string); ok {
				err = errors.New(s)
			} else {
				err = errors.New("panic")
			}
		}
		session.cancelFunc()
		peer, err := facade.UnlockLocalUser(userId)
		log.Infow("unlock local user return", "session_id", sessionId, "peer", peer, "err", err)
		// 从agent dict释放
		session.hall.sessionDict.Delete(userId)
		atomic.AddInt64(&session.hall.sessionCount, -1)
		log.Infow("session close finished", "session_id", sessionId)
	}()
	// 开始关闭操作
	if session.player != nil {
		session.player.Logout(ctx)
	}
	return
}

// 踢下线
// 协程安全
func (self *hall_sesssion) Kick(ctx context.Context, reason string) (err error) {
	return self.sendPacketAndClose(ctx, hallpb.PacketType_KICK, reason)
}

// 顶号下线
// 协程安全
func (self *hall_sesssion) Instead(ctx context.Context, reason string) (err error) {
	return self.sendPacketAndClose(ctx, hallpb.PacketType_USER_INSTEAD, reason)
}

// 推送消息
// 协程安全
func (self *hall_sesssion) Push(push gira.ProtoPush) (err error) {
	var data []byte
	if data, err = self.hall.proto.PushEncode(push); err != nil {
		return
	} else {
		defer func() {
			if e := recover(); e != nil {
				log.Error(e)
				debug.PrintStack()
				err = e.(error)
			}
		}()
		resp := &hallpb.ClientMessageResponse{}
		resp.Type = hallpb.PacketType_DATA
		resp.SessionId = self.sessionId
		resp.ReqId = 0
		resp.Data = data
		resp.Route = push.GetPushName()
		// WARN: chResponse有可能已经关闭
		self.chClientResponse <- resp
		return
	}
}
