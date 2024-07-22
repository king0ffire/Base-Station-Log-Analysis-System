// SocketStatus map 可以接受任意类型的socket uid，目前直接使用conn的地址
package socket

import (
	"sync"

	"github.com/gorilla/sessions"
)

type Socketidinterface interface {
	WriteJSON(v interface{}) error
	comparable
}

type SocketStatus[socketidtype Socketidinterface] struct {
	Filter   string
	Cookie   *sessions.Session
	Socketid socketidtype
	Lock     sync.Mutex
}
type SocketStatusManager[socketidtype Socketidinterface] struct {
	SocketStatus map[socketidtype]*SocketStatus[socketidtype]
	Lock         sync.RWMutex
}

// key could be *websocket.Conn
func NewManager[socketidtype Socketidinterface]() *SocketStatusManager[socketidtype] {
	return &SocketStatusManager[socketidtype]{SocketStatus: make(map[socketidtype]*SocketStatus[socketidtype])}
}

func (m *SocketStatusManager[socketidtype]) Add(key socketidtype, filter string, cookie *sessions.Session) {
	m.Lock.Lock()
	m.SocketStatus[key] = &SocketStatus[socketidtype]{Filter: filter, Cookie: cookie, Socketid: key}
	m.Lock.Unlock()
}
func (m *SocketStatusManager[socketidtype]) Delete(key socketidtype) {
	m.Lock.Lock()
	delete(m.SocketStatus, key)
	m.Lock.Unlock()
}

func (m *SocketStatusManager[socketidtype]) Get(key socketidtype) (*SocketStatus[socketidtype], bool) {
	m.Lock.RLock()
	v, ok := m.SocketStatus[key]
	m.Lock.RUnlock()
	return v, ok
}

func (m *SocketStatusManager[socketidtype]) Set(key socketidtype, cookie *sessions.Session) bool {
	s, ok := m.Get(key)
	if ok {
		m.Lock.Lock()
		s.Cookie = cookie
		m.Lock.Unlock()
		return true
	}
	return false
}

func (m *SocketStatusManager[socketidtype]) KeyAndValue() ([]socketidtype, []*SocketStatus[socketidtype]) {
	socketlist := []socketidtype{}
	statuslist := []*SocketStatus[socketidtype]{}

	m.Lock.RLock()
	for k, v := range m.SocketStatus {
		socketlist = append(socketlist, k)
		statuslist = append(statuslist, v)
	}
	m.Lock.RUnlock()
	return socketlist, statuslist
}
