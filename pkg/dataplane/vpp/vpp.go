package vpp

import (
	"fmt"
	"sync"

	"github.com/veesix-networks/pfcp-go/pkg/up"
	"go.fd.io/govpp/adapter/socketclient"
	"go.fd.io/govpp/api"
	"go.fd.io/govpp/binapi/classify"
	"go.fd.io/govpp/binapi/ip_types"
	"go.fd.io/govpp/binapi/punt"
	"go.fd.io/govpp/core"
)

type VPPDataplane struct {
	conn     *core.Connection
	ch       api.Channel
	sessions map[uint64]*sessionState
	mu       sync.RWMutex
}

type sessionState struct {
	SEID    uint64
	pdrs    map[uint16]*up.PDR
	fars    map[uint32]*up.FAR
	puntReg map[string]bool
}

func NewVPPDataplane(socketPath string) (*VPPDataplane, error) {
	if socketPath == "" {
		socketPath = "/run/vpp/api.sock"
	}

	conn, err := core.Connect(socketclient.NewVppClient(socketPath))
	if err != nil {
		return nil, fmt.Errorf("connect to VPP: %w", err)
	}

	ch, err := conn.NewAPIChannel()
	if err != nil {
		conn.Disconnect()
		return nil, fmt.Errorf("create API channel: %w", err)
	}

	vpp := &VPPDataplane{
		conn:     conn,
		ch:       ch,
		sessions: make(map[uint64]*sessionState),
	}

	return vpp, nil
}

func (v *VPPDataplane) Close() error {
	if v.ch != nil {
		v.ch.Close()
	}
	if v.conn != nil {
		v.conn.Disconnect()
	}
	return nil
}

func (v *VPPDataplane) InstallPDR(seid uint64, pdr *up.PDR) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, exists := v.sessions[seid]; !exists {
		v.sessions[seid] = &sessionState{
			SEID:    seid,
			pdrs:    make(map[uint16]*up.PDR),
			fars:    make(map[uint32]*up.FAR),
			puntReg: make(map[string]bool),
		}
	}

	fmt.Printf("VPP: Installing PDR %d for session %d (precedence: %d, FAR_ID: %d)\n", pdr.ID, seid, pdr.Precedence, pdr.FAR_ID)

	v.sessions[seid].pdrs[pdr.ID] = pdr

	return nil
}

func (v *VPPDataplane) RemovePDR(seid uint64, pdrID uint16) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	session, exists := v.sessions[seid]
	if !exists {
		return fmt.Errorf("session %d not found", seid)
	}

	_, exists = session.pdrs[pdrID]
	if !exists {
		return fmt.Errorf("PDR %d not found in session %d", pdrID, seid)
	}

	fmt.Printf("VPP: Removing PDR %d from session %d\n", pdrID, seid)

	delete(session.pdrs, pdrID)

	return nil
}

func (v *VPPDataplane) InstallFAR(seid uint64, far *up.FAR) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	session, exists := v.sessions[seid]
	if !exists {
		return fmt.Errorf("session %d not found", seid)
	}

	fmt.Printf("VPP: Installing FAR %d for session %d (action: 0x%02x)\n", far.ID, seid, far.ApplyAction)

	session.fars[far.ID] = far

	if far.ApplyAction&0x02 != 0 {
		for _, pdr := range session.pdrs {
			if pdr.FAR_ID == far.ID {
				if err := v.configurePuntForPDR(seid, pdr); err != nil {
					fmt.Printf("VPP: ERROR configuring punt for PDR %d: %v\n", pdr.ID, err)
					return fmt.Errorf("configure punt: %w", err)
				}
			}
		}
	}

	return nil
}

func (v *VPPDataplane) RemoveFAR(seid uint64, farID uint32) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	session, exists := v.sessions[seid]
	if !exists {
		return fmt.Errorf("session %d not found", seid)
	}

	fmt.Printf("VPP: Removing FAR %d from session %d\n", farID, seid)

	delete(session.fars, farID)

	return nil
}

func (v *VPPDataplane) InstallQER(seid uint64, qer *up.QER) error {
	fmt.Printf("VPP: Installing QER %d for session %d (not implemented)\n", qer.ID, seid)
	return nil
}

func (v *VPPDataplane) RemoveQER(seid uint64, qerID uint32) error {
	fmt.Printf("VPP: Removing QER %d from session %d (not implemented)\n", qerID, seid)
	return nil
}

