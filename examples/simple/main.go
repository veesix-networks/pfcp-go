package main

import (
	"context"
	"log"
	"time"

	"github.com/veesix-networks/pfcp-go/pkg/cp"
	"github.com/veesix-networks/pfcp-go/pkg/dataplane/mock"
	"github.com/veesix-networks/pfcp-go/pkg/protocol"
	"github.com/veesix-networks/pfcp-go/pkg/up"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := cp.NewMemoryStore()

	cpCfg := &cp.Config{
		NodeID:            "cp-node-1",
		ListenAddr:        ":8805",
		HeartbeatInterval: 60 * time.Second,
		RetransmitN1:      3,
		RetransmitT1:      3 * time.Second,
	}

	cpFunc, err := cp.NewCPFunction(cpCfg, store)
	if err != nil {
		log.Fatalf("Failed to create CP function: %v", err)
	}

	go func() {
		if err := cpFunc.Start(ctx); err != nil {
			log.Printf("CP function error: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)

	mockDP := mock.NewMockDataplane()

	upCfg := &up.Config{
		NodeID:            "up-node-1",
		CPAddress:         "127.0.0.1:8805",
		LocalAddr:         ":8806",
		HeartbeatInterval: 60 * time.Second,
	}

	upFunc, err := up.NewUPFunction(upCfg, mockDP)
	if err != nil {
		log.Fatalf("Failed to create UP function: %v", err)
	}

	go func() {
		if err := upFunc.Start(ctx); err != nil {
			log.Printf("UP function error: %v", err)
		}
	}()

	time.Sleep(2 * time.Second)

	log.Println("Creating PFCP session for DHCP punt rule...")

	pdrs := []*cp.PDR{
		{
			ID:         1,
			Precedence: 1000,
			PDI: &cp.PacketDetectionInfo{
				SourceInterface: protocol.SourceInterfaceAccess,
				SDFFilter:       "permit in udp from any to any 67-68",
			},
			FAR_ID: 1,
		},
	}

	fars := []*cp.FAR{
		{
			ID:          1,
			ApplyAction: protocol.ApplyActionForward | protocol.ApplyActionNotify,
			ForwardingParameters: &cp.ForwardingParams{
				DestinationInterface: protocol.DestinationInterfaceCPFunction,
			},
		},
	}

	seid, err := cpFunc.CreateSession("up-node-1", pdrs, fars, nil, nil)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	log.Printf("PFCP session created successfully: SEID=%d", seid)

	pdrCount, farCount, _, _, err := mockDP.GetSessionRules(seid)
	if err != nil {
		log.Fatalf("Failed to get session rules: %v", err)
	}

	log.Printf("Session rules installed: %d PDRs, %d FARs", pdrCount, farCount)

	time.Sleep(2 * time.Second)

	log.Println("Deleting PFCP session...")
	if err := cpFunc.DeleteSession(seid); err != nil {
		log.Fatalf("Failed to delete session: %v", err)
	}

	log.Println("PFCP session deleted successfully")

	time.Sleep(1 * time.Second)

	log.Println("Shutting down...")
	cancel()
	time.Sleep(1 * time.Second)
}
