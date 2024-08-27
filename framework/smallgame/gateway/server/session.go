package server

import (
	"context"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/Lyndon-Zhang/gira"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gen/service/hallpb"
	"golang.org/x/sync/errgroup"
)

type client_session struct {
	ctx             context.Context
	cancelFunc      context.CancelFunc
	sessionId       uint64
	memberId        string
	client          gira.GatewayConn
	stream          hallpb.Hall_ClientStreamClient
	server          *Server
	pendingMessages []gira.GatewayMessage
}

func newSession(server *Server, sessionId uint64, memberId string) *client_session {
	return &client_session{
		server:          server,
		sessionId:       sessionId,
		memberId:        memberId,
		pendingMessages: make([]gira.GatewayMessage, 0),
	}
}

func (session *client_session) serve(ctx context.Context, client gira.GatewayConn, message gira.GatewayMessage, dataReq gira.ProtoRequest) error {
	sessionId := session.sessionId
	var err error
	var stream hallpb.Hall_ClientStreamClient
	hall := session.server
	server := hall.SelectPeer()
	log.Infow("session open", "session_id", sessionId, "peer", server)
	if server == nil {
		hall.loginErrResponse(message, dataReq, errors.ErrUpstreamUnavailable)
		return errors.ErrUpstreamUnavailable
	}
	// stream绑定server
	streamCtx, streamCancelFunc := context.WithCancel(ctx)
	defer func() {
		streamCancelFunc()
	}()
	stream, err = server.NewClientStream(streamCtx)
	if err != nil {
		session.server.loginErrResponse(message, dataReq, errors.ErrUpstreamUnreachable)
		return err
	}
	session.stream = stream
	session.client = client
	// session 绑定 hall
	session.ctx, session.cancelFunc = context.WithCancel(hall.ctx)
	errGroup, _ := errgroup.WithContext(session.ctx)
	atomic.AddInt64(&session.server.SessionCount, 1)
	defer func() {
		log.Infow("session close", "session_id", sessionId)
		atomic.AddInt64(&session.server.SessionCount, -1)
		session.cancelFunc()
	}()
	// 将上游消息转发到客户端
	errGroup.Go(func() (err error) {
		defer func() {
			log.Infow("upstream=>client goroutine exit", "session_id", sessionId)
			if e := recover(); e != nil {
				log.Error(e)
				debug.PrintStack()
				err = e.(error)
				session.cancelFunc()
			}
		}()
		for {
			var resp *hallpb.ClientMessageResponse
			// 上游关闭时，stream并不会返回，会一直阻塞
			if resp, err = stream.Recv(); err == nil {
				session.processStreamMessage(resp)
				// } else if err != io.EOF {
				// 	log.Infow("上游连接异常关闭", "session_id", sessionId, "error", err)
				// 	session.cancelFunc()
				// 	return err
			} else {
				select {
				case <-session.ctx.Done():
					return session.ctx.Err()
				default:
				}
				log.Infow("上游连接关闭", "session_id", sessionId, "error", err)
				session.stream = nil
				client.SendServerSuspend("")
				// 重新选择节点
				for {
					// log.Infow("重新选择节点", "session_id", sessionId)
					server = hall.SelectPeer()
					if server != nil {
						log.Infow("重新选择节点", "session_id", sessionId, "full_name", server.FullName)
						streamCancelFunc()
						streamCtx, streamCancelFunc = context.WithCancel(server.ctx)
						stream, err = server.NewClientStream(streamCtx)
						if err != nil {
							streamCancelFunc()
							select {
							case <-session.ctx.Done():
								return session.ctx.Err()
							default:
								time.Sleep(1 * time.Second)
							}
						} else {
							session.stream = stream
							client.SendServerResume("")
							log.Infow("重新选择节点, 连接成功", "session_id", sessionId, "full_name", server.FullName)
							break
						}
					} else {
						select {
						case <-session.ctx.Done():
							return session.ctx.Err()
						default:
							log.Infow("无节点可用,1秒后重试", "session_id", sessionId)
							time.Sleep(1 * time.Second)
						}
					}
				}
			}
		}
	})
	errGroup.Go(func() (err error) {
		select {
		case <-session.ctx.Done():
			client.Close()
			streamCancelFunc()
			return session.ctx.Err()
		}
	})
	// 转发消息协程
	errGroup.Go(func() (err error) {
		defer func() {
			log.Infow("client=>upstream goroutine exit", "session_id", sessionId)
			if e := recover(); e != nil {
				log.Error(e)
				debug.PrintStack()
				err = e.(error)
				session.cancelFunc()
			}
		}()
		// 接收客户端消息
		if err = session.processClientMessage(message); err != nil {
			log.Infow("client=>upstream request fail", "session_id", sessionId, "error", err)
			session.pendingMessages = append(session.pendingMessages, message)
		}
		for {
			message, err = client.Recv(session.ctx)
			if err != nil {
				log.Infow("recv fail", "session_id", sessionId, "error", err)
				session.cancelFunc()
				return err
			}
			for len(session.pendingMessages) > 0 {
				p := session.pendingMessages[0]
				if err = session.processClientMessage(p); err != nil {
					break
				} else {
					log.Infow("补发消息", "session_id", sessionId, "req_id", message.ReqId())
					session.pendingMessages = session.pendingMessages[1:]
				}
			}
			if len(session.pendingMessages) > 0 {
				session.pendingMessages = append(session.pendingMessages, message)
				continue
			}
			if err = session.processClientMessage(message); err != nil {
				log.Warnw("client=>upstream request fail", "session_id", sessionId, "error", err)
				session.pendingMessages = append(session.pendingMessages, message)
			}
		}
	})
	err = errGroup.Wait()
	log.Infow("session wait", "error", err)
	return err
}

// 处理客户端的消息
func (self *client_session) processClientMessage(message gira.GatewayMessage) error {
	sessionId := self.sessionId
	memberId := self.memberId
	log.Infow("client=>upstream", "session_id", sessionId, "len", len(message.Payload()), "req_id", message.ReqId())
	if self.stream == nil {
		log.Warnw("当前服务器不可以用，无法转发", "req_id", message.ReqId())
		return errors.ErrUpstreamUnavailable
	} else {
		data := &hallpb.ClientMessageRequest{
			MemberId:  memberId,
			SessionId: sessionId,
			ReqId:     message.ReqId(),
			Data:      message.Payload(),
		}
		if err := self.stream.Send(data); err != nil {
			return err
		}
		return nil
	}
}

// 处理上游的消息
func (session *client_session) processStreamMessage(message *hallpb.ClientMessageResponse) error {
	sessionId := session.sessionId
	log.Infow("upstream=>client", "session_id", sessionId, "type", message.Type, "route", message.Route, "len", len(message.Data), "req_id", message.ReqId)

	switch message.Type {
	case hallpb.PacketType_DATA:
		if message.ReqId != 0 {
			session.client.Response(message.ReqId, message.Data)
		} else {
			session.client.Push("", message.Data)
		}
	case hallpb.PacketType_USER_INSTEAD:
		session.client.Kick(string(message.Data))
	case hallpb.PacketType_KICK:
		session.client.Kick(string(message.Data))
	}
	return nil
}
