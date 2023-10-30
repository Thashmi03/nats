package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/stan.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

const (
	exampleScheme      = "example"
	exampleServiceName = "lb.example.grpc.io"
)

var (
	addr             string
	addrsresolver    []resolver.Address
	grpcResolver     *exampleResolver
	serviceFailCount = make(map[string]int8)
)

func DataReceive(msg *stan.Msg) {
	var data map[string]interface{}
	err := json.Unmarshal(msg.Data, &data)
	if err != nil {
		log.Printf("Error unmarshalling data: %s", err)
		return
	}
	fmt.Println("******************\nReceived a message:")
	fmt.Println(data)

	if app, ok := data["application"].(string); ok {
		if ip, ok := data["InstanceIP"].(string); ok {
			if port, ok := data["InstancePort"].(string); ok {
				addr = ip + port
				addrs := grpcResolver.addrsStore
				found := false
				for _, value := range addrs[app] {
					if value == addr {
						found = true
						break
					}
				}
				if !found {
					addrs[app] = append(addrs[app], addr)
					fmt.Println("mapped", addrs)
					grpcResolver.updateLoadBalancer()
				}
			}
		}
	}
}

func InitializeLoadbalancer() {
	roundrobinConn, err := grpc.Dial(
		fmt.Sprintf("%s:///%s", exampleScheme, exampleServiceName),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer roundrobinConn.Close()
}

func main() {
	// NATS Streaming cluster ID, client ID, and subject
	clusterID := "test-cluster"
	clientID := "test-thashmi"
	const SER_REG_SUB = "ServiceRegistery"

	// Connect to the NATS Streaming server
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL("nats://192.168.1.10:4222"))
	if err != nil {
		log.Fatalf("Error connecting to NATS Streaming: %v", err)
	}
	defer sc.Close()
	InitializeLoadbalancer()


	// Subscribe to the subject
	_, err = sc.Subscribe(SER_REG_SUB, func(msg *stan.Msg) {
		DataReceive(msg)
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject: %v", err)
	}

	healthcheck()
	fmt.Printf("Subscribed to subject %s. Waiting for messages...\n", SER_REG_SUB)

	select {}
}

// resolver
type exampleResolverBuilder struct{}

func (*exampleResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	grpcResolver = &exampleResolver{
		target:     target,
		cc:         cc,
		addrsStore: map[string][]string{},
	}
	fmt.Println("******************\nResolver invoked\nbuild")
	fmt.Println("in build", grpcResolver.addrsStore)
	grpcResolver.updateLoadBalancer()
	return grpcResolver, nil
}
func (*exampleResolverBuilder) Scheme() string { return exampleScheme }

type exampleResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *exampleResolver) updateLoadBalancer() {
	log.Printf("Updating LB")
	for service, instances := range r.addrsStore {
		log.Printf("Service: %s; Instances: %s", service, instances)
		addrsresolver = make([]resolver.Address, len(instances))
		for i, s := range instances {
			addrsresolver[i] = resolver.Address{Addr: s}
		}
		r.cc.UpdateState(resolver.State{Addresses: addrsresolver})
	}
	
}

func (*exampleResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*exampleResolver) Close()                                  {}

func init() {
	resolver.Register(&exampleResolverBuilder{})
}

// healthcheck

func healthcheck() {
	for  {
		for service, instances := range grpcResolver.addrsStore {
			// fmt.Printf("Service: %s; Instances: %s\n", service, instances)
			addrsresolver = make([]resolver.Address, len(instances))
			for _, serviceHost := range instances {
				err := ping(serviceHost)
				if err != nil {
					serviceFailCount[serviceHost] = serviceFailCount[serviceHost] + 1
					log.Printf("Health Check Failed for service, %s for %d time(s)", service, serviceFailCount[serviceHost])
					if serviceFailCount[serviceHost] == 3 {
						removeAddress(service, serviceHost)
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func removeAddress(service, serverAddress string) {
	log.Printf("Remove address : %s from service: %s", serverAddress, service)
	addresses := grpcResolver.addrsStore[service]
	idx := -1
	for i, v := range addresses {
		if v == serverAddress {
			idx = i
			break
		}
	}
	log.Printf("Found the address at %d", idx)
	if idx != -1 {
		addresses = append(addresses[:idx], addresses[idx+1:]...)
		grpcResolver.addrsStore[service] = addresses
		grpcResolver.updateLoadBalancer()
	}
	log.Println("Updated address after remove: ", addresses)
}

func ping(address string) error {
	// conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	// if err != nil {
	// 	return err
	// }
	// defer conn.Close()
	// return nil	
	conn,err:=grpc.Dial(address,grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
