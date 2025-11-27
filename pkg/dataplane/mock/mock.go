package mock

import (
	"fmt"
	"log"
	"sync"

	"github.com/veesix-networks/pfcp-go/pkg/up"
)

type MockDataplane struct {
	pdrs map[uint64]map[uint16]*up.PDR
	fars map[uint64]map[uint32]*up.FAR
	qers map[uint64]map[uint32]*up.QER
	urrs map[uint64]map[uint32]*up.URR
	mu   sync.RWMutex
}

func NewMockDataplane() *MockDataplane {
	return &MockDataplane{
		pdrs: make(map[uint64]map[uint16]*up.PDR),
		fars: make(map[uint64]map[uint32]*up.FAR),
		qers: make(map[uint64]map[uint32]*up.QER),
		urrs: make(map[uint64]map[uint32]*up.URR),
	}
}

func (m *MockDataplane) InstallPDR(seid uint64, pdr *up.PDR) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pdrs[seid] == nil {
		m.pdrs[seid] = make(map[uint16]*up.PDR)
	}

	m.pdrs[seid][pdr.ID] = pdr
	log.Printf("[Mock] Installed PDR %d for session %d (precedence=%d, FAR_ID=%d)",
		pdr.ID, seid, pdr.Precedence, pdr.FAR_ID)

	return nil
}

func (m *MockDataplane) RemovePDR(seid uint64, pdrID uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pdrs[seid] != nil {
		delete(m.pdrs[seid], pdrID)
		log.Printf("[Mock] Removed PDR %d from session %d", pdrID, seid)
	}

	return nil
}

func (m *MockDataplane) InstallFAR(seid uint64, far *up.FAR) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.fars[seid] == nil {
		m.fars[seid] = make(map[uint32]*up.FAR)
	}

	m.fars[seid][far.ID] = far
	log.Printf("[Mock] Installed FAR %d for session %d (action=0x%02x)",
		far.ID, seid, far.ApplyAction)

	return nil
}

func (m *MockDataplane) RemoveFAR(seid uint64, farID uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.fars[seid] != nil {
		delete(m.fars[seid], farID)
		log.Printf("[Mock] Removed FAR %d from session %d", farID, seid)
	}

	return nil
}

func (m *MockDataplane) InstallQER(seid uint64, qer *up.QER) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.qers[seid] == nil {
		m.qers[seid] = make(map[uint32]*up.QER)
	}

	m.qers[seid][qer.ID] = qer
	log.Printf("[Mock] Installed QER %d for session %d", qer.ID, seid)

	return nil
}

func (m *MockDataplane) RemoveQER(seid uint64, qerID uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.qers[seid] != nil {
		delete(m.qers[seid], qerID)
		log.Printf("[Mock] Removed QER %d from session %d", qerID, seid)
	}

	return nil
}

func (m *MockDataplane) InstallURR(seid uint64, urr *up.URR) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.urrs[seid] == nil {
		m.urrs[seid] = make(map[uint32]*up.URR)
	}

	m.urrs[seid][urr.ID] = urr
	log.Printf("[Mock] Installed URR %d for session %d", urr.ID, seid)

	return nil
}

func (m *MockDataplane) RemoveURR(seid uint64, urrID uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.urrs[seid] != nil {
		delete(m.urrs[seid], urrID)
		log.Printf("[Mock] Removed URR %d from session %d", urrID, seid)
	}

	return nil
}

func (m *MockDataplane) DeleteSession(seid uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.pdrs, seid)
	delete(m.fars, seid)
	delete(m.qers, seid)
	delete(m.urrs, seid)

	log.Printf("[Mock] Deleted session %d", seid)

	return nil
}

func (m *MockDataplane) GetSessionRules(seid uint64) (int, int, int, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.pdrs[seid]; !exists {
		return 0, 0, 0, 0, fmt.Errorf("session %d not found", seid)
	}

	return len(m.pdrs[seid]), len(m.fars[seid]), len(m.qers[seid]), len(m.urrs[seid]), nil
}
