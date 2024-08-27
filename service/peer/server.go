package peer

import (
	"context"
	"runtime"

	"github.com/Lyndon-Zhang/gira/facade"
	"github.com/Lyndon-Zhang/gira/options/service_options"
	"github.com/Lyndon-Zhang/gira/service/peer/peerpb"
)

type PeerService struct {
	ctx        context.Context
	peerServer *peer_server
}

type peer_server struct {
	peerpb.UnimplementedPeerServer
}

func (self *peer_server) HealthCheck(context.Context, *peerpb.HealthCheckRequest) (*peerpb.HealthCheckResponse, error) {
	resp := &peerpb.HealthCheckResponse{
		BuildTime:     facade.GetBuildTime(),
		AppVersion:    facade.GetAppVersion(),
		UpTime:        facade.GetUpTime(),
		ResVersion:    facade.GetResVersion(),
		LoaderVersion: facade.GetLoaderVersion(),
	}
	return resp, nil
}

func (self *peer_server) MemStats(context.Context, *peerpb.MemStatsRequest) (*peerpb.MemStatsResponse, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	resp := &peerpb.MemStatsResponse{
		Alloc:        memStats.Alloc,
		TotalAlloc:   memStats.TotalAlloc,
		Sys:          memStats.Sys,
		Mallocs:      memStats.Mallocs,
		Frees:        memStats.Frees,
		HeapAlloc:    memStats.HeapAlloc,
		HeapSys:      memStats.HeapSys,
		HeapIdle:     memStats.HeapIdle,
		HeapInuse:    memStats.HeapInuse,
		HeapReleased: memStats.HeapReleased,
		HeapObjects:  memStats.HeapObjects,
	}
	return resp, nil
}
func NewService() *PeerService {
	return &PeerService{
		peerServer: &peer_server{},
	}
}

func (self *PeerService) Serve() error {
	<-self.ctx.Done()
	return nil
}

func (self *PeerService) OnStop() error {
	return nil
}

func (self *PeerService) OnStart(ctx context.Context) error {
	self.ctx = ctx
	if _, err := facade.RegisterServiceName(GetServiceName()); err != nil {
		return err
	}
	peerpb.RegisterPeerServer(facade.GrpcServer(), self.peerServer)
	return nil
}

func GetServiceName() string {
	return facade.NewServiceName(peerpb.PeerServerName, service_options.WithAsAppServiceOption())
}
