package main

import (
	"encoding/json"
	"fmt"
	"log"
	"user/constant"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

func main() {
	// NATS Streaming cluster ID and client ID
	clusterID := "test-cluster" // Replace with your NATS Streaming cluster ID.
	clientID := "test-thash"  // Replace with your NATS Streaming client ID.

	// Connect to the NATS Streaming server
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	// Connect to the NATS Streaming server
	sc, err := stan.Connect(clusterID, clientID, stan.NatsConn(nc))
	if err != nil {
		log.Fatalf("Error connecting to NATS Streaming: %v", err)
	}
	defer sc.Close()

	// Define the subject you want to send the message to.
	subject := "ServiceRegistery" // Replace with the desired subject.

	// Define the message you want to send.
	m:=map[string]string{
		"application":"TestServer1",
		"InstanceIP":"192.168.1.31",
		"InstancePort":constant.Port1,
	}
	// m:=map[string]string{
	// 	"application":"TestServer2",
	// 	"InstanceIP":"192.168.1.31",
	// 	"InstancePort":constant.Port2,
	// }
	jsonData, err := json.Marshal(m)
	if err != nil {
		fmt.Println("Error marshaling data to JSON:", err)
		return
	}
	message := []byte(jsonData)

	// Publish the message to the specified subject.
	err = sc.Publish(subject, message)
	if err != nil {
		log.Fatalf("Error publishing message: %v", err)
	}

	fmt.Printf("Message sent to subject %s\n", subject)
}
