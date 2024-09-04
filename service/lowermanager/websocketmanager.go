// SocketStatus map 可以接受任意类型的socket uid，目前直接使用conn的地址
package lowermanager

import (
	"sync"

	"github.com/gorilla/sessions"
)

type WebSocketID interface {
	WriteJSON(v interface{}) error
	comparable
}

type WebSocketStatus[websocketidtype WebSocketID] struct {
	Filter   string
	Cookie   *sessions.Session
	Socketid websocketidtype
	Lock     sync.Mutex
}
type WebSocketStatusManager[websocketidtype WebSocketID] struct {
	WebSocketStatus map[websocketidtype]*WebSocketStatus[websocketidtype]
	lock            sync.RWMutex
}

// key could be *websocket.Conn
func NewWebSocketStatusManager[websocketidtype WebSocketID]() *WebSocketStatusManager[websocketidtype] {
	return &WebSocketStatusManager[websocketidtype]{WebSocketStatus: make(map[websocketidtype]*WebSocketStatus[websocketidtype])}
}

func (m *WebSocketStatusManager[websocketidtype]) Add(key websocketidtype, filter string, cookie *sessions.Session) {
	m.lock.Lock()
	m.WebSocketStatus[key] = &WebSocketStatus[websocketidtype]{Filter: filter, Cookie: cookie, Socketid: key}
	m.lock.Unlock()
}
func (m *WebSocketStatusManager[websocketidtype]) Delete(key websocketidtype) {
	m.lock.Lock()
	delete(m.WebSocketStatus, key)
	m.lock.Unlock()
}

func (m *WebSocketStatusManager[websocketidtype]) Get(key websocketidtype) (*WebSocketStatus[websocketidtype], bool) {
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

func (m *WebSocketStatusManager[websocketidtype]) KeyAndValue() ([]websocketidtype, []*WebSocketStatus[websocketidtype]) {
	socketlist := []websocketidtype{}
	statuslist := []*WebSocketStatus[websocketidtype]{}

	m.lock.RLock()
	for k, v := range m.WebSocketStatus {
		socketlist = append(socketlist, k)
		statuslist = append(statuslist, v)
	}
	m.lock.RUnlock()
	return socketlist, statuslist
}
