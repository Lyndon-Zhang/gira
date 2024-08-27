package server

import (
	"context"
	"time"

	"github.com/Lyndon-Zhang/gira/codes"
	log "github.com/Lyndon-Zhang/gira/corelog"
	"github.com/Lyndon-Zhang/gira/errors"
	"github.com/Lyndon-Zhang/gira/facade"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gateway"
	"github.com/Lyndon-Zhang/gira/framework/smallgame/gateway/config"
	jwt "github.com/Lyndon-Zhang/gira/jwt/account"
)

type Server struct {
	proto      gira.Proto
	upstreams  upstream_map
	ctx        context.Context
	cancelFunc context.CancelFunc

	SessionCount    int64 // 会话数量
	ConnectionCount int64 // 连接数量
}

func NewServer(proto gira.Proto) *Server {
	return &Server{
		proto: proto,
	}
}

func (server *Server) OnCreate() error {
	server.ctx, server.cancelFunc = context.WithCancel(facade.Context())
	return nil
}

func (server *Server) loginErrResponse(message gira.GatewayMessage, req gira.ProtoRequest, err error) error {
	log.Info(err)
	resp, err := server.proto.NewResponse(req)
	if err == nil {
		return err
	}
	resp.SetErrorCode(codes.Code(err))
	resp.SetErrorMsg(codes.Msg(err))
	if data, err := server.proto.ResponseEncode("Login", int32(message.ReqId()), resp); err != nil {
		return err
	} else {
		message.Response(data)
	}
	return nil
}

// 客户端流数据
// 1.处理客户端发过来的第1个消息，必须满足LoginRequest接口
// 2.验证token
// 3.验证成功后，创建session,交给session处理接下来的客户端消息
func (server *Server) ServeClientStream(client gira.GatewayConn) {
	sessionId := client.Id()
	// 最大人数判断 -1:无限制 0=允许登录
	if config.Gateway.Framework.Gateway.MaxSessionCount == -1 {
		log.Warnw("reach max session count", "session_id", sessionId, "session_count", server.SessionCount, "max_session_count", config.Gateway.Framework.Gateway.MaxSessionCount)
	} else if config.Gateway.Framework.Gateway.MaxSessionCount == 0 || server.SessionCount >= config.Gateway.Framework.Gateway.MaxSessionCount {
		log.Warnw("reach max session count", "session_id", sessionId, "session_count", server.SessionCount, "max_session_count", config.Gateway.Framework.Gateway.MaxSessionCount)
		return
	}
	log.Infow("client stream open", "session_id", sessionId)
	defer func() {
		log.Infow("client stream exit", "session_id", sessionId)
	}()
	var req gira.GatewayMessage
	var err error
	var memberId string
	// 需要在一定时间内发登录协议
	timeoutCtx, timeoutFunc := context.WithTimeout(server.ctx, time.Duration(config.Gateway.Framework.Gateway.WaitLoginTimeout)*time.Second)
	defer timeoutFunc()
	req, err = client.Recv(timeoutCtx)
	if err != nil {
		log.Warnw("client recv fail", "session_id", sessionId, "error", err)
		return
	}
	// log.Infow("client=>gate request", "session_id", sessionId, "req_id", req.ReqId)
	if name, _, dataReq, err := server.proto.RequestDecode(req.Payload()); err != nil {
		log.Warnw("client=>gate request decode fail", "session_id", sessionId, "error", err)
	} else if name != "Login" {
		log.Warnw("expeted login request", "name", name, "session_id", sessionId)
	} else {
		// 检验token
		loginReq, ok := dataReq.(gateway.LoginRequest)
		if !ok {
			log.Errorw("login request cast fail", "session_id", sessionId)
			return
		}
		memberId = loginReq.GetMemberId()
		token := loginReq.GetToken()
		log.Infow("client login", "session_id", sessionId, "member_id", memberId, "token", token)
		// 验证memberid和token
		if claims, err := jwt.ParseJwtToken(token, config.Gateway.Module.Jwt.Secret); err != nil {
			log.Errorw("invalid token", "session_id", sessionId, "token", token, "error", err)
			server.loginErrResponse(req, dataReq, err)
		} else if claims.MemberId != memberId {
			log.Errorw("invalid memberid", "token", token, "member_id", memberId, "expected_member_id", claims.MemberId)
			server.loginErrResponse(req, dataReq, errors.ErrInvalidJwt)
		} else {
			memberId = claims.MemberId
			session := newSession(server, sessionId, memberId)
			session.serve(server.ctx, client, req, dataReq)
		}
	}
}

// 选择一个节点
func (server *Server) SelectPeer() *Upstream {
	return server.upstreams.SelectPeer()
}

func (server *Server) OnPeerAdd(peer *gira.Peer) {
	if peer.Name != config.Gateway.Framework.Gateway.Upstream.Name {
		return
	}
	if peer.Id < config.Gateway.Framework.Gateway.Upstream.MinId || peer.Id > config.Gateway.Framework.Gateway.Upstream.MaxId {
		return
	}
	server.upstreams.OnPeerAdd(server.ctx, peer)
}

func (server *Server) OnPeerDelete(peer *gira.Peer) {
	if peer.Name != config.Gateway.Framework.Gateway.Upstream.Name {
		return
	}
	if peer.Id < config.Gateway.Framework.Gateway.Upstream.MinId || peer.Id > config.Gateway.Framework.Gateway.Upstream.MaxId {
		return
	}
	server.upstreams.OnPeerDelete(peer)
}

func (server *Server) OnPeerUpdate(peer *gira.Peer) {
	if peer.Name != config.Gateway.Framework.Gateway.Upstream.Name {
		return
	}
	if peer.Id < config.Gateway.Framework.Gateway.Upstream.MinId || peer.Id > config.Gateway.Framework.Gateway.Upstream.MaxId {
		return
	}
	server.upstreams.OnPeerUpdate(peer)
}
