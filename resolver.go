package dnscache

import (
	"context"
	"math/rand"
	"net"
	"sync"

	"github.com/hashicorp/go-multierror"
)

var randPerm = rand.Perm

func DialContextFunc(r *Resolver, dialFn func(context.Context, string, string) (net.Conn, error)) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := r.LookupIP(ctx, host)
		if err != nil {
			return nil, err
		}
		var result error
		for _, i := range randPerm(len(ips)) {
			conn, err := dialFn(ctx, network, net.JoinHostPort(ips[i].String(), port))
			if err == nil {
				return conn, nil
			}
			result = multierror.Append(result, err)
		}
		return nil, result
	}
}

func lookupIP(ctx context.Context, host string) ([]net.IP, error) {
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, len(addrs))
	for i, ia := range addrs {
		ips[i] = ia.IP
	}
	return ips, nil
}

type Resolver struct {
	lookupIPFn func(ctx context.Context, host string) ([]net.IP, error)
	cache      sync.Map // map[string][]net.IP
	once       sync.Once
}

func (r *Resolver) init() {
	if r.lookupIPFn == nil {
		r.lookupIPFn = lookupIP
	}
}

func (r *Resolver) setIP(host string, ips []net.IP) {
	r.cache.Store(host, ips)
}

func (r *Resolver) getIP(host string) ([]net.IP, bool) {
	if v, ok := r.cache.Load(host); ok {
		return v.([]net.IP), true
	}
	return nil, false
}

func (r *Resolver) LookupIP(ctx context.Context, host string) ([]net.IP, error) {
	r.once.Do(r.init)
	if ips, ok := r.getIP(host); ok {
		return ips, nil
	}
	ips, err := r.lookupIPFn(ctx, host)
	if err != nil {
		return nil, err
	}
	r.setIP(host, ips)
	return ips, nil
}

func (r *Resolver) Reflesh() error {
	r.once.Do(r.init)
	var result error
	r.cache.Range(func(key, value interface{}) bool {
		host := key.(string)
		ips, err := r.lookupIPFn(context.Background(), host)
		if err != nil {
			result = multierror.Append(result, err)
		} else {
			r.setIP(host, ips)
		}
		return true
	})
	return result
}
