package vpp

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

type SDFFilter struct {
	FlowDescription string
	Action          string
	Direction       string
	Protocol        uint8
	SrcAddr         string
	DstAddr         string
	PortStart       uint16
	PortEnd         uint16
	AddressFamily   uint8
	ToS             uint16
	SPI             uint32
	FlowLabel       uint32
	FilterID        uint32
}

func parseSDFFilter(sdfIEValue []byte) (*SDFFilter, error) {
	if len(sdfIEValue) < 2 {
		return nil, fmt.Errorf("SDF filter too short: %d bytes", len(sdfIEValue))
	}

	flags := sdfIEValue[0]
	fdFlag := flags & 0x01
	ttcFlag := flags & 0x02
	spiFlag := flags & 0x04
	flFlag := flags & 0x08
	bidFlag := flags & 0x10

	offset := 2

	var sdf *SDFFilter

	if fdFlag != 0 {
		if len(sdfIEValue) < offset+2 {
			return nil, fmt.Errorf("SDF filter missing Flow Description length")
		}

		fdLen := binary.BigEndian.Uint16(sdfIEValue[offset : offset+2])
		offset += 2

		if len(sdfIEValue) < offset+int(fdLen) {
			return nil, fmt.Errorf("SDF filter Flow Description truncated")
		}

		flowDesc := string(sdfIEValue[offset : offset+int(fdLen)])
		offset += int(fdLen)

		var err error
		sdf, err = parseFlowDescription(flowDesc)
		if err != nil {
			return nil, err
		}
	} else {
		sdf = &SDFFilter{}
	}

	if ttcFlag != 0 {
		if len(sdfIEValue) < offset+2 {
			return nil, fmt.Errorf("SDF filter missing ToS/Traffic Class")
		}
		sdf.ToS = binary.BigEndian.Uint16(sdfIEValue[offset : offset+2])
		offset += 2
	}

	if spiFlag != 0 {
		if len(sdfIEValue) < offset+4 {
			return nil, fmt.Errorf("SDF filter missing SPI")
		}
		sdf.SPI = binary.BigEndian.Uint32(sdfIEValue[offset : offset+4])
		offset += 4
	}

	if flFlag != 0 {
		if len(sdfIEValue) < offset+3 {
			return nil, fmt.Errorf("SDF filter missing Flow Label")
		}
		sdf.FlowLabel = uint32(sdfIEValue[offset])<<16 | uint32(sdfIEValue[offset+1])<<8 | uint32(sdfIEValue[offset+2])
		sdf.FlowLabel &= 0x000FFFFF
		offset += 3
	}

	if bidFlag != 0 {
		if len(sdfIEValue) < offset+4 {
			return nil, fmt.Errorf("SDF filter missing Filter ID")
		}
		sdf.FilterID = binary.BigEndian.Uint32(sdfIEValue[offset : offset+4])
		offset += 4
	}

	return sdf, nil
}

func parseFlowDescription(flowDesc string) (*SDFFilter, error) {
	parts := strings.Fields(flowDesc)
	if len(parts) < 7 {
		return nil, fmt.Errorf("invalid flow description: %s", flowDesc)
	}

	sdf := &SDFFilter{
		FlowDescription: flowDesc,
		Action:          parts[0],
		Direction:       parts[1],
	}

	protoStr := parts[2]
	switch strings.ToLower(protoStr) {
	case "tcp":
		sdf.Protocol = 6
	case "udp":
		sdf.Protocol = 17
	case "icmp":
		sdf.Protocol = 1
	case "icmpv6":
		sdf.Protocol = 58
	case "ip":
		sdf.Protocol = 0
	default:
		proto, err := strconv.ParseUint(protoStr, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid protocol: %s", protoStr)
		}
		sdf.Protocol = uint8(proto)
	}

	if parts[3] != "from" {
		return nil, fmt.Errorf("expected 'from', got: %s", parts[3])
	}
	sdf.SrcAddr = parts[4]

	if parts[5] != "to" {
		return nil, fmt.Errorf("expected 'to', got: %s", parts[5])
	}
	sdf.DstAddr = parts[6]

	if strings.Contains(sdf.SrcAddr, ":") || strings.Contains(sdf.DstAddr, ":") {
		sdf.AddressFamily = 1
	} else {
		sdf.AddressFamily = 0
	}

	if len(parts) > 7 {
		if err := parsePortRange(parts[7], sdf); err != nil {
			return nil, err
		}
	}

	return sdf, nil
}

func parsePortRange(portRange string, sdf *SDFFilter) error {
	if strings.Contains(portRange, "-") {
		ports := strings.Split(portRange, "-")
		if len(ports) != 2 {
			return fmt.Errorf("invalid port range: %s", portRange)
		}
		start, err := strconv.ParseUint(ports[0], 10, 16)
		if err != nil {
			return fmt.Errorf("invalid start port: %s", ports[0])
		}
		end, err := strconv.ParseUint(ports[1], 10, 16)
		if err != nil {
			return fmt.Errorf("invalid end port: %s", ports[1])
		}
		sdf.PortStart = uint16(start)
		sdf.PortEnd = uint16(end)
	} else {
		port, err := strconv.ParseUint(portRange, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid port: %s", portRange)
		}
		sdf.PortStart = uint16(port)
		sdf.PortEnd = uint16(port)
	}
	return nil
}
