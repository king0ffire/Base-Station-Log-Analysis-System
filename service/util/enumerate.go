package util

import "encoding/json"

type State int

const (
	Running State = iota
	Idle
	Noschedule
	Created
	Finished
	Failed
)

type Calltype int

const (
	Cmd Calltype = iota
	Rpc
)

type Task int

const (
	Dbg Task = iota
	Sctp
	Delete
)

func (s State) String() string {
	switch s {
	case Running:
		return "Running"
	case Idle:
		return "Idle"
	case Noschedule:
		return "Noschedule"
	case Created:
		return "Created"
	case Finished:
		return "Finished"
	case Failed:
		return "Failed"
	default:
		return "Unknown"
	}
}
func (s State) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}
func (c Calltype) String() string {
	switch c {
	case Cmd:
		return "Cmd"
	case Rpc:
		return "Rpc"
	default:
		return "Unknown"
	}
}
func (c Calltype) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}
func (t Task) String() string {
	switch t {
	case Dbg:
		return "Dbg"
	case Sctp:
		return "Sctp"
	case Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}
func (t Task) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}
