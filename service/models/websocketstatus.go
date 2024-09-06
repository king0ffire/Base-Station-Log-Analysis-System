package models

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