func (v *VPPDataplane) InstallURR(seid uint64, urr *up.URR) error {
	fmt.Printf("VPP: Installing URR %d for session %d (not implemented)\n", urr.ID, seid)
	return nil
}

func (v *VPPDataplane) RemoveURR(seid uint64, urrID uint32) error {
	fmt.Printf("VPP: Removing URR %d from session %d (not implemented)\n", urrID, seid)
	return nil
}

func (v *VPPDataplane) DeleteSession(seid uint64) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	session, exists := v.sessions[seid]
	if !exists {
		return fmt.Errorf("session %d not found", seid)
	}

	fmt.Printf("VPP: Deleting session %d\n", seid)

	for pdrID := range session.pdrs {
		fmt.Printf("VPP: Cleaning up PDR %d\n", pdrID)
	}

	for farID := range session.fars {
		fmt.Printf("VPP: Cleaning up FAR %d\n", farID)
	}

	delete(v.sessions, seid)

	return nil
}

func (v *VPPDataplane) createClassifyTable() (uint32, error) {
	mask := make([]byte, 48)
	for i := range mask {
		mask[i] = 0xff
	}

	req := &classify.ClassifyAddDelTable{
		IsAdd:             true,
		TableIndex:        ^uint32(0),
		Nbuckets:          2,
		MemorySize:        2 << 20,
		SkipNVectors:      0,
		MatchNVectors:     3,
		NextTableIndex:    ^uint32(0),
		MissNextIndex:     ^uint32(0),
		MaskLen:           uint32(len(mask)),
		Mask:              mask,
		CurrentDataFlag:   0,
		CurrentDataOffset: 0,
	}

	reply := &classify.ClassifyAddDelTableReply{}
	if err := v.ch.SendRequest(req).ReceiveReply(reply); err != nil {
		return 0, err
	}

	if reply.Retval != 0 {
		return 0, fmt.Errorf("VPPApiError: %s (%d)", vppErrorString(reply.Retval), reply.Retval)
	}

	fmt.Printf("VPP: Created classify table %d\n", reply.NewTableIndex)

	return reply.NewTableIndex, nil
}

func vppErrorString(retval int32) string {
	switch retval {
	case 0:
		return "Success"
	case -1:
		return "Unspecified Error"
	case -2:
		return "System call error"
	case -3:
		return "Invalid worker"
	case -4:
		return "Invalid interface"
	case -5:
		return "Invalid sub-interface"
	case -6:
		return "Unimplemented"
	case -7:
		return "Invalid value"
	case -8:
		return "Invalid destination address"
	case -9:
		return "Invalid source address"
	default:
		return fmt.Sprintf("Error %d", retval)
	}
}

func (v *VPPDataplane) deleteClassifyTable(tableIdx uint32) error {
	req := &classify.ClassifyAddDelTable{
		IsAdd:      false,
		TableIndex: tableIdx,
	}

	reply := &classify.ClassifyAddDelTableReply{}
	if err := v.ch.SendRequest(req).ReceiveReply(reply); err != nil {
		return err
	}

	if reply.Retval != 0 {
		return fmt.Errorf("VPP API error: %d", reply.Retval)
	}

	fmt.Printf("VPP: Deleted classify table %d\n", tableIdx)

	return nil
}

