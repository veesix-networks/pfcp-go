package up

import (
	"net"
	"time"

	"github.com/veesix-networks/pfcp-go/pkg/protocol"
)

func (up *UPFunction) handleSessionEstablishmentRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	seid := up.allocSEID()

	session := &Session{
		LocalSEID:  seid,
		RemoteSEID: msg.Header.SEID,
		PDRs:       make(map[uint16]*PDR),
		FARs:       make(map[uint32]*FAR),
		QERs:       make(map[uint32]*QER),
		URRs:       make(map[uint32]*URR),
		CreatedAt:  time.Now(),
	}

	createPDRs := msg.FindAllIEs(protocol.IETypeCreatePDR)
	for _, createPDR := range createPDRs {
		pdrIEs, err := protocol.ParseGroupedIE(createPDR.Value)
		if err != nil {
			continue
		}

		var pdr PDR
		pdr.PDI = &PDI{}

		for _, ie := range pdrIEs {
			switch ie.Type {
			case protocol.IETypePDR_ID:
				id, _ := ie.GetPDR_ID()
				pdr.ID = id
			case protocol.IETypePrecedence:
				pdr.Precedence = 0
			case protocol.IETypeFAR_ID:
				farID, _ := ie.GetFAR_ID()
				pdr.FAR_ID = farID
			case protocol.IETypePDI:
				pdiIEs, _ := protocol.ParseGroupedIE(ie.Value)
				for _, pdiIE := range pdiIEs {
					switch pdiIE.Type {
					case protocol.IETypeSourceInterface:
						if len(pdiIE.Value) > 0 {
							pdr.PDI.SourceInterface = pdiIE.Value[0]
						}
					case protocol.IETypeSDFFilter:
						pdr.PDI.SDFFilter = pdiIE.Value
					case protocol.IETypeUE_IPAddress:
						pdr.PDI.UE_IPAddress = string(pdiIE.Value)
					case protocol.IETypeNetworkInstance:
						pdr.PDI.NetworkInstance = string(pdiIE.Value)
					case protocol.IETypeApplicationID:
						pdr.PDI.ApplicationID = string(pdiIE.Value)
					}
				}
			}
		}

		session.PDRs[pdr.ID] = &pdr
		up.dataplane.InstallPDR(seid, &pdr)
	}

	createFARs := msg.FindAllIEs(protocol.IETypeCreateFAR)
	for _, createFAR := range createFARs {
		farIEs, err := protocol.ParseGroupedIE(createFAR.Value)
		if err != nil {
			continue
		}

		var far FAR
		for _, ie := range farIEs {
			switch ie.Type {
			case protocol.IETypeFAR_ID:
				farID, _ := ie.GetFAR_ID()
				far.ID = farID
			case protocol.IETypeApplyAction:
				if len(ie.Value) > 0 {
					far.ApplyAction = ie.Value[0]
				}
			}
		}

		session.FARs[far.ID] = &far
		up.dataplane.InstallFAR(seid, &far)
	}

	up.mu.Lock()
	up.sessions[seid] = session
	up.mu.Unlock()

	resp := protocol.NewSessionEstablishmentResponse(
		msg.Header.SequenceNumber,
		msg.Header.SEID,
		protocol.CauseRequestAccepted,
		seid,
	)

	return up.transport.SendResponse(resp, addr)
}

func (up *UPFunction) handleSessionModificationRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	resp := &protocol.Message{
		Header: protocol.MessageHeader{
			Version:        protocol.Version1,
			MessageType:    protocol.MsgTypeSessionModificationResponse,
			SEIDPresent:    true,
			SEID:           msg.Header.SEID,
			SequenceNumber: msg.Header.SequenceNumber,
		},
		IEs: []*protocol.IE{
			protocol.NewCauseIE(protocol.CauseRequestAccepted),
		},
	}

	return up.transport.SendResponse(resp, addr)
}

func (up *UPFunction) handleSessionDeletionRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	seid := msg.Header.SEID

	up.mu.Lock()
	delete(up.sessions, seid)
	up.mu.Unlock()

	up.dataplane.DeleteSession(seid)

	resp := protocol.NewSessionDeletionResponse(
		msg.Header.SequenceNumber,
		msg.Header.SEID,
		protocol.CauseRequestAccepted,
	)

	return up.transport.SendResponse(resp, addr)
}

func (up *UPFunction) handleHeartbeatRequest(msg *protocol.Message, addr *net.UDPAddr) error {
	resp := protocol.NewHeartbeatResponse(msg.Header.SequenceNumber, up.recoveryTS)
	return up.transport.SendResponse(resp, addr)
}
