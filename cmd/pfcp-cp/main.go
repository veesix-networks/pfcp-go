package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/veesix-networks/pfcp-go/api/pfcp/v1"
	"github.com/veesix-networks/pfcp-go/pkg/cp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	nodeID := flag.String("node-id", "cp-node-1", "PFCP Control Plane node ID")
	listenAddr := flag.String("listen-addr", ":8805", "Listen address")
	grpcAddr := flag.String("grpc-addr", ":50051", "gRPC API address")
	heartbeatInterval := flag.Duration("heartbeat-interval", 60*time.Second, "Heartbeat interval")
	retransmitN1 := flag.Int("retransmit-n1", 3, "Max retransmission attempts")
	retransmitT1 := flag.Duration("retransmit-t1", 3*time.Second, "Retransmission timeout")

	flag.Parse()

	log.Printf("Starting PFCP Control Plane Function")
	log.Printf("  Node ID: %s", *nodeID)
	log.Printf("  Listen Address: %s", *listenAddr)
	log.Printf("  gRPC Address: %s", *grpcAddr)
	log.Printf("  Heartbeat Interval: %s", *heartbeatInterval)
	log.Printf("  Retransmit N1: %d", *retransmitN1)
	log.Printf("  Retransmit T1: %s", *retransmitT1)

	store := cp.NewMemoryStore()

	cpCfg := &cp.Config{
		NodeID:            *nodeID,
		ListenAddr:        *listenAddr,
		HeartbeatInterval: *heartbeatInterval,
		RetransmitN1:      *retransmitN1,
		RetransmitT1:      *retransmitT1,
	}

	cpFunc, err := cp.NewCPFunction(cpCfg, store)
	if err != nil {
		log.Fatalf("Failed to create CP function: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := cpFunc.Start(ctx); err != nil {
			log.Printf("CP function error: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC address: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterControlPlaneServer(grpcServer, cp.NewGRPCServer(cpFunc))
	reflection.Register(grpcServer)

	go func() {
		log.Printf("gRPC server listening on %s", *grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	log.Println("PFCP Control Plane ready")

	<-sigCh
	log.Println("Shutting down...")
	grpcServer.GracefulStop()
	cancel()
	time.Sleep(1 * time.Second)
}
