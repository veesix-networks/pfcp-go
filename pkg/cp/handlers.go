package cp

import (
	"fmt"
	"net"
	"time"

	"github.com/veesix-networks/pfcp-go/pkg/protocol"
)

func (cp *CPFunction) handleAssociationSetupRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	nodeIDIE := msg.FindIE(protocol.IETypeNodeID)
	if nodeIDIE == nil {
		return fmt.Errorf("no node ID in association setup request")
	}

	recoveryTSIE := msg.FindIE(protocol.IETypeRecoveryTimeStamp)
	if recoveryTSIE == nil {
		return fmt.Errorf("no recovery timestamp in association setup request")
	}

	nodeID := string(nodeIDIE.Value[1:])

	assoc := &Association{
		NodeID:        nodeIDIE.Value[1:],
		RemoteAddr:    addr,
		RecoveryTS:    0,
		EstablishedAt: time.Now(),
	}

	cp.mu.Lock()
	cp.associations[nodeID] = assoc
	cp.mu.Unlock()

	fmt.Printf("Association established from UP node: %s (%s)\n", nodeID, addr)

	resp := protocol.NewAssociationSetupResponse(
		msg.Header.SequenceNumber,
		cp.nodeID,
		protocol.CauseRequestAccepted,
		cp.recoveryTS,
	)

	return cp.transport.SendResponse(resp, addr)
}

func (cp *CPFunction) handleAssociationReleaseRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	nodeIDIE := msg.FindIE(protocol.IETypeNodeID)
	if nodeIDIE == nil {
		return fmt.Errorf("no node ID in association release request")
	}

	nodeID := string(nodeIDIE.Value[1:])

	cp.mu.Lock()
	delete(cp.associations, nodeID)
	cp.mu.Unlock()

	resp := protocol.NewAssociationReleaseResponse(
		msg.Header.SequenceNumber,
		cp.nodeID,
		protocol.CauseRequestAccepted,
	)

	return cp.transport.SendResponse(resp, addr)
}

func (cp *CPFunction) handleHeartbeatRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	resp := protocol.NewHeartbeatResponse(msg.Header.SequenceNumber, cp.recoveryTS)
	return cp.transport.SendResponse(resp, addr)
}

func (cp *CPFunction) handleSessionReportRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	return nil
}