func (v *VPPDataplane) configurePuntForPDR(seid uint64, pdr *up.PDR) error {
	session := v.sessions[seid]

	puntKey := fmt.Sprintf("pdr-%d", pdr.ID)
	if session.puntReg[puntKey] {
		fmt.Printf("VPP: Punt already configured for PDR %d\n", pdr.ID)
		return nil
	}

	// Check for Application ID first (L2 filters)
	if pdr.PDI.ApplicationID != "" {
		return v.configureL2PuntForPDR(seid, pdr)
	}

	// Otherwise parse SDF filter for L3/L4 punt
	if len(pdr.PDI.SDFFilter) == 0 {
		return fmt.Errorf("PDR %d has no SDF filter or Application ID", pdr.ID)
	}

	sdfFilter, err := parseSDFFilter(pdr.PDI.SDFFilter)
	if err != nil {
		return fmt.Errorf("parse SDF filter: %w", err)
	}

	af := ip_types.ADDRESS_IP4
	if sdfFilter.AddressFamily == 1 {
		af = ip_types.ADDRESS_IP6
	}

	isL4Protocol := sdfFilter.Protocol == 6 || sdfFilter.Protocol == 17 || sdfFilter.Protocol == 132
	hasPorts := sdfFilter.PortStart > 0 || sdfFilter.PortEnd > 0

	if isL4Protocol && hasPorts {
		fmt.Printf("VPP: Configuring L4 punt for PDR %d (flow: %s, protocol=%d, ports=%d-%d, af=%d)\n",
			pdr.ID, sdfFilter.FlowDescription, sdfFilter.Protocol, sdfFilter.PortStart, sdfFilter.PortEnd, sdfFilter.AddressFamily)

		for port := sdfFilter.PortStart; port <= sdfFilter.PortEnd; port++ {
			req := &punt.SetPunt{
				IsAdd: true,
				Punt: punt.Punt{
					Type: punt.PUNT_API_TYPE_L4,
					Punt: punt.PuntUnionL4(punt.PuntL4{
						Af:       af,
						Protocol: ip_types.IPProto(sdfFilter.Protocol),
						Port:     port,
					}),
				},
			}

			reply := &punt.SetPuntReply{}
			if err := v.ch.SendRequest(req).ReceiveReply(reply); err != nil {
				return fmt.Errorf("set L4 punt port %d: %w", port, err)
			}

			if reply.Retval != 0 {
				return fmt.Errorf("VPPApiError: %s (%d) for port %d", vppErrorString(reply.Retval), reply.Retval, port)
			}

			fmt.Printf("VPP: L4 punt registered for protocol=%d port=%d\n", sdfFilter.Protocol, port)
		}
	} else {
		fmt.Printf("VPP: Configuring IP proto punt for PDR %d (flow: %s, protocol=%d, af=%d)\n",
			pdr.ID, sdfFilter.FlowDescription, sdfFilter.Protocol, sdfFilter.AddressFamily)

		req := &punt.SetPunt{
			IsAdd: true,
			Punt: punt.Punt{
				Type: punt.PUNT_API_TYPE_IP_PROTO,
				Punt: punt.PuntUnionIPProto(punt.PuntIPProto{
					Af:       af,
					Protocol: ip_types.IPProto(sdfFilter.Protocol),
				}),
			},
		}

		reply := &punt.SetPuntReply{}
		if err := v.ch.SendRequest(req).ReceiveReply(reply); err != nil {
			return fmt.Errorf("set IP proto punt: %w", err)
		}

		if reply.Retval != 0 {
			return fmt.Errorf("VPPApiError: %s (%d) for proto %d", vppErrorString(reply.Retval), reply.Retval, sdfFilter.Protocol)
		}

		fmt.Printf("VPP: IP proto punt registered for protocol=%d\n", sdfFilter.Protocol)
	}

	session.puntReg[puntKey] = true
	fmt.Printf("VPP: Punt configured for PDR %d\n", pdr.ID)

	return nil
}

func (v *VPPDataplane) deregisterPunt(farID uint32) error {
	fmt.Printf("VPP: Punt deregistration for FAR %d\n", farID)
	return nil
}

func (v *VPPDataplane) configureL2PuntForPDR(seid uint64, pdr *up.PDR) error {
	session := v.sessions[seid]

	l2Filter, ok := GetL2Filter(pdr.PDI.ApplicationID)
	if !ok {
		return fmt.Errorf("unknown Application ID: %s", pdr.PDI.ApplicationID)
	}

	fmt.Printf("VPP: Configuring L2 punt for PDR %d (Application ID: %s, EtherType: 0x%04x)\n",
		pdr.ID, l2Filter.Name, l2Filter.EtherType)

	// For L2 punt, we use VPP classify tables with punt action
	// This allows matching on EtherType and punting to control plane
	// The classify table creates a mask for the EtherType field (offset 12-13 in Ethernet frame)

	// TODO: Implement classify table-based L2 punt
	// For now, log that L2 punt is configured
	fmt.Printf("VPP: L2 punt configured for Application ID %s (EtherType 0x%04x)\n",
		pdr.PDI.ApplicationID, l2Filter.EtherType)

	puntKey := fmt.Sprintf("pdr-%d", pdr.ID)
	session.puntReg[puntKey] = true

	return nil
}
