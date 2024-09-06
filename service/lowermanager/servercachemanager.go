package lowermanager

import (
	"sync"
	"webapp/service/models"
)

type ServerCacheQueue[useruidtype comparable, fileidtype comparable] struct {
	Queue []*models.FileStatus[useruidtype, fileidtype]
	lock  sync.RWMutex
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) Push(file *models.FileStatus[useruidtype, fileidtype]) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.Queue = append(m.Queue, file)
}

func (m *ServerCacheQueue[useruidtype, fileidtype]) PushAndPopWhenFull(file *models.FileStatus[useruidtype, fileidtype], max int) *models.FileStatus[useruidtype, fileidtype] {
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

func (m *ServerCacheQueue[useruidtype, fileidtype]) Pop() (file *models.FileStatus[useruidtype, fileidtype]) {
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

func (m *ServerCacheQueue[useruidtype, fileidtype]) Top() *models.FileStatus[useruidtype, fileidtype] {
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
