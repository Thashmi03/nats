package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	// "runtime"
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
	addrs         = make(map[string][]string)
	addr          string
	count         int64
	addrsresolver []resolver.Address
	value         []string
	grpcResolver  *exampleResolver
	grpcaddress   = make(map[string][]string)
    app1 string
    flag bool
)

//nats

func DataReceive(msg *stan.Msg) {
	var data map[string]interface{}
	err := json.Unmarshal(msg.Data, &data)
	if err != nil {
		log.Printf("Error unmarshalling data: %s", err)
		return
	}
	// Now you can work with the map data
	fmt.Println("******************\nReceived a message:")
	for i := 1; i <= 1; i++ {
        var ok bool
		if app,_ := data["application"].(string);ok {
			if ip, ok := data["InstanceIP"].(string); ok {
				if port, ok := data["InstancePort"].(string); ok {
					addr = ip + port
					addrs[app] = append(addrs[app], addr)
					value, _ = addrs[app]
                    app1 = app
                  
				}
			}
		}
		fmt.Println(data)
		fmt.Println("mapped", addrs)
	}
}
func ResolverCalling(){
    grpcResolver = &exampleResolver{
		addrsStore: map[string][]string{
            exampleServiceName:value,
			app1: value,
		},
	}
	grpcResolver.start()
}
func RoundRobin() {
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
	clusterID := "test-cluster"            // Replace with your NATS Streaming cluster ID.
	clientID := "test-publish"             // Replace with your NATS Streaming client ID.
	const SER_REG_SUB = "ServiceRegistery" // Replace with the subject you want to subscribe to.

	// Connect to the NATS Streaming server
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL("nats://192.168.1.31:4222"))
	if err != nil {
		log.Fatalf("Error connecting to NATS Streaming: %v", err)
	}
	defer sc.Close()

	// Subscribe to the subject
	_, err = sc.Subscribe(SER_REG_SUB, func(msg *stan.Msg) {
		DataReceive(msg)
		if !flag{
            RoundRobin()
            flag=true
        }else{
            ResolverCalling()
        }
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject: %v", err)
	}
	fmt.Printf("Subscribed to subject %s. Waiting for messages...\n", SER_REG_SUB)
	select {}
}

// resolver
type exampleResolverBuilder struct{}

func (*exampleResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {

	grpcResolver = &exampleResolver{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
            exampleServiceName:value,
			// app1: value,
		},
	}
	fmt.Println("******************\nResolver invoked\nbuild")
	fmt.Println("in build", grpcResolver.addrsStore)
	grpcResolver.start()
	return grpcResolver, nil
}
func (*exampleResolverBuilder) Scheme() string { return exampleScheme }

type exampleResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *exampleResolver) start() {
	fmt.Println("start")
	addrStrs := r.addrsStore[r.target.Endpoint()]
	fmt.Println("addstrs", addrStrs)
	addrsresolver = make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrsresolver[i] = resolver.Address{Addr: s}
	}

	var extractedAddrs []string
	for _, s := range addrStrs {
		_, port, err := net.SplitHostPort(s)
		if err != nil {
			log.Printf("Error extracting IP and port: %v", err)
			continue
		}
		// Check if the port is empty
		if port == "" {
			log.Printf("Missing port in address: %s", s)
			continue
		}
		extractedAddrs = append(extractedAddrs, s)
	}
	fmt.Println(extractedAddrs)
	r.cc.UpdateState(resolver.State{Addresses: addrsresolver})
	go func() {
		go healthCheck(extractedAddrs)
	}()
}

func (*exampleResolver) ResolveNow(o resolver.ResolveNowOptions) {

}
func (*exampleResolver) Close() {}

func init() {
	resolver.Register(&exampleResolverBuilder{})
}

// healthcheck
func healthCheck(addrs []string) {
	fmt.Println("******************\nHeathcheck invoked")
	for i := 1; i <= 3; i++ {
		for _, addr := range addrs {
			err := ping(addr)
			if err != nil {
				log.Printf("Server at %s is down: %s", addr, err)
				count++
				if count == 3 {
					addrs = removeAddress(addr, addrs)
					fmt.Println(addrs)
					count = 0 // reset count
				}
			} else {
				log.Printf("Server at %s is up and running.", addr)
			}
			time.Sleep(10 * time.Second)
		}
	}
}

func removeAddress(target string, list []string) []string {
	for i, v := range list {
		if v == target {
			list[i] = list[len(list)-1]
			fmt.Println(list)
			fmt.Println("address removed")
			return list[:len(list)-1]
		}
	}
	return list
}

func ping(address string) error {
	fmt.Println("Ping invoked")
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}
