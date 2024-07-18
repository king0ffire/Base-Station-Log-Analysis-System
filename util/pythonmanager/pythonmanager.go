package pythonmanager

import (
	"os/exec"
	"sync"
)

type PythonProcessStatus struct {
	Cmd       *exec.Cmd
	Completed bool
	Uid       string
}

var PythonProcessStatusMap = make(map[string]*PythonProcessStatus)
var PythonProcessStatusMapLock sync.RWMutex

func PythonProcessStatusMapAdd(uid string, cmd *exec.Cmd) {
	PythonProcessStatusMapLock.Lock()
	PythonProcessStatusMap[uid] = &PythonProcessStatus{cmd, false, uid}
	PythonProcessStatusMapLock.Unlock()
}

func PythonProcessStatusMapDelete(uid string) {
	PythonProcessStatusMapLock.Lock()
	delete(PythonProcessStatusMap, uid)
	PythonProcessStatusMapLock.Unlock()
}

func PythonProcessStatusMapGet(uid string) (*PythonProcessStatus, bool) {
	PythonProcessStatusMapLock.RLock()
	defer PythonProcessStatusMapLock.RUnlock()
	return PythonProcessStatusMap[uid], true
}
