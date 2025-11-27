package cp

type NorthboundStore interface {
	StoreSession(seid uint64, session *Session) error
	GetSession(seid uint64) (*Session, error)
	DeleteSession(seid uint64) error
	ListSessions() ([]uint64, error)
}

type MemoryStore struct {
	sessions map[uint64]*Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[uint64]*Session),
	}
}

func (m *MemoryStore) StoreSession(seid uint64, session *Session) error {
	m.sessions[seid] = session
	return nil
}

func (m *MemoryStore) GetSession(seid uint64) (*Session, error) {
	session, ok := m.sessions[seid]
	if !ok {
		return nil, nil
	}
	return session, nil
}

func (m *MemoryStore) DeleteSession(seid uint64) error {
	delete(m.sessions, seid)
	return nil
}

func (m *MemoryStore) ListSessions() ([]uint64, error) {
	seids := make([]uint64, 0, len(m.sessions))
	for seid := range m.sessions {
		seids = append(seids, seid)
	}
	return seids, nil
}
