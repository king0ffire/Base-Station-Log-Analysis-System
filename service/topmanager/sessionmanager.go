package topmanager

import (
	"sync"
	"webapp/service/lowermanager"
	"webapp/service/models"

	"github.com/gorilla/sessions"
)

type SessionStatus[sessionidtype comparable, fileidtype comparable, websocketidtype models.WebSocketID] struct {
	FileStatusManager      *lowermanager.FileStatusManager[sessionidtype, fileidtype]
	WebSocketstatusManager *lowermanager.WebSocketStatusManager[websocketidtype]
}

type SessionStatusManager[sessionidtype comparable, fileidtype comparable, websocketidtype models.WebSocketID] struct {
	SessionStatus map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, websocketidtype]
	lock          sync.RWMutex
}

func NewSessionStatusManager[sessionidtype comparable, fileidtype comparable, websocketidtype models.WebSocketID]() *SessionStatusManager[sessionidtype, fileidtype, websocketidtype] {
	return &SessionStatusManager[sessionidtype, fileidtype, websocketidtype]{SessionStatus: make(map[sessionidtype]*SessionStatus[sessionidtype, fileidtype, websocketidtype])}
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) Add(key sessionidtype) {
	m.lock.Lock()
	m.SessionStatus[key] = &SessionStatus[sessionidtype, fileidtype, websocketidtype]{FileStatusManager: lowermanager.NewFileStatusManager[sessionidtype, fileidtype](), WebSocketstatusManager: lowermanager.NewWebSocketStatusManager[websocketidtype]()}
	m.lock.Unlock()
}
func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) Delete(key sessionidtype) {
	m.lock.Lock()
	delete(m.SessionStatus, key)
	m.lock.Unlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) AddFile(sessionid sessionidtype, fileuid fileidtype, filestatus *models.FileStatus[sessionidtype, fileidtype]) {
	m.lock.RLock()
	m.SessionStatus[sessionid].FileStatusManager.Add(fileuid, filestatus)
	m.lock.RUnlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) AddSocket(key sessionidtype, websocketid websocketidtype, fileter string, cookie *sessions.Session) {
	m.lock.RLock()
	m.SessionStatus[key].WebSocketstatusManager.Add(websocketid, fileter, cookie)
	m.lock.RUnlock()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) Get(key sessionidtype) (*SessionStatus[sessionidtype, fileidtype, websocketidtype], bool) {
	m.lock.RLock()
	v, ok := m.SessionStatus[key]
	m.lock.RUnlock()
	return v, ok
}
func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) FileKeyAndValue(key sessionidtype) ([]fileidtype, []*models.FileStatus[sessionidtype, fileidtype]) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.SessionStatus[key].FileStatusManager.KeyAndValue()
}

func (m *SessionStatusManager[sessionidtype, fileidtype, websocketidtype]) WebSocketKeyAndValue(key sessionidtype) ([]websocketidtype, []*models.WebSocketStatus[websocketidtype]) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.SessionStatus[key].WebSocketstatusManager.KeyAndValue()
}
