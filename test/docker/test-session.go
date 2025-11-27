package main

import (
	"context"
	"log"
	"time"

	"github.com/veesix-networks/pfcp-go/pkg/cp"
	"github.com/veesix-networks/pfcp-go/pkg/protocol"
)

func main() {
	log.Println("Connecting to PFCP Control Plane...")

	store := cp.NewMemoryStore()

	cpCfg := &cp.Config{
		NodeID:            "test-client",
		ListenAddr:        ":8806",
		HeartbeatInterval: 60 * time.Second,
		RetransmitN1:      3,
		RetransmitT1:      3 * time.Second,
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

	time.Sleep(2 * time.Second)

	log.Println("Creating PFCP session with DHCP punt rule...")

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

	log.Printf("SUCCESS: PFCP session created with SEID=%d", seid)
	log.Printf("  - Installed 1 PDR (DHCP traffic detection)")
	log.Printf("  - Installed 1 FAR (punt to control plane)")

	time.Sleep(5 * time.Second)

	log.Println("Deleting PFCP session...")
	if err := cpFunc.DeleteSession(seid); err != nil {
		log.Fatalf("Failed to delete session: %v", err)
	}

	log.Println("SUCCESS: PFCP session deleted")
}
