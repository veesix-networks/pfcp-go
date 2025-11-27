package up

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/veesix-networks/pfcp-go/pkg/protocol"
)

type UPFunction struct {
	config       *Config
	nodeID       []byte
	recoveryTS   uint32
	transport    *protocol.Transport
	cpAddr       *net.UDPAddr
	sessions     map[uint64]*Session
	dataplane    Dataplane
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

type Config struct {
	NodeID            string
	CPAddress         string
	LocalAddr         string
	HeartbeatInterval time.Duration
}

type Session struct {
	LocalSEID  uint64
	RemoteSEID uint64
	PDRs       map[uint16]*PDR
	FARs       map[uint32]*FAR
	QERs       map[uint32]*QER
	URRs       map[uint32]*URR
	CreatedAt  time.Time
}

type PDR struct {
	ID         uint16
	Precedence uint32
	FAR_ID     uint32
	PDI        *PDI
}

type PDI struct {
	SourceInterface uint8
	SDFFilter       []byte
	UE_IPAddress    string
	NetworkInstance string
	ApplicationID   string
}

type FAR struct {
	ID          uint32
	ApplyAction uint8
}

type QER struct {
	ID uint32
}

type URR struct {
	ID uint32
}

func NewUPFunction(cfg *Config, dp Dataplane) (*UPFunction, error) {
	transportCfg := &protocol.TransportConfig{
		LocalAddr: cfg.LocalAddr,
		N1:        3,
		T1:        3 * time.Second,
	}

	transport, err := protocol.NewTransport(transportCfg)
	if err != nil {
		return nil, fmt.Errorf("create transport: %w", err)
	}

	cpAddr, err := net.ResolveUDPAddr("udp", cfg.CPAddress)
	if err != nil {
		return nil, fmt.Errorf("resolve CP address: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	up := &UPFunction{
		config:     cfg,
		nodeID:     []byte(cfg.NodeID),
		recoveryTS: uint32(time.Now().Unix()),
		transport:  transport,
		cpAddr:     cpAddr,
		sessions:   make(map[uint64]*Session),
		dataplane:  dp,
		ctx:        ctx,
		cancel:     cancel,
	}

	up.registerHandlers()

	return up, nil
}

func (up *UPFunction) Start(ctx context.Context) error {
	if err := up.establishAssociation(); err != nil {
		return fmt.Errorf("establish association: %w", err)
	}

	up.wg.Add(1)
	go up.heartbeatLoop()

	<-ctx.Done()
	return up.Stop()
}

func (up *UPFunction) Stop() error {
	up.cancel()
	up.transport.Close()
	up.wg.Wait()
	return nil
}

func (up *UPFunction) registerHandlers() {
	up.transport.RegisterHandler(protocol.MsgTypeSessionEstablishmentRequest, up.handleSessionEstablishmentRequest)
	up.transport.RegisterHandler(protocol.MsgTypeSessionModificationRequest, up.handleSessionModificationRequest)
	up.transport.RegisterHandler(protocol.MsgTypeSessionDeletionRequest, up.handleSessionDeletionRequest)
	up.transport.RegisterHandler(protocol.MsgTypeHeartbeatRequest, up.handleHeartbeatRequest)
}

func (up *UPFunction) establishAssociation() error {
	req := protocol.NewAssociationSetupRequest(0, up.nodeID, up.recoveryTS)
	resp, err := up.transport.SendRequest(req, up.cpAddr, 3*time.Second, 3)
	if err != nil {
		return fmt.Errorf("send association setup request: %w", err)
	}

	causeIE := resp.FindIE(protocol.IETypeCause)
	if causeIE == nil {
		return fmt.Errorf("no cause IE in response")
	}

	cause, err := causeIE.GetCause()
	if err != nil || cause != protocol.CauseRequestAccepted {
		return fmt.Errorf("association setup rejected: cause=%d", cause)
	}

	fmt.Printf("Association established with CP %s\n", up.cpAddr)
	return nil
}

func (up *UPFunction) heartbeatLoop() {
	defer up.wg.Done()

	ticker := time.NewTicker(up.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-up.ctx.Done():
			return
		case <-ticker.C:
			req := protocol.NewHeartbeatRequest(0, up.recoveryTS)
			up.transport.SendRequest(req, up.cpAddr, 3*time.Second, 3)
		}
	}
}

func (up *UPFunction) allocSEID() uint64 {
	up.mu.Lock()
	defer up.mu.Unlock()

	for seid := uint64(1); ; seid++ {
		if _, exists := up.sessions[seid]; !exists {
			return seid
		}
	}
}
