package cp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/veesix-networks/pfcp-go/pkg/protocol"
)

type CPFunction struct {
	config       *Config
	nodeID       []byte
	recoveryTS   uint32
	transport    *protocol.Transport
	associations map[string]*Association
	sessions     map[uint64]*Session
	nextSEID     uint64
	store        NorthboundStore
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

type Config struct {
	NodeID            string
	ListenAddr        string
	HeartbeatInterval time.Duration
	RetransmitN1      int
	RetransmitT1      time.Duration
}

type Association struct {
	NodeID        []byte
	RemoteAddr    *net.UDPAddr
	RecoveryTS    uint32
	Features      []byte
	LastHeartbeat time.Time
	EstablishedAt time.Time
}

type Session struct {
	LocalSEID  uint64
	RemoteSEID uint64
	NodeID     string
	PDRs       map[uint16]*PDR
	FARs       map[uint32]*FAR
	QERs       map[uint32]*QER
	URRs       map[uint32]*URR
	CreatedAt  time.Time
}

type PDR struct {
	ID         uint16
	Precedence uint32
	PDI        *PacketDetectionInfo
	FAR_ID     uint32
	QER_IDs    []uint32
	URR_IDs    []uint32
}

type PacketDetectionInfo struct {
	SourceInterface uint8
	NetworkInstance string
	UE_IPAddress    net.IP
	SDFFilter       string
	ApplicationID   string
}

type FAR struct {
	ID                   uint32
	ApplyAction          uint8
	ForwardingParameters *ForwardingParams
}

type ForwardingParams struct {
	DestinationInterface uint8
	NetworkInstance      string
}

type QER struct {
	ID         uint32
	GateStatus uint8
	MBR_UL     uint64
	MBR_DL     uint64
	GBR_UL     uint64
	GBR_DL     uint64
}

type URR struct {
	ID                uint32
	MeasurementMethod uint8
	ReportingTriggers uint32
	VolumeThreshold   uint64
	TimeThreshold     uint32
}

func NewCPFunction(cfg *Config, store NorthboundStore) (*CPFunction, error) {
	transportCfg := &protocol.TransportConfig{
		LocalAddr: cfg.ListenAddr,
		N1:        cfg.RetransmitN1,
		T1:        cfg.RetransmitT1,
	}

	transport, err := protocol.NewTransport(transportCfg)
	if err != nil {
		return nil, fmt.Errorf("create transport: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cp := &CPFunction{
		config:       cfg,
		nodeID:       []byte(cfg.NodeID),
		recoveryTS:   uint32(time.Now().Unix()),
		transport:    transport,
		associations: make(map[string]*Association),
		sessions:     make(map[uint64]*Session),
		nextSEID:     1,
		store:        store,
		ctx:          ctx,
		cancel:       cancel,
	}

	cp.registerHandlers()

	return cp, nil
}

func (cp *CPFunction) Start(ctx context.Context) error {
	cp.wg.Add(1)
	go cp.heartbeatLoop()

	<-ctx.Done()
	return cp.Stop()
}

func (cp *CPFunction) Stop() error {
	cp.cancel()
	cp.transport.Close()
	cp.wg.Wait()
	return nil
}

func (cp *CPFunction) registerHandlers() {
	cp.transport.RegisterHandler(protocol.MsgTypeAssociationSetupRequest, cp.handleAssociationSetupRequest)
	cp.transport.RegisterHandler(protocol.MsgTypeAssociationReleaseRequest, cp.handleAssociationReleaseRequest)
	cp.transport.RegisterHandler(protocol.MsgTypeHeartbeatRequest, cp.handleHeartbeatRequest)
	cp.transport.RegisterHandler(protocol.MsgTypeSessionReportRequest, cp.handleSessionReportRequest)
}

func (cp *CPFunction) heartbeatLoop() {
	defer cp.wg.Done()

	ticker := time.NewTicker(cp.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.sendHeartbeats()
		}
	}
}

func (cp *CPFunction) sendHeartbeats() {
	cp.mu.RLock()
	associations := make([]*Association, 0, len(cp.associations))
	for _, assoc := range cp.associations {
		associations = append(associations, assoc)
	}
	cp.mu.RUnlock()

	for _, assoc := range associations {
		req := protocol.NewHeartbeatRequest(0, cp.recoveryTS)
		_, err := cp.transport.SendRequest(req, assoc.RemoteAddr, 3*time.Second, 3)
		if err != nil {
			continue
		}

		cp.mu.Lock()
		assoc.LastHeartbeat = time.Now()
		cp.mu.Unlock()
	}
}

func (cp *CPFunction) CreateSession(nodeID string, pdrs []*PDR, fars []*FAR, qers []*QER, urrs []*URR) (uint64, error) {
	cp.mu.RLock()
	assoc, ok := cp.associations[nodeID]
	cp.mu.RUnlock()

	if !ok {
		return 0, fmt.Errorf("no association with node %s", nodeID)
	}

	seid := cp.allocSEID()

	createPDRs, err := cp.marshalPDRs(pdrs)
	if err != nil {
		return 0, fmt.Errorf("marshal PDRs: %w", err)
	}

	createFARs, err := cp.marshalFARs(fars)
	if err != nil {
		return 0, fmt.Errorf("marshal FARs: %w", err)
	}

	createQERs, err := cp.marshalQERs(qers)
	if err != nil {
		return 0, fmt.Errorf("marshal QERs: %w", err)
	}

	createURRs, err := cp.marshalURRs(urrs)
	if err != nil {
		return 0, fmt.Errorf("marshal URRs: %w", err)
	}

	req := protocol.NewSessionEstablishmentRequest(0, 0, createPDRs, createFARs, createQERs, createURRs)
	resp, err := cp.transport.SendRequest(req, assoc.RemoteAddr, cp.config.RetransmitT1, cp.config.RetransmitN1)
	if err != nil {
		return 0, fmt.Errorf("send request: %w", err)
	}

	causeIE := resp.FindIE(protocol.IETypeCause)
	if causeIE == nil {
		return 0, fmt.Errorf("no cause IE in response")
	}

	cause, err := causeIE.GetCause()
	if err != nil || cause != protocol.CauseRequestAccepted {
		return 0, fmt.Errorf("session establishment rejected: cause=%d", cause)
	}

	session := &Session{
		LocalSEID:  seid,
		RemoteSEID: resp.Header.SEID,
		NodeID:     nodeID,
		PDRs:       make(map[uint16]*PDR),
		FARs:       make(map[uint32]*FAR),
		QERs:       make(map[uint32]*QER),
		URRs:       make(map[uint32]*URR),
		CreatedAt:  time.Now(),
	}

	for _, pdr := range pdrs {
		session.PDRs[pdr.ID] = pdr
	}
	for _, far := range fars {
		session.FARs[far.ID] = far
	}
	for _, qer := range qers {
		session.QERs[qer.ID] = qer
	}
	for _, urr := range urrs {
		session.URRs[urr.ID] = urr
	}

	cp.mu.Lock()
	cp.sessions[seid] = session
	cp.mu.Unlock()

	if cp.store != nil {
		cp.store.StoreSession(seid, session)
	}

	return seid, nil
}

func (cp *CPFunction) DeleteSession(seid uint64) error {
	cp.mu.RLock()
	session, ok := cp.sessions[seid]
	cp.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %d not found", seid)
	}

	cp.mu.RLock()
	assoc, ok := cp.associations[session.NodeID]
	cp.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no association with node %s", session.NodeID)
	}

	req := protocol.NewSessionDeletionRequest(0, session.RemoteSEID)
	resp, err := cp.transport.SendRequest(req, assoc.RemoteAddr, cp.config.RetransmitT1, cp.config.RetransmitN1)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	causeIE := resp.FindIE(protocol.IETypeCause)
	if causeIE == nil {
		return fmt.Errorf("no cause IE in response")
	}

	cause, _ := causeIE.GetCause()
	if cause != protocol.CauseRequestAccepted {
		return fmt.Errorf("session deletion rejected: cause=%d", cause)
	}

	cp.mu.Lock()
	delete(cp.sessions, seid)
	cp.mu.Unlock()

	if cp.store != nil {
		cp.store.DeleteSession(seid)
	}

	return nil
}

