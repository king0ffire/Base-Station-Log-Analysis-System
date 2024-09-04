package topmanager

import (
	"sync"
	"webapp/service/lowermanager"
)

type ServerCacheQueue[useruidtype comparable, fileidtype comparable] struct {
	Queue []*lowermanager.FileStatus[useruidtype, fileidtype]
	lock  sync.RWMutex
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Push(file *lowermanager.FileStatus[useruidtype, fileidtype]) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Queue = append(m.Queue, file)
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) PushAndPopWhenFull(file *lowermanager.FileStatus[useruidtype, fileidtype], max int) *lowermanager.FileStatus[useruidtype, fileidtype] {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.Queue) < max {
		m.Queue = append(m.Queue, file)
		return nil
	}
	filetobedeleted := m.Queue[0]
	m.Queue = m.Queue[1:]
	m.Queue = append(m.Queue, file)
	return filetobedeleted
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Pop() (file *lowermanager.FileStatus[useruidtype, fileidtype]) {
	m.lock.Lock()
	defer m.lock.Unlock()
	res := m.Queue[0]
	if len(m.Queue) > 0 {
		m.Queue = m.Queue[1:]
	}
	return res
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.Queue)
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Top() *lowermanager.FileStatus[useruidtype, fileidtype] {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if len(m.Queue) > 0 {
		return m.Queue[0]
	}
	return nil
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Delete(uid fileidtype) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for i, v := range m.Queue {
		if v.Uid == uid {
			m.Queue = append(m.Queue[:i], m.Queue[i+1:]...)
			return
		}
	}
}
