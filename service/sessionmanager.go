package service

import (
	"sync"
	"webapp/service/file"
	"webapp/service/websocketmanager"

	"github.com/gorilla/sessions"
)

type SessionStatus[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface] struct {
	FileStatusManager   *file.FileStatusManager[fileidtype, sessionidtype]
	SocketstatusManager *websocketmanager.SocketStatusManager[socketidtype]
}

type SessionStatusManager[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface] struct {
	SessionStatus map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, socketidtype]
	Lock          sync.RWMutex
}

func NewManager[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface]() *SessionStatusManager[sessionidtype, fileidtype, socketidtype] {
	return &SessionStatusManager[sessionidtype, fileidtype, socketidtype]{SessionStatus: make(map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, socketidtype])}
}

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) Add(key sessionidtype) {
	m.Lock.Lock()
	m.SessionStatus[key] = &SessionStatus[sessionidtype, fileidtype, socketidtype]{FileStatusManager: file.NewManager[fileidtype, sessionidtype](), SocketstatusManager: websocketmanager.NewManager[socketidtype]()}
	m.Lock.Unlock()
}
func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) Delete(key sessionidtype) {
	m.Lock.Lock()
	delete(m.SessionStatus, key)
	m.Lock.Unlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) AddFile(sessionid sessionidtype, fileuid fileidtype, filestatus *file.FileStatus[fileidtype, sessionidtype]) {
	m.Lock.RLock()
	m.SessionStatus[sessionid].FileStatusManager.Add(fileuid, filestatus)
	m.Lock.RUnlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) AddSocket(key sessionidtype, socketid socketidtype, fileter string, cookie *sessions.Session) {
	m.Lock.RLock()
	m.SessionStatus[key].SocketstatusManager.Add(socketid, fileter, cookie)
	m.Lock.RUnlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) Get(key sessionidtype) (*SessionStatus[sessionidtype, fileidtype, socketidtype], bool) {
	m.Lock.RLock()
	v, ok := m.SessionStatus[key]
	m.Lock.RUnlock()
	return v, ok
}
func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) FileKeyAndValue(key sessionidtype) ([]fileidtype, []*file.FileStatus[fileidtype, sessionidtype]) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.SessionStatus[key].FileStatusManager.KeyAndValue()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) SocketKeyAndValue(key sessionidtype) ([]socketidtype, []*websocketmanager.SocketStatus[socketidtype]) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.SessionStatus[key].SocketstatusManager.KeyAndValue()
}