func (cp *CPFunction) allocSEID() uint64 {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	seid := cp.nextSEID
	cp.nextSEID++
	return seid
}

func (cp *CPFunction) marshalPDRs(pdrs []*PDR) ([]*protocol.IE, error) {
	var ies []*protocol.IE
	for _, pdr := range pdrs {
		pdrIEs := []*protocol.IE{
			protocol.NewPDR_ID_IE(pdr.ID),
			protocol.NewPrecedenceIE(pdr.Precedence),
		}

		if pdr.PDI != nil {
			pdiIEs := []*protocol.IE{
				protocol.NewSourceInterfaceIE(pdr.PDI.SourceInterface),
			}

			if pdr.PDI.UE_IPAddress != nil {
				isV6 := pdr.PDI.UE_IPAddress.To4() == nil
				pdiIEs = append(pdiIEs, protocol.NewUE_IPAddressIE(pdr.PDI.UE_IPAddress, isV6))
			}

			if pdr.PDI.SDFFilter != "" {
				pdiIEs = append(pdiIEs, protocol.NewSDFFilterIE(pdr.PDI.SDFFilter))
			}

			if pdr.PDI.ApplicationID != "" {
				pdiIEs = append(pdiIEs, protocol.NewApplicationIDIE(pdr.PDI.ApplicationID))
			}

			pdiIE, err := protocol.NewGroupedIE(protocol.IETypePDI, pdiIEs)
			if err != nil {
				return nil, err
			}
			pdrIEs = append(pdrIEs, pdiIE)
		}

		pdrIEs = append(pdrIEs, protocol.NewFAR_ID_IE(pdr.FAR_ID))

		for _, qerID := range pdr.QER_IDs {
			pdrIEs = append(pdrIEs, protocol.NewQER_ID_IE(qerID))
		}

		for _, urrID := range pdr.URR_IDs {
			pdrIEs = append(pdrIEs, protocol.NewURR_ID_IE(urrID))
		}

		createPDR, err := protocol.NewGroupedIE(protocol.IETypeCreatePDR, pdrIEs)
		if err != nil {
			return nil, err
		}
		ies = append(ies, createPDR)
	}
	return ies, nil
}

