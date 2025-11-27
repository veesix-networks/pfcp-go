package protocol

import "encoding/binary"

type Message struct {
	Header MessageHeader
	IEs    []*IE
}

func (m *Message) Marshal() ([]byte, error) {
	headerBuf, err := m.Header.Marshal()
	if err != nil {
		return nil, err
	}

	var iesBuf []byte
	for _, ie := range m.IEs {
		ieBuf, err := ie.Marshal()
		if err != nil {
			return nil, err
		}
		iesBuf = append(iesBuf, ieBuf...)
	}

	m.Header.MessageLength = uint16(len(iesBuf))
	headerBuf, err = m.Header.Marshal()
	if err != nil {
		return nil, err
	}

	return append(headerBuf, iesBuf...), nil
}

func (m *Message) Unmarshal(data []byte) error {
	if err := m.Header.Unmarshal(data); err != nil {
		return err
	}

	headerLen := m.Header.Len()
	iesData := data[headerLen : headerLen+int(m.Header.MessageLength)]

	offset := 0
	for offset < len(iesData) {
		ie := &IE{}
		n, err := ie.Unmarshal(iesData[offset:])
		if err != nil {
			return err
		}
		m.IEs = append(m.IEs, ie)
		offset += n
	}

	return nil
}

func (m *Message) FindIE(ieType uint16) *IE {
	for _, ie := range m.IEs {
		if ie.Type == ieType {
			return ie
		}
	}
	return nil
}

func (m *Message) FindAllIEs(ieType uint16) []*IE {
	var result []*IE
	for _, ie := range m.IEs {
		if ie.Type == ieType {
			result = append(result, ie)
		}
	}
	return result
}

func NewHeartbeatRequest(seqNum uint32, recoveryTS uint32) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeHeartbeatRequest,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewRecoveryTimeStampIE(recoveryTS),
		},
	}
}

func NewHeartbeatResponse(seqNum uint32, recoveryTS uint32) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeHeartbeatResponse,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewRecoveryTimeStampIE(recoveryTS),
		},
	}
}

func NewAssociationSetupRequest(seqNum uint32, nodeID []byte, recoveryTS uint32) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeAssociationSetupRequest,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewNodeIDIE(nodeID),
			NewRecoveryTimeStampIE(recoveryTS),
		},
	}
}

func NewAssociationSetupResponse(seqNum uint32, nodeID []byte, cause uint8, recoveryTS uint32) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeAssociationSetupResponse,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewNodeIDIE(nodeID),
			NewCauseIE(cause),
			NewRecoveryTimeStampIE(recoveryTS),
		},
	}
}

func NewAssociationReleaseRequest(seqNum uint32, nodeID []byte) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeAssociationReleaseRequest,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewNodeIDIE(nodeID),
		},
	}
}

func NewAssociationReleaseResponse(seqNum uint32, nodeID []byte, cause uint8) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeAssociationReleaseResponse,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewNodeIDIE(nodeID),
			NewCauseIE(cause),
		},
	}
}

func NewSessionEstablishmentRequest(seqNum uint32, seid uint64, createPDRs, createFARs, createQERs, createURRs []*IE) *Message {
	ies := make([]*IE, 0, len(createPDRs)+len(createFARs)+len(createQERs)+len(createURRs))
	ies = append(ies, createPDRs...)
	ies = append(ies, createFARs...)
	ies = append(ies, createQERs...)
	ies = append(ies, createURRs...)

	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeSessionEstablishmentRequest,
			SEIDPresent:    true,
			SEID:           seid,
			SequenceNumber: seqNum,
		},
		IEs: ies,
	}
}

func NewSessionEstablishmentResponse(seqNum uint32, seid uint64, cause uint8, localSEID uint64) *Message {
	fseidValue := make([]byte, 9)
	fseidValue[0] = 0x02
	binary.BigEndian.PutUint64(fseidValue[1:], localSEID)

	fseidIE := &IE{
		Type:  IETypePDI,
		Value: fseidValue,
	}

	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeSessionEstablishmentResponse,
			SEIDPresent:    true,
			SEID:           seid,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewCauseIE(cause),
			fseidIE,
		},
	}
}

func NewSessionDeletionRequest(seqNum uint32, seid uint64) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeSessionDeletionRequest,
			SEIDPresent:    true,
			SEID:           seid,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{},
	}
}

func NewSessionDeletionResponse(seqNum uint32, seid uint64, cause uint8) *Message {
	return &Message{
		Header: MessageHeader{
			Version:        Version1,
			MessageType:    MsgTypeSessionDeletionResponse,
			SEIDPresent:    true,
			SEID:           seid,
			SequenceNumber: seqNum,
		},
		IEs: []*IE{
			NewCauseIE(cause),
		},
	}
}
