package pythonmanager

import (
	"fmt"
	"os/exec"
	"sync"
)

type PythonProcessStatus[fileidtype comparable] struct {
	Cmd   *exec.Cmd
	State string // created running completed failed
	Uid   fileidtype
}

type PythonProcessStatusManager[fileidtype comparable] struct {
	Lock         sync.RWMutex
	PythonStatus map[fileidtype]*PythonProcessStatus[fileidtype]
}

func NewManager[fileidtype comparable]() *PythonProcessStatusManager[fileidtype] {
	return &PythonProcessStatusManager[fileidtype]{PythonStatus: make(map[fileidtype]*PythonProcessStatus[fileidtype])}
}

func (m *PythonProcessStatusManager[fileidtype]) Add(uid fileidtype, cmd *exec.Cmd) {
	m.Lock.Lock()
	m.PythonStatus[uid] = &PythonProcessStatus[fileidtype]{cmd, "created", uid}
	m.Lock.Unlock()
}

func (m *PythonProcessStatusManager[fileidtype]) Delete(fileid fileidtype) {
	m.Forcestop(fileid)
	m.Lock.Lock()
	delete(m.PythonStatus, fileid)
	m.Lock.Unlock()
}

func (m *PythonProcessStatusManager[fileidtype]) Get(uid fileidtype) (*PythonProcessStatus[fileidtype], bool) {
	m.Lock.RLock()
	v, ok := m.PythonStatus[uid]
	m.Lock.RUnlock()
	return v, ok
}

func (m *PythonProcessStatusManager[fileidtype]) Set(uid fileidtype, state string) bool {
	m.Lock.Lock()
	v, ok := m.PythonStatus[uid]
	if ok {
		v.State = state
	}
	m.Lock.Unlock()
	return ok
}
func (m *PythonProcessStatusManager[fileidtype]) Forcestop(fileid fileidtype) {
	fmt.Println("forcestop", fileid)
	cmdstatus, _ := m.Get(fileid)
	if err := cmdstatus.Cmd.Process.Kill(); err != nil {
		fmt.Println("kill python process:", err)
	}
}
