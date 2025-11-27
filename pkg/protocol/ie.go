package protocol

import (
	"encoding/binary"
	"fmt"
	"net"
)

type IE struct {
	Type         uint16
	EnterpriseID uint16
	Value        []byte
}

func (ie *IE) Marshal() ([]byte, error) {
	hasEnterpriseID := ie.Type >= 32768
	length := len(ie.Value)

	if hasEnterpriseID {
		length += 2
	}

	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint16(buf[0:2], ie.Type)
	binary.BigEndian.PutUint16(buf[2:4], uint16(length))

	if hasEnterpriseID {
		binary.BigEndian.PutUint16(buf[4:6], ie.EnterpriseID)
		copy(buf[6:], ie.Value)
	} else {
		copy(buf[4:], ie.Value)
	}

	return buf, nil
}

func (ie *IE) Unmarshal(data []byte) (int, error) {
	if len(data) < 4 {
		return 0, fmt.Errorf("IE too short: %d bytes", len(data))
	}

	ie.Type = binary.BigEndian.Uint16(data[0:2])
	length := binary.BigEndian.Uint16(data[2:4])

	if len(data) < 4+int(length) {
		return 0, fmt.Errorf("IE data truncated: need %d bytes, have %d", 4+length, len(data))
	}

	hasEnterpriseID := ie.Type >= 32768

	if hasEnterpriseID {
		if length < 2 {
			return 0, fmt.Errorf("IE with enterprise ID too short")
		}
		ie.EnterpriseID = binary.BigEndian.Uint16(data[4:6])
		ie.Value = make([]byte, length-2)
		copy(ie.Value, data[6:6+length-2])
	} else {
		ie.Value = make([]byte, length)
		copy(ie.Value, data[4:4+length])
	}

	return 4 + int(length), nil
}

func NewCauseIE(cause uint8) *IE {
	return &IE{
		Type:  IETypeCause,
		Value: []byte{cause},
	}
}

func NewNodeIDIE(nodeID []byte) *IE {
	value := make([]byte, 1+len(nodeID))
	value[0] = 0
	copy(value[1:], nodeID)
	return &IE{
		Type:  IETypeNodeID,
		Value: value,
	}
}

func NewRecoveryTimeStampIE(timestamp uint32) *IE {
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, timestamp)
	return &IE{
		Type:  IETypeRecoveryTimeStamp,
		Value: value,
	}
}

func NewSourceInterfaceIE(iface uint8) *IE {
	return &IE{
		Type:  IETypeSourceInterface,
		Value: []byte{iface},
	}
}

func NewDestinationInterfaceIE(iface uint8) *IE {
	return &IE{
		Type:  IETypeDestinationInterface,
		Value: []byte{iface},
	}
}

func NewApplyActionIE(action uint8) *IE {
	return &IE{
		Type:  IETypeApplyAction,
		Value: []byte{action},
	}
}

func NewPDR_ID_IE(id uint16) *IE {
	value := make([]byte, 2)
	binary.BigEndian.PutUint16(value, id)
	return &IE{
		Type:  IETypePDR_ID,
		Value: value,
	}
}

func NewFAR_ID_IE(id uint32) *IE {
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, id)
	return &IE{
		Type:  IETypeFAR_ID,
		Value: value,
	}
}

func NewQER_ID_IE(id uint32) *IE {
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, id)
	return &IE{
		Type:  IETypeQER_ID,
		Value: value,
	}
}

func NewURR_ID_IE(id uint32) *IE {
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, id)
	return &IE{
		Type:  IETypeURR_ID,
		Value: value,
	}
}

func NewPrecedenceIE(precedence uint32) *IE {
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, precedence)
	return &IE{
		Type:  IETypePrecedence,
		Value: value,
	}
}

func NewUE_IPAddressIE(ip net.IP, isV6 bool) *IE {
	flags := uint8(0)
	var value []byte

	if isV6 {
		flags |= 0x01
		value = make([]byte, 1+16)
		value[0] = flags
		copy(value[1:], ip.To16())
	} else {
		flags |= 0x02
		value = make([]byte, 1+4)
		value[0] = flags
		copy(value[1:], ip.To4())
	}

	return &IE{
		Type:  IETypeUE_IPAddress,
		Value: value,
	}
}

func NewSDFFilterIE(flowDescription string) *IE {
	flags := uint8(0x01) // FD flag - Flow Description present
	fdBytes := []byte(flowDescription)

	// Structure: flags (1) + spare (1) + fdLen (2) + fdBytes
	value := make([]byte, 2+2+len(fdBytes))
	value[0] = flags
	value[1] = 0 // Spare byte
	binary.BigEndian.PutUint16(value[2:4], uint16(len(fdBytes)))
	copy(value[4:], fdBytes)

	return &IE{
		Type:  IETypeSDFFilter,
		Value: value,
	}
}

func NewApplicationIDIE(appID string) *IE {
	return &IE{
		Type:  IETypeApplicationID,
		Value: []byte(appID),
	}
}

func NewGroupedIE(ieType uint16, children []*IE) (*IE, error) {
	var value []byte
	for _, child := range children {
		childData, err := child.Marshal()
		if err != nil {
			return nil, err
		}
		value = append(value, childData...)
	}

	return &IE{
		Type:  ieType,
		Value: value,
	}, nil
}

func ParseGroupedIE(data []byte) ([]*IE, error) {
	var ies []*IE
	offset := 0

	for offset < len(data) {
		ie := &IE{}
		n, err := ie.Unmarshal(data[offset:])
		if err != nil {
			return nil, err
		}
		ies = append(ies, ie)
		offset += n
	}

	return ies, nil
}

func (ie *IE) GetCause() (uint8, error) {
	if ie.Type != IETypeCause || len(ie.Value) < 1 {
		return 0, fmt.Errorf("invalid Cause IE")
	}
	return ie.Value[0], nil
}

func (ie *IE) GetPDR_ID() (uint16, error) {
	if ie.Type != IETypePDR_ID || len(ie.Value) < 2 {
		return 0, fmt.Errorf("invalid PDR_ID IE")
	}
	return binary.BigEndian.Uint16(ie.Value), nil
}

func (ie *IE) GetFAR_ID() (uint32, error) {
	if ie.Type != IETypeFAR_ID || len(ie.Value) < 4 {
		return 0, fmt.Errorf("invalid FAR_ID IE")
	}
	return binary.BigEndian.Uint32(ie.Value), nil
}
