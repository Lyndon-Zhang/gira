package registryclient

///
///
/// key不设置过期时间，程序正常退出时自动清理，非正常退出，要程序重启来解锁
///
/// 注册表结构:
///   /peer_type_user/<<AppName>>/<<UserId>> => <<AppFullName>>
///   /peer_user/<<AppFullName>>/<<UserId>> time
///   /user/<<UserId>> => <<AppFullName>>
///
import (
	"context"
	"fmt"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type player_registry struct {
	userPrefix string // /user/<<UserId>>/       	 	可以根据user_id查找当前所在的服
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func newConfigPlayerRegistry(r *RegistryClient) (*player_registry, error) {
	ctx, cancelFunc := context.WithCancel(r.ctx)
	self := &player_registry{
		userPrefix: "/user/",
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
	return self, nil
}

// 查找玩家位置
func (self *player_registry) WhereIsUser(r *RegistryClient, userId string) (*gira.Peer, error) {
	client := r.client
	userKey := fmt.Sprintf("%s%s", self.userPrefix, userId)
	kv := clientv3.NewKV(client)
	var err error
	getResp, err := kv.Get(self.ctx, userKey)
	if err != nil {
		return nil, err
	}
	if len(getResp.Kvs) <= 0 {
		return nil, errors.ErrUserNotFound
	}
	fullName := string(getResp.Kvs[0].Value)
	peer := r.GetPeer(fullName)
	if peer == nil {
		return nil, errors.ErrPeerNotFound
	} else {
		return peer, nil
	}
}
