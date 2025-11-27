package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/veesix-networks/pfcp-go/pkg/dataplane/mock"
	"github.com/veesix-networks/pfcp-go/pkg/dataplane/vpp"
	"github.com/veesix-networks/pfcp-go/pkg/up"
)

func main() {
	nodeID := flag.String("node-id", "up-node-1", "PFCP User Plane node ID")
	cpAddress := flag.String("cp-address", "127.0.0.1:8805", "Control Plane address")
	localAddr := flag.String("local-addr", ":8805", "Local listen address")
	heartbeatInterval := flag.Duration("heartbeat-interval", 60*time.Second, "Heartbeat interval")
	dataplaneType := flag.String("dataplane", "vpp", "Dataplane type (mock or vpp)")
	vppSocket := flag.String("vpp-socket", "/run/vpp/api.sock", "VPP API socket path")

	flag.Parse()

	log.Printf("Starting PFCP User Plane Function")
	log.Printf("  Node ID: %s", *nodeID)
	log.Printf("  CP Address: %s", *cpAddress)
	log.Printf("  Local Address: %s", *localAddr)
	log.Printf("  Heartbeat Interval: %s", *heartbeatInterval)
	log.Printf("  Dataplane: %s", *dataplaneType)

	var dp up.Dataplane
	var err error

	switch *dataplaneType {
	case "vpp":
		log.Printf("  VPP Socket: %s", *vppSocket)
		dp, err = vpp.NewVPPDataplane(*vppSocket)
		if err != nil {
			log.Fatalf("Failed to create VPP dataplane: %v", err)
		}
		log.Println("VPP dataplane initialized")
	case "mock":
		dp = mock.NewMockDataplane()
		log.Println("Mock dataplane initialized")
	default:
		log.Fatalf("Unknown dataplane type: %s", *dataplaneType)
	}

	upCfg := &up.Config{
		NodeID:            *nodeID,
		CPAddress:         *cpAddress,
		LocalAddr:         *localAddr,
		HeartbeatInterval: *heartbeatInterval,
	}

	upFunc, err := up.NewUPFunction(upCfg, dp)
	if err != nil {
		log.Fatalf("Failed to create UP function: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := upFunc.Start(ctx); err != nil {
			log.Printf("UP function error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	log.Println("PFCP User Plane ready")

	<-sigCh
	log.Println("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