func (cp *CPFunction) marshalFARs(fars []*FAR) ([]*protocol.IE, error) {
	var ies []*protocol.IE
	for _, far := range fars {
		farIEs := []*protocol.IE{
			protocol.NewFAR_ID_IE(far.ID),
			protocol.NewApplyActionIE(far.ApplyAction),
		}

		if far.ForwardingParameters != nil {
			fpIEs := []*protocol.IE{
				protocol.NewDestinationInterfaceIE(far.ForwardingParameters.DestinationInterface),
			}

			fpIE, err := protocol.NewGroupedIE(protocol.IETypeForwardingParameters, fpIEs)
			if err != nil {
				return nil, err
			}
			farIEs = append(farIEs, fpIE)
		}

		createFAR, err := protocol.NewGroupedIE(protocol.IETypeCreateFAR, farIEs)
		if err != nil {
			return nil, err
		}
		ies = append(ies, createFAR)
	}
	return ies, nil
}

func (cp *CPFunction) marshalQERs(qers []*QER) ([]*protocol.IE, error) {
	var ies []*protocol.IE
	for _, qer := range qers {
		qerIEs := []*protocol.IE{
			protocol.NewQER_ID_IE(qer.ID),
		}

		createQER, err := protocol.NewGroupedIE(protocol.IETypeCreateQER, qerIEs)
		if err != nil {
			return nil, err
		}
		ies = append(ies, createQER)
	}
	return ies, nil
}

func (cp *CPFunction) marshalURRs(urrs []*URR) ([]*protocol.IE, error) {
	var ies []*protocol.IE
	for _, urr := range urrs {
		urrIEs := []*protocol.IE{
			protocol.NewURR_ID_IE(urr.ID),
		}

		createURR, err := protocol.NewGroupedIE(protocol.IETypeCreateURR, urrIEs)
		if err != nil {
			return nil, err
		}
		ies = append(ies, createURR)
	}
	return ies, nil
}
