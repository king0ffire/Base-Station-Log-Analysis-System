package socket

import (
	"sync"

	"github.com/gorilla/sessions"
	"github.com/gorilla/websocket"
)

type SocketStatus struct {
	Filter  string
	Session *sessions.Session
}

var socketStatusManager = make(map[*websocket.Conn]*SocketStatus)
var socketStatusManagerLock sync.RWMutex

func SocketManagerAdd(filter string, session *sessions.Session, conn *websocket.Conn) {
	socketStatusManagerLock.Lock()
	socketStatusManager[conn] = &SocketStatus{Filter: filter, Session: session}
	socketStatusManagerLock.Unlock()
}
func SocketManagerDelete(conn *websocket.Conn) {
	socketStatusManagerLock.Lock()
	delete(socketStatusManager, conn)
	socketStatusManagerLock.Unlock()
}

func SocketManagerGetsAll() ([]*websocket.Conn, []*SocketStatus) {
	socketlist := []*websocket.Conn{}
	statuslist := []*SocketStatus{}

	socketStatusManagerLock.RLock()
	for k, v := range socketStatusManager {
		statuslist = append(statuslist, v)
		socketlist = append(socketlist, k)
	}
	socketStatusManagerLock.RUnlock()
	return socketlist, statuslist
}

func SocketManagerGet(key *websocket.Conn) (*SocketStatus, bool) {
	socketStatusManagerLock.RLock()
	v, ok := socketStatusManager[key]
	socketStatusManagerLock.RUnlock()
	return v, ok
}

func SocketManagerSetSession(key *websocket.Conn, session *sessions.Session) {
	socketStatusManagerLock.Lock()
	socketStatusManager[key].Session = session
	socketStatusManagerLock.Unlock()
}
