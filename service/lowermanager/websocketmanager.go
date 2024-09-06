// SocketStatus map 可以接受任意类型的socket uid，目前直接使用conn的地址
package lowermanager

import (
	"sync"
	"webapp/service/models"

	"github.com/gorilla/sessions"
)

type WebSocketStatusManager[websocketidtype models.WebSocketID] struct {
	WebSocketStatus map[websocketidtype]*models.WebSocketStatus[websocketidtype]
	lock            sync.RWMutex
}

// key could be *websocket.Conn
func NewWebSocketStatusManager[websocketidtype models.WebSocketID]() *WebSocketStatusManager[websocketidtype] {
	return &WebSocketStatusManager[websocketidtype]{WebSocketStatus: make(map[websocketidtype]*models.WebSocketStatus[websocketidtype])}
}

func (m *WebSocketStatusManager[websocketidtype]) Add(key websocketidtype, filter string, cookie *sessions.Session) {
	m.lock.Lock()
	m.WebSocketStatus[key] = &models.WebSocketStatus[websocketidtype]{Filter: filter, Cookie: cookie, Socketid: key}
	m.lock.Unlock()
}
func (m *WebSocketStatusManager[websocketidtype]) Delete(key websocketidtype) {
	m.lock.Lock()
	delete(m.WebSocketStatus, key)
	m.lock.Unlock()
}

func (m *WebSocketStatusManager[websocketidtype]) Get(key websocketidtype) (*models.WebSocketStatus[websocketidtype], bool) {
	m.lock.RLock()
	v, ok := m.WebSocketStatus[key]
	m.lock.RUnlock()
	return v, ok
}

func (m *WebSocketStatusManager[websocketidtype]) Set(key websocketidtype, cookie *sessions.Session) bool {
	s, ok := m.Get(key)
	if ok {
		m.lock.Lock()
		s.Cookie = cookie
		m.lock.Unlock()
		return true
	}
	return false
}

func (m *WebSocketStatusManager[websocketidtype]) KeyAndValue() ([]websocketidtype, []*models.WebSocketStatus[websocketidtype]) {
	socketlist := []websocketidtype{}
	statuslist := []*models.WebSocketStatus[websocketidtype]{}

	m.lock.RLock()
	for k, v := range m.WebSocketStatus {
		socketlist = append(socketlist, k)
		statuslist = append(statuslist, v)
	}
	m.lock.RUnlock()
	return socketlist, statuslist
}
