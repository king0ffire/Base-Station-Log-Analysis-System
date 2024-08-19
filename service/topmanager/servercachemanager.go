package topmanager

import (
	"sync"
	"webapp/service/lowermanager"
)

type ServerCacheQueue[useruidtype comparable, fileidtype comparable] struct {
	Queue []*lowermanager.FileStatus[useruidtype, fileidtype]
	Lock  sync.RWMutex
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Push(file *lowermanager.FileStatus[useruidtype, fileidtype]) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Queue = append(m.Queue, file)
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Pop() {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	if len(m.Queue) > 0 {
		m.Queue = m.Queue[1:]
	}
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Len() int {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return len(m.Queue)
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Top() *lowermanager.FileStatus[useruidtype, fileidtype] {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	if len(m.Queue) > 0 {
		return m.Queue[0]
	}
	return nil
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Delete(uid fileidtype) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	for i, v := range m.Queue {
		if v.Uid == uid {
			m.Queue = append(m.Queue[:i], m.Queue[i+1:]...)
			return
		}
	}
}
