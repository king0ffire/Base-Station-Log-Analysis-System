package session

import (
	"sync"
	"webapp/util/file"
	"webapp/util/socket"

	"github.com/gorilla/sessions"
)

type SessionStatus[sessionidtype comparable, fileidtype comparable, socketidtype socket.Socketidinterface] struct {
	FileStatusManager   *file.FileStatusManager[fileidtype, sessionidtype]
	SocketstatusManager *socket.SocketStatusManager[socketidtype]
}

type SessionStatusManager[sessionidtype comparable, fileidtype comparable, socketidtype socket.Socketidinterface] struct {
	SessionStatus map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, socketidtype]
	Lock          sync.RWMutex
}

func NewManager[sessionidtype comparable, fileidtype comparable, socketidtype socket.Socketidinterface]() *SessionStatusManager[sessionidtype, fileidtype, socketidtype] {
	return &SessionStatusManager[sessionidtype, fileidtype, socketidtype]{SessionStatus: make(map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, socketidtype])}
}

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) Add(key sessionidtype) {
	m.Lock.Lock()
	m.SessionStatus[key] = &SessionStatus[sessionidtype, fileidtype, socketidtype]{FileStatusManager: file.NewManager[fileidtype, sessionidtype](), SocketstatusManager: socket.NewManager[socketidtype]()}
	m.Lock.Unlock()
}
func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) Delete(key sessionidtype) {
	m.Lock.Lock()
	delete(m.SessionStatus, key)
	m.Lock.Unlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) AddFile(key sessionidtype, fileuid fileidtype, filename string, current int, max int, userid sessionidtype) {
	m.Lock.RLock()
	m.SessionStatus[key].FileStatusManager.Add(fileuid, filename, current, max, userid)
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

func (m *SessionStatusManager[sessionidtype, fileidtype, socketidtype]) SocketKeyAndValue(key sessionidtype) ([]socketidtype, []*socket.SocketStatus[socketidtype]) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.SessionStatus[key].SocketstatusManager.KeyAndValue()
}
