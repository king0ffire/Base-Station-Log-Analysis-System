package topmanager

import (
	"sync"
	"webapp/service/lowermanager"

	"github.com/gorilla/sessions"
)

type SessionStatus[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID] struct {
	FileStatusManager      *lowermanager.FileStatusManager[sessionidtype, fileidtype]
	WebSocketstatusManager *lowermanager.WebSocketStatusManager[websocketidtype]
}

type SessionStatusManager[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID] struct {
	SessionStatus map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, websocketidtype]
	Lock          sync.RWMutex
}

func NewSessionStatusManager[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID]() *SessionStatusManager[sessionidtype, fileidtype, websocketidtype] {
	return &SessionStatusManager[sessionidtype, fileidtype, websocketidtype]{SessionStatus: make(map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, websocketidtype])}
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) Add(key sessionidtype) {
	m.Lock.Lock()
	m.SessionStatus[key] = &SessionStatus[sessionidtype, fileidtype, websocketidtype]{FileStatusManager: lowermanager.NewFileStatusManager[sessionidtype, fileidtype](), WebSocketstatusManager: lowermanager.NewWebSocketStatusManager[websocketidtype]()}
	m.Lock.Unlock()
}
func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) Delete(key sessionidtype) {
	m.Lock.Lock()
	delete(m.SessionStatus, key)
	m.Lock.Unlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) AddFile(sessionid sessionidtype, fileuid fileidtype, filestatus *lowermanager.FileStatus[sessionidtype, fileidtype]) {
	m.Lock.RLock()
	m.SessionStatus[sessionid].FileStatusManager.Add(fileuid, filestatus)
	m.Lock.RUnlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) AddSocket(key sessionidtype, websocketid websocketidtype, fileter string, cookie *sessions.Session) {
	m.Lock.RLock()
	m.SessionStatus[key].WebSocketstatusManager.Add(websocketid, fileter, cookie)
	m.Lock.RUnlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) Get(key sessionidtype) (*SessionStatus[sessionidtype, fileidtype, websocketidtype], bool) {
	m.Lock.RLock()
	v, ok := m.SessionStatus[key]
	m.Lock.RUnlock()
	return v, ok
}
func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) FileKeyAndValue(key sessionidtype) ([]fileidtype, []*lowermanager.FileStatus[sessionidtype, fileidtype]) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.SessionStatus[key].FileStatusManager.KeyAndValue()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) WebSocketKeyAndValue(key sessionidtype) ([]websocketidtype, []*lowermanager.WebSocketStatus[websocketidtype]) {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return m.SessionStatus[key].WebSocketstatusManager.KeyAndValue()
}
