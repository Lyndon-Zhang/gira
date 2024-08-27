package registry

import (
	"fmt"
	"sync"

	"github.com/Lyndon-Zhang/gira"
	"google.golang.org/grpc/resolver"
)

// grpc resolver

const (
	PeerScheme string = "peer"
)

func formatPeerUrl(fullName string) string {
	return fmt.Sprintf("%s:///%s", PeerScheme, fullName)
}

type peer_resolver_builder struct {
	r         *Registry
	resolvers sync.Map
}

type peer_resolver struct {
	cc       resolver.ClientConn
	builder  *peer_resolver_builder
	reg      *Registry
	fullName string
	Address  string
}

func (builder *peer_resolver_builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &peer_resolver{
		cc:       cc,
		reg:      builder.r,
		builder:  builder,
		fullName: target.Endpoint(),
	}

	if last, loaded := builder.resolvers.LoadOrStore(r.fullName, r); loaded {
		r = last.(*peer_resolver)
		addr := resolver.Address{
			Addr: r.Address,
		}
		cc.UpdateState(resolver.State{
			Addresses: []resolver.Address{addr},
		})
		return r, nil
	} else {
		if peer := r.reg.peerRegistry.getPeer(r.reg, r.fullName); peer == nil {
			return r, nil
		} else {
			r.Address = peer.Address
			addr := resolver.Address{
				Addr: r.Address,
			}
			cc.UpdateState(resolver.State{
				Addresses: []resolver.Address{addr},
			})
			return r, nil
		}
	}
}

func (builder *peer_resolver_builder) onPeerUpdate(reg *Registry, peer *gira.Peer) error {
	if last, ok := builder.resolvers.Load(peer.FullName); !ok {
		return nil
	} else {
		peerResolver := last.(*peer_resolver)
		return peerResolver.onPeerUpdate(reg, peer)
	}
}

func (builder *peer_resolver_builder) Scheme() string {
	return PeerScheme
}

func (r *peer_resolver) ResolveNow(o resolver.ResolveNowOptions) {
}

func (r *peer_resolver) Close() {
	r.builder.resolvers.Delete(r.fullName)
}

func (r *peer_resolver) onPeerUpdate(reg *Registry, peer *gira.Peer) error {
	r.Address = peer.Address
	addr := resolver.Address{
		Addr: r.Address,
	}
	r.cc.UpdateState(resolver.State{
		Addresses: []resolver.Address{addr},
	})
	return nil
}
