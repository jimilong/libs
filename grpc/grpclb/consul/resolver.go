package consul

import (
	"fmt"
	"time"

	consul "github.com/hashicorp/consul/api"
	"google.golang.org/grpc/resolver"
)

const schema = "consul"

type consulResolver struct {
	endpoint    string
	cc          resolver.ClientConn
	client      *consul.Client
	lastIndex   uint64
	serviceName string
}

func NewResolver(endpoints string) resolver.Builder {
	return &consulResolver{endpoint: endpoints}
}

func (r *consulResolver) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	var err error
	// generate consul client, return if error
	conf := &consul.Config{
		Scheme:  "http",
		Address: r.endpoint,
	}
	if r.client, err = consul.NewClient(conf); err != nil {
		return nil, fmt.Errorf("wonaming: creat consul error: %v", err)
	}
	r.cc = cc
	r.serviceName = target.Endpoint
	go r.watch()

	return r, err
}

func (r *consulResolver) Scheme() string {
	return schema
}

func (r *consulResolver) ResolveNow(rn resolver.ResolveNowOption) {
}

func (r *consulResolver) Close() {}

func (r *consulResolver) watch() {
	for {
		// watch consul
		addrs, index, err := r.queryConsul(&consul.QueryOptions{WaitIndex: r.lastIndex})
		if err != nil {
			time.Sleep(1 * time.Second)
			//todo log
			continue
		}

		r.lastIndex = index
		r.cc.NewAddress(addrs)
	}
}

// queryConsul is helper function to query consul
func (r *consulResolver) queryConsul(q *consul.QueryOptions) ([]resolver.Address, uint64, error) {
	// query consul
	cs, meta, err := r.client.Health().Service(r.serviceName, "", true, q)
	if err != nil {
		return nil, 0, err
	}

	addrList := make([]resolver.Address, 0, 5)
	for _, s := range cs {
		// addr should like: 127.0.0.1:8001
		addrList = append(addrList, resolver.Address{Addr: fmt.Sprintf("%s:%d", s.Service.Address, s.Service.Port)})
	}

	return addrList, meta.LastIndex, nil
}
