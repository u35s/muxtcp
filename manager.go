package muxtcp

import "sync"

type muxtcpManager struct {
	lock sync.RWMutex
	pool map[uint]*MuxTcp
}

var m *muxtcpManager

func init() {
	m = new(muxtcpManager)
	m.pool = make(map[uint]*MuxTcp)
}

func Add(key uint, mux *MuxTcp) {
	m.lock.Lock()
	m.pool[key] = mux
	m.lock.Unlock()
}

func Get(key uint) *MuxTcp {
	m.lock.RLock()
	defer m.lock.RUnlock()
	mux, ok := m.pool[key]
	if ok {
		return mux
	}
	return nil
}

func Remove(key uint) {
	m.lock.Lock()
	delete(m.pool, key)
	m.lock.Unlock()
}
