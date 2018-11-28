package consul

import (
	"fmt"
	"log"
	"time"

	consul "github.com/hashicorp/consul/api"
)

var (
	client    *consul.Client
	serviceID string
	err       error
)

// Register is the helper function to self-register service into Etcd/Consul server
// name - service name
// host - service host
// port - service port
// endpoint - consul dial address, for example: "127.0.0.1:8500"
// interval - interval of self-register to consul
// ttl - ttl of the register information
func Register(name string, host string, port int, endpoint string, interval time.Duration, ttl int) error {
	conf := &consul.Config{Scheme: "http", Address: endpoint}
	if client, err = consul.NewClient(conf); err != nil {
		return fmt.Errorf("wonaming: create consul client error: %v", err)
	}

	serviceID = fmt.Sprintf("%s-%s-%d", name, host, port)

	// routine to update ttl
	go keepAlive(interval)

	// initial register service
	if err = client.Agent().ServiceRegister(&consul.AgentServiceRegistration{
		ID:      serviceID,
		Name:    name,
		Address: host,
		Port:    port,
		Check: &consul.AgentServiceCheck{
			TTL:    fmt.Sprintf("%ds", ttl),
			Status: "passing",
		},
	}); err != nil {
		return fmt.Errorf("wonaming: initial register service '%s' host to consul error: %s", name, err.Error())
	}

	return nil
}

func keepAlive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		err = client.Agent().UpdateTTL(serviceID, "ok", "passing")
		if err != nil {
			log.Println("wonaming: update ttl of service error: ", err.Error())
		}
	}
}

func Unregister() {
	if err = client.Agent().ServiceDeregister(serviceID); err != nil {
		log.Println("wonaming: deregister service error: ", err.Error())
	} else {
		log.Println("wonaming: deregistered service from consul server.")
	}

	if err = client.Agent().CheckDeregister(serviceID); err != nil {
		log.Println("wonaming: deregister check error: ", err.Error())
	}
}
