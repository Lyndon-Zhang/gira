package registryclient

///
///
///
/// 注册表结构:
///   /unique_service/<<ServiceName>> => <<AppFullName>>
///   /service/<<ServiceGroupName>>/<<ServiceName>> => <<AppFullName>>
///   /peer_service/<<AppFullName>>/<<ServiceUniqueName>> => <<AppFullName>>
///
import (
	"context"
	"fmt"
	"strings"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/options/service_options"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type service_registry struct {
	servicePrefix string // /service/<<ServiceName>>/      							可以根据服务名查找当前所在的服
	ctx           context.Context
	cancelFunc    context.CancelFunc
}

func newConfigServiceRegistry(r *RegistryClient) (*service_registry, error) {
	ctx, cancelFunc := context.WithCancel(r.ctx)
	self := &service_registry{
		servicePrefix: "/service/",
		ctx:           ctx,
		cancelFunc:    cancelFunc,
	}
	return self, nil
}

func (self *service_registry) NewServiceName(r *RegistryClient, serviceName string, opt ...service_options.RegisterOption) string {
	opts := service_options.RegisterOptions{}
	for _, v := range opt {
		v.ConfigRegisterOption(&opts)
	}
	if opts.AsAppService {
		serviceName = fmt.Sprintf("%s/%d", serviceName, r.appId)
	}
	return serviceName
}

// 查找服务位置
func (self *service_registry) WhereIsService(r *RegistryClient, serviceName string, opt ...service_options.WhereOption) (peers []*gira.Peer, err error) {
	opts := service_options.WhereOptions{}
	for _, v := range opt {
		v.ConfigWhereOption(&opts)
	}
	if opts.Catalog || opts.Prefix {
		var getOpts []clientv3.OpOption
		getOpts = append(getOpts, clientv3.WithPrefix())
		client := r.client
		kv := clientv3.NewKV(client)
		var getResp *clientv3.GetResponse
		key := fmt.Sprintf("%s%s", self.servicePrefix, serviceName)
		if getResp, err = kv.Get(self.ctx, key, getOpts...); err != nil {
			return
		}
		multicastCount := opts.MaxCount
		for _, kv := range getResp.Kvs {
			peer := r.GetPeer(string(kv.Value))
			if peer != nil {
				peers = append(peers, peer)
				// 多播指定数量
				if multicastCount > 0 {
					multicastCount--
					if multicastCount <= 0 {
						break
					}
				}
			}
		}
		return
	} else {
		var getOpts []clientv3.OpOption
		client := r.client
		kv := clientv3.NewKV(client)
		var getResp *clientv3.GetResponse
		key := fmt.Sprintf("%s%s", self.servicePrefix, serviceName)
		if getResp, err = kv.Get(self.ctx, key, getOpts...); err != nil {
			return
		}
		for _, kv := range getResp.Kvs {
			peer := r.GetPeer(string(kv.Value))
			if peer != nil {
				peers = append(peers, peer)
			}
		}
		return
	}
}

// 列出全部的服务
func (self *service_registry) ListServiceKvs(r *RegistryClient) (kvs map[string][]string, err error) {
	client := r.client
	kv := clientv3.NewKV(client)
	var getResp *clientv3.GetResponse
	if getResp, err = kv.Get(self.ctx, self.servicePrefix, clientv3.WithPrefix()); err != nil {
		return
	}
	kvs = make(map[string][]string)
	for _, kv := range getResp.Kvs {
		words := strings.Split(string(kv.Key), "/")
		var serviceFullName string
		if len(words) == 4 {
			serviceFullName = fmt.Sprintf("%s/%s", words[2], words[3])
		} else if len(words) == 3 {
			serviceFullName = words[2]
		}
		peerFullName := string(kv.Value)
		kvs[peerFullName] = append(kvs[peerFullName], serviceFullName)
	}
	return
}
