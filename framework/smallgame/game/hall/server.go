package hall

// grpc server 服务

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/codes"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gen/service/hallpb"
	"golang.org/x/sync/errgroup"
)

type hall_server struct {
	hallpb.UnimplementedHallServer
	hall *hall_service
}

// 踢用户下线
func (self *hall_server) Kick(ctx context.Context, req *hallpb.KickRequest) (*hallpb.KickResponse, error) {
	resp := &hallpb.KickResponse{}
	if err := self.hall.Kick(ctx, req.UserId, req.Reason); err != nil {
		return nil, err
	}
	return resp, nil
}

// 顶号下线
func (self *hall_server) UserInstead(ctx context.Context, req *hallpb.UserInsteadRequest) (*hallpb.UserInsteadResponse, error) {
	resp := &hallpb.UserInsteadResponse{}
	reason := fmt.Sprintf("账号在%s登录", req.Address)
	if err := self.hall.Instead(ctx, req.UserId, reason); err != nil {
		resp.ErrorCode = codes.Code(err)
		resp.ErrorMsg = codes.Msg(err)
		return resp, nil
	} else {
		return resp, nil
	}
}

func (self *hall_server) SendMessage(context.Context, *hallpb.SendMessageRequest) (*hallpb.SendMessageResponse, error) {
	resp := &hallpb.SendMessageResponse{}
	return resp, nil
}

func (self *hall_server) CallMessage(context.Context, *hallpb.CallMessageRequest) (*hallpb.CallMessageResponse, error) {
	resp := &hallpb.CallMessageResponse{}
	return resp, nil
}

func (self *hall_server) MustPush(ctx context.Context, req *hallpb.MustPushRequest) (resp *hallpb.MustPushResponse, err error) {
	resp = &hallpb.MustPushResponse{}
	var push gira.ProtoPush
	userId := req.UserId
	_, _, push, err = self.hall.proto.PushDecode(req.Data)
	if err != nil {
		return
	}
	defer func() {
		log.Warnw("grpc_hall_server must push panic", "user_id", req.UserId, "route", push.GetPushName())
		if e := recover(); e != nil {
			log.Error(e)
			debug.PrintStack()
			err = e.(error)
		}
	}()
	if v, ok := self.hall.sessionDict.Load(userId); !ok {
		err = errors.ErrNoSession
		return
	} else {
		session, _ := v.(*hall_sesssion)
		// WARN: chPush有可能已经关闭
		session.chPeerPush <- push
		return
	}
}

func (self *hall_server) Info(ctx context.Context, req *hallpb.InfoRequest) (*hallpb.InfoResponse, error) {
	resp := &hallpb.InfoResponse{
		BuildTime:    facade.GetBuildTime(),
		AppVersion:   facade.GetAppVersion(),
		SessionCount: self.hall.sessionCount,
	}
	return resp, nil
}

func (self *hall_server) HealthCheck(ctx context.Context, req *hallpb.HealthCheckRequest) (*hallpb.HealthCheckResponse, error) {
	resp := &hallpb.HealthCheckResponse{
		PlayerCount: self.hall.sessionCount,
	}
	resp.Status = self.hall.status
	return resp, nil
}

func (self *hall_server) GateStream(client hallpb.Hall_GateStreamServer) error {
	// ticker := time.NewTicker(time.Duration(self.hall.config.Framework.Hall.GatewayReportInterval) * time.Second)
	defer func() {
		// ticker.Stop()
		log.Infow("gate stream exit")
	}()
	// if self.hall.isDestory {
	// 	return gira.ErrAlreadyDestory
	// }
	errGroup, errCtx := errgroup.WithContext(self.hall.gateStreamCtx)
	// 定时发送在线人数
	// errGroup.Go(func() error {
	// 	for {
	// 		select {
	// 		case <-self.hall.gateStreamCtx.Done():
	// 			return self.hall.gateStreamCtx.Err()
	// 		case <-errCtx.Done():
	// 			return errCtx.Err()
	// 		}
	// 	}
	// })
	// 循环接收gate的命令
	errGroup.Go(func() error {
		for {
			select {
			case <-self.hall.gateStreamCtx.Done():
				return self.hall.gateStreamCtx.Err()
			case <-errCtx.Done():
				return errCtx.Err()
				// default:
				// 	req, err := client.Recv()
				// 	if err != nil {
				// 		log.Infow("gate recv fail", "error", err)
				// 		// 连接关闭，返回错误，会触发errCtx的cancel
				// 		return err
				// 	} else {
				// 		log.Infow("gate recv", "req", req)
				// 	}
			}
		}
	})
	return errGroup.Wait()
}

