package topmanager

import "sync"

type ServerCacheQueue[fileidtype comparable, useruidtype comparable] struct {
	Queue []*FileStatus[fileidtype, useruidtype]
	Lock  sync.RWMutex
}

func (m *ServerCacheQueue[fileidtype, useruidtype]) Push(file *FileStatus[fileidtype, useruidtype]) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Queue = append(m.Queue, file)
}

func (m *ServerCacheQueue[fileidtype, useruidtype]) Pop() {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	if len(m.Queue) > 0 {
		m.Queue = m.Queue[1:]
	}
}

func (m *ServerCacheQueue[fileidtype, useruidtype]) Len() int {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return len(m.Queue)
}

func (m *ServerCacheQueue[fileidtype, useruidtype]) Top() *FileStatus[fileidtype, useruidtype] {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	if len(m.Queue) > 0 {
		return m.Queue[0]
	}
	return nil
}

func (m *ServerCacheQueue[fileidtype, useruidtype]) Delete(uid fileidtype) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	for i, v := range m.Queue {
		if v.Uid == uid {
			m.Queue = append(m.Queue[:i], m.Queue[i+1:]...)
			return
		}
	}
}
