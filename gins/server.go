package gins

/// 参考 https://juejin.cn/post/6844903833273892871

import (
	"context"
	"net/http"
	"time"

	"github.com/Lyndon-Zhang/gira"
	log "github.com/Lyndon-Zhang/gira/corelog"
)

type HttpServer struct {
	config     gira.HttpConfig
	Handler    http.Handler
	server     *http.Server
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewConfigHttpServer(ctx context.Context, config gira.HttpConfig, router http.Handler) (*HttpServer, error) {
	s := &HttpServer{
		config:  config,
		Handler: router,
	}
	s.ctx, s.cancelFunc = context.WithCancel(ctx)
	server := &http.Server{
		Addr:           s.config.Addr,
		Handler:        s.Handler,
		ReadTimeout:    time.Duration(s.config.ReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(s.config.WriteTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.server = server
	return s, nil
}

func (self *HttpServer) Stop() error {
	log.Debugw("http server on stop")
	self.cancelFunc()
	return nil
}

func (self *HttpServer) Serve() error {
	log.Debugw("http server started", "addr", self.config.Addr)
	go func() {
		<-self.ctx.Done()
		self.server.Close()
	}()
	var err error
	if self.config.Ssl && len(self.config.CertFile) > 0 && len(self.config.KeyFile) > 0 {
		if err = self.server.ListenAndServeTLS(self.config.CertFile, self.config.KeyFile); err == http.ErrServerClosed {
			err = nil
		}
	} else {
		if err = self.server.ListenAndServe(); err == http.ErrServerClosed {
			err = nil
		}
	}
	return err
}