func (self *hall_server) ClientStream(client hallpb.Hall_ClientStreamServer) error {
	var req *hallpb.ClientMessageRequest
	var err error
	var memberId string
	// if self.hall.isDestory {
	// 	return gira.ErrAlreadyDestory
	// }
	if req, err = client.Recv(); err != nil {
		log.Errorw("recv fail", "error", err)
		return err
	}
	hall := self.hall
	sessionId := req.SessionId
	// 会话的第一条消息一定要带memberid来完成登录
	memberId = req.MemberId
	log.Infow("bind memberid", "session_id", sessionId, "member_id", memberId)
	if memberId == "" {
		log.Errorw("invalid memberid", "session_id", sessionId, "member_id", memberId)
		return errors.ErrInvalidMemberId
	}
	// 申请建立会话
	var session *hall_sesssion
	session, err = newSession(self.hall, sessionId, memberId)
	if err != nil {
		log.Errorw("create session fail", "session_id", sessionId, "error", err)
		return err
	}
	go session.serve()

	// 建立成功， 函数结束后通知释放
	// response管理
	session.chClientResponse = make(chan *hallpb.ClientMessageResponse, hall.config.ResponseBufferSize)
	session.chClientRequest = make(chan *hallpb.ClientMessageRequest, hall.config.RequestBufferSize)

	// 绑定到hall
	clientCtx, clientCancelFunc := context.WithCancel(self.hall.backgroundCtx)
	session.clientCancelFunc = clientCancelFunc
	// recv ctrl context
	defer func() {
		clientCancelFunc()
		if err := session.Close(self.hall.backgroundCtx); err != nil {
			log.Errorw("session close fail", "session_id", sessionId, "error", err)
		}
	}()
	// 读取client消息，塞进管道
	chRecvErr := make(chan struct{})
	go func() {
		defer func() {
			defer close(session.chClientRequest)
			log.Infow("read goroutine exit", "session_id", sessionId)
		}()
		session.chClientRequest <- req
		for {
			req, err = client.Recv()
			if clientCtx.Err() != nil {
				return
			}
			if err != nil {
				log.Infow("client recv fail", "error", err, "session_id", sessionId)
				close(chRecvErr)
				return
			} else {
				session.chClientRequest <- req
			}
		}
	}()
	// 将response消息转发给网关
	func() {
		defer func() {
			select {
			case <-session.chClientResponse:
			default:
				close(session.chClientResponse)
			}
			log.Infow("response goroutine exit", "session_id", sessionId)
		}()
		for {
			select {
			case resp := <-session.chClientResponse:
				if resp == nil {
					log.Infow("response chan close", "session_id", sessionId)
					return
				}
				if err := client.Send(resp); err != nil {
					log.Warnw("client response fail", "session_id", sessionId, "name", resp.Route, "error", err)
				} else {
					log.Infow("response", "session_id", sessionId, "name", resp.Route, "req_id", resp.ReqId, "data", len(resp.Data))
				}
			case <-clientCtx.Done():
				// 发完剩下的消息
				close(session.chClientResponse)
				goto FLUSH_RESPONSE
			case <-chRecvErr:
				// 已经断开了，马上结束
				return
			}
		}
	FLUSH_RESPONSE:
		for {
			select {
			case resp := <-session.chClientResponse:
				if resp == nil {
					log.Infow("response chan close", "session_id", sessionId)
					return
				}
				if err := client.Send(resp); err != nil {
					log.Warnw("client response fail", "session_id", sessionId, "name", resp.Route, "error", err)
				} else {
					log.Infow("response", "session_id", sessionId, "name", resp.Route, "req_id", resp.ReqId, "data", len(resp.Data))
				}
			case <-chRecvErr:
				// 已经断开了，马上结束
				return
			}
		}
	}()
	log.Infow("client stream exit", "error", err, "session_id", sessionId, "member_id", memberId)
	return err
}
