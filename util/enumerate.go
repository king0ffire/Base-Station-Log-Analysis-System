package util

import "encoding/json"

type State int

const (
	Running State = iota
	Idle

	Created
	Terminated
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

type Action int

const (
	Start Action = iota
	Stop
)

func (s State) String() string {
	switch s {
	case Running:
		return "Running"
	case Idle:
		return "Idle"
	case Created:
		return "Created"
	case Finished:
		return "Finished"
	case Failed:
		return "Failed"
	case Terminated:
		return "Terminated"
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

func (a Action) String() string {
	switch a {
	case Start:
		return "Start"
	case Stop:
		return "Stop"
	default:
		return "Unknown"
	}
}

func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}
