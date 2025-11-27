package protocol

import (
	"encoding/binary"
	"fmt"
)

type MessageHeader struct {
	Version         uint8
	FollowOn        bool
	MessagePriority bool
	SEIDPresent     bool
	MessageType     uint8
	MessageLength   uint16
	SEID            uint64
	SequenceNumber  uint32
	Priority        uint8
}

func (h *MessageHeader) Marshal() ([]byte, error) {
	var buf []byte

	if h.SEIDPresent {
		buf = make([]byte, 16)
	} else {
		buf = make([]byte, 8)
	}

	flags := h.Version << 5

	if h.FollowOn {
		flags |= 0x04
	}
	if h.MessagePriority {
		flags |= 0x02
	}
	if h.SEIDPresent {
		flags |= 0x01
	}

	buf[0] = flags
	buf[1] = h.MessageType
	binary.BigEndian.PutUint16(buf[2:4], h.MessageLength)

	if h.SEIDPresent {
		binary.BigEndian.PutUint64(buf[4:12], h.SEID)
		buf[12] = byte(h.SequenceNumber >> 16)
		buf[13] = byte(h.SequenceNumber >> 8)
		buf[14] = byte(h.SequenceNumber)

		if h.MessagePriority {
			buf[15] = h.Priority << 4
		} else {
			buf[15] = 0
		}
	} else {
		buf[4] = byte(h.SequenceNumber >> 16)
		buf[5] = byte(h.SequenceNumber >> 8)
		buf[6] = byte(h.SequenceNumber)

		if h.MessagePriority {
			buf[7] = h.Priority << 4
		} else {
			buf[7] = 0
		}
	}

	return buf, nil
}

func (h *MessageHeader) Unmarshal(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("header too short: %d bytes", len(data))
	}

	flags := data[0]
	h.Version = (flags >> 5) & 0x07
	h.FollowOn = (flags & 0x04) != 0
	h.MessagePriority = (flags & 0x02) != 0
	h.SEIDPresent = (flags & 0x01) != 0

	h.MessageType = data[1]
	h.MessageLength = binary.BigEndian.Uint16(data[2:4])

	if h.SEIDPresent {
		if len(data) < 16 {
			return fmt.Errorf("header with SEID too short: %d bytes", len(data))
		}
		h.SEID = binary.BigEndian.Uint64(data[4:12])
		h.SequenceNumber = uint32(data[12])<<16 | uint32(data[13])<<8 | uint32(data[14])
		if h.MessagePriority {
			h.Priority = data[15] >> 4
		}
	} else {
		h.SequenceNumber = uint32(data[4])<<16 | uint32(data[5])<<8 | uint32(data[6])
		if h.MessagePriority {
			h.Priority = data[7] >> 4
		}
	}

	return nil
}

func (h *MessageHeader) Len() int {
	if h.SEIDPresent {
		return 16
	}
	return 8
}
