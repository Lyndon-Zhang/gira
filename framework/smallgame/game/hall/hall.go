package hall

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Lyndon-Zhang/gira"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/game"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/game/config"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gen/service/hallpb"
	"github.com/Lyndon-Zhang/gira/options/service_options"
)

type HallService interface {
	gira.Service
	SessionCount() int64
	Push(ctx context.Context, userId string, req gira.ProtoPush) error
	MustPush(ctx context.Context, userId string, resp gira.ProtoPush) (err error)
}

func NewService(proto gira.Proto, config config.GameConfig, hallHandler game.HallHandler, playerHandler gira.ProtoHandler) (HallService, error) {
	service := &hall_service{
		proto:         proto,
		config:        config,
		hallHandler:   hallHandler,
		playerHandler: playerHandler,
		status:        hallpb.HallStatus_UnAvailable,
	}
	service.hallServer = &hall_server{
		hall: service,
	}
	return service, nil
}

func GetServiceName() string {
	return facade.NewServiceName(hallpb.HallServerName, service_options.WithAsAppServiceOption())
}

type hall_service struct {
	hallServer           *hall_server
	ctx                  context.Context
	gateStreamCtx        context.Context
	gateStreamCancelFunc context.CancelFunc
	backgroundCtx        context.Context
	backgroundCancelFunc context.CancelFunc
	sessionDict          sync.Map
	sessionCount         int64
	hallHandler          game.HallHandler
	playerHandler        gira.ProtoHandler
	proto                gira.Proto
	config               config.GameConfig
	status               hallpb.HallStatus
}

func (hall *hall_service) OnStart(ctx context.Context) error {
	hall.ctx = ctx
	if _, err := facade.RegisterServiceName(GetServiceName()); err != nil {
		return err
	}
	// 后台运行
	hall.backgroundCtx, hall.backgroundCancelFunc = context.WithCancel(context.Background())
	hall.gateStreamCtx, hall.gateStreamCancelFunc = context.WithCancel(hall.backgroundCtx)
	hallpb.RegisterHallServer(facade.GrpcServer(), hall.hallServer)
	return nil
}

func (hall *hall_service) OnStop() error {
	return nil
}

func (hall *hall_service) Serve() error {
	//
	// 1.服务器关闭时保存数据后再退出
	// 2.处理actor调用
	//
	// hall.ctx, hall.cancelFunc = context.WithCancel(facade.Context())
	// defer hall.cancelFunc()
	// facade.Go(func() error {
	// 	select {
	// 	case <-hall.ctx.Done():
	// 		// 服务器停止，保存数据
	// 		for {
	// 			log.Infow("hall停止中", "session_count", hall.sessionCount)
	// 			if hall.sessionCount <= 0 {
	// 				break
	// 			}
	// 			time.Sleep(100 * time.Millisecond)
	// 		}
	// 	}
	// 	return nil
	// })
	hall.status = hallpb.HallStatus_OK
	for {
		select {
		case <-hall.ctx.Done():
			log.Infow("hall exit")
			goto TAG_CLEAN_UP
		}
	}
TAG_CLEAN_UP:
	hall.status = hallpb.HallStatus_UnAvailable
	hall.gateStreamCancelFunc()

	for {
		log.Infow("hall on stop", "session_count", hall.sessionCount)
		sessions := make([]*hall_sesssion, 0)
		hall.sessionDict.Range(func(key, value any) bool {
			session, _ := value.(*hall_sesssion)
			sessions = append(sessions, session)
			return true
		})
		for _, session := range sessions {
			session.Close(hall.backgroundCtx)
		}
		log.Infow("hall on stop", "session_count", hall.sessionCount)
		if hall.sessionCount <= 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	hall.backgroundCancelFunc()
	return nil
}

func (hall *hall_service) SessionCount() int64 {
	return hall.sessionCount
}

// 踢用户下线
// 协程程安全
func (hall *hall_service) Kick(ctx context.Context, userId string, reason string) (err error) {
	if v, ok := hall.sessionDict.Load(userId); !ok {
		return errors.ErrUserNotFound
	} else {
		session := v.(*hall_sesssion)
		return session.Kick(ctx, reason)
	}
}

// 顶号下线
// 协程程安全
func (hall *hall_service) Instead(ctx context.Context, userId string, reason string) error {
	log.Infow("user instead", "user_id", userId)
	if v, ok := hall.sessionDict.Load(userId); !ok {
		// 偿试解锁
		peer, err := facade.UnlockLocalUser(userId)
		log.Infow("unlock local user return", "user_id", userId, "peer", peer, "err", err)
		if err != nil {
			return err
		} else {
			return nil
		}
	} else {
		session := v.(*hall_sesssion)
		if err := session.Instead(ctx, reason); err != nil {
			log.Errorw("user instead fail", "user_id", userId, "error", err)
			return err
		} else {
			log.Infow("user instead success", "user_id", userId)
			return nil
		}
	}
}

// 推送消息给其他玩家
func (hall *hall_service) Push(ctx context.Context, userId string, push gira.ProtoPush) (err error) {
	defer func() {
		log.Warnw("hall_server push panic", "user_id", userId, "route", push.GetPushName())
		if e := recover(); e != nil {
			log.Error(e)
			debug.PrintStack()
			err = e.(error)
		}
	}()
	if v, ok := hall.sessionDict.Load(userId); !ok {
		var data []byte
		var peer *gira.Peer
		data, err = hall.proto.PushEncode(push)
		if err != nil {
			return
		}
		peer, err = facade.WhereIsUser(userId)
		if err != nil {
			return
		}
		if _, err = hallpb.DefaultHallClients.Unicast().WherePeer(peer).MustPush(ctx, &hallpb.MustPushRequest{
			UserId: userId,
			Data:   data,
		}); err != nil {
			return
		}
		return
	} else {
		session, _ := v.(*hall_sesssion)
		// WARN: chPush有可能已经关闭
		session.chPeerPush <- push
		return
	}
}

// 推送消息给其他玩家
func (hall *hall_service) MustPush(ctx context.Context, userId string, push gira.ProtoPush) (err error) {
	defer func() {
		log.Warnw("hall_server must push panic", "user_id", userId, "route", push.GetPushName())
		if e := recover(); e != nil {
			log.Error(e)
			debug.PrintStack()
			err = e.(error)
		}
	}()
	if v, ok := hall.sessionDict.Load(userId); !ok {
		var data []byte
		var peer *gira.Peer
		data, err = hall.proto.PushEncode(push)
		if err != nil {
			return
		}
		peer, err = facade.WhereIsUser(userId)
		if err != nil {
			return
		}
		if _, err = hallpb.DefaultHallClients.Unicast().WherePeer(peer).MustPush(ctx, &hallpb.MustPushRequest{
			UserId: userId,
			Data:   data,
		}); err != nil {
			return
		}
		return
	} else {
		session, _ := v.(*hall_sesssion)
		// WARN: chPush有可能已经关闭
		session.chPeerPush <- push
		return
	}
}
