package pythonmanager

import (
	"fmt"
	"os/exec"
	"sync"
	"webapp/service/file"
	"webapp/service/socket"
	"webapp/service/util"
)

type PythonStatusManager[useridtype comparable, fileidtype comparable] interface {
	Add(fileuid fileidtype, filelocation string)
	Delete(fileid fileidtype)
	Get(uid fileidtype) (map[util.Task]*PythonTaskStatus[fileidtype], bool)
	SetState(uid fileidtype, cmdname util.Task, state util.State) bool
	Forcestop(fileid fileidtype)
	Start(filestatus *file.FileStatus[fileidtype, useridtype], task util.Task) error
}

type PythonTaskStatus[fileidtype comparable] struct {
	Calltype    util.Calltype //"cmd", "rpc"
	Task        util.Task
	State       util.State // idle, running
	Cmd         *exec.Cmd
	Uid         fileidtype
	Createdtime string
}

type PythonCmdStatusManager[fileidtype comparable] struct {
	Lock         sync.RWMutex
	PythonStatus map[fileidtype]map[util.Task]*PythonTaskStatus[fileidtype] //一个file下有三个cmd，这三个cmd对应不同的状态
}

func NewManager[fileidtype comparable]() *PythonCmdStatusManager[fileidtype] {
	return &PythonCmdStatusManager[fileidtype]{PythonStatus: make(map[fileidtype]map[util.Task]*PythonTaskStatus[fileidtype])}
}

func (m *PythonCmdStatusManager[fileidtype]) Add(fileuid fileidtype, filelocation string) {
	dbgcmd := exec.Command("python", "./scripts/dbg_main.py", filelocation, "1")
	idscmd := exec.Command("python", "./scripts/sctp_main.py", filelocation, "1")
	removecachecmd := exec.Command("python", "./scripts/remove_cache.py", filelocation, "1")

	m.Lock.Lock()
	m.PythonStatus[fileuid] = make(map[util.Task]*PythonTaskStatus[fileidtype])
	m.PythonStatus[fileuid][util.Dbg] = &PythonTaskStatus[fileidtype]{Calltype: util.Cmd, Cmd: dbgcmd, State: util.Idle, Uid: fileuid}
	m.PythonStatus[fileuid][util.Sctp] = &PythonTaskStatus[fileidtype]{Calltype: util.Cmd, Cmd: idscmd, State: util.Idle, Uid: fileuid}
	m.PythonStatus[fileuid][util.Delete] = &PythonTaskStatus[fileidtype]{Calltype: util.Cmd, Cmd: removecachecmd, State: util.Idle, Uid: fileuid}
	m.Lock.Unlock()
}

func (m *PythonCmdStatusManager[fileidtype]) Delete(fileid fileidtype) {
	m.Forcestop(fileid)
	m.Lock.Lock()
	delete(m.PythonStatus, fileid)
	m.Lock.Unlock()
}

func (m *PythonCmdStatusManager[fileidtype]) Get(uid fileidtype) (map[util.Task]*PythonTaskStatus[fileidtype], bool) {
	m.Lock.RLock()
	v, ok := m.PythonStatus[uid]
	m.Lock.RUnlock()
	return v, ok
}

func (m *PythonCmdStatusManager[fileidtype]) SetState(uid fileidtype, cmdname util.Task, state util.State) bool {
	m.Lock.Lock()
	v, ok := m.PythonStatus[uid][cmdname]
	if ok {
		v.State = state
	}
	m.Lock.Unlock()
	return ok
}
func (m *PythonCmdStatusManager[fileidtype]) Forcestop(fileid fileidtype) {
	fmt.Println("forcestop", fileid)
	cmdstatus, _ := m.Get(fileid)
	if err := cmdstatus[util.Dbg].Cmd.Process.Kill(); err != nil {
		fmt.Println("kill dbg process error:", err)
	}
	if cmdstatus[util.Sctp].Cmd.Process != nil {
		if err := cmdstatus[util.Sctp].Cmd.Process.Kill(); err != nil {
			fmt.Println("kill ids process error:", err)
		}
	}
	if cmdstatus[util.Delete].Cmd.Process != nil {
		if err := cmdstatus[util.Delete].Cmd.Process.Kill(); err != nil {
			fmt.Println("kill removecache process error:", err)
		}
	}
}

type PythonServiceStatusManager[useridtype comparable, fileidtype comparable] struct {
	Lock         sync.RWMutex
	PythonStatus map[fileidtype]map[util.Task]*PythonTaskStatus[fileidtype] //一个file下有三个cmd，这三个cmd对应不同的状态
	PythonServer *socket.PythonServerBySocket
}

func NewServerManager[useridtype comparable, fileidtype comparable]() *PythonServiceStatusManager[useridtype, fileidtype] {
	m := &PythonServiceStatusManager[useridtype, fileidtype]{PythonStatus: make(map[fileidtype]map[util.Task]*PythonTaskStatus[fileidtype])}
	m.PythonServer = socket.NewSocket()

	return m
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Add(fileuid fileidtype, filelocation string) {
	//dbgcmd := exec.Command("python", "./scripts/dbg_main.py", filelocation, "0")
	idscmd := exec.Command("python", "./scripts/sctp_main.py", filelocation, "1")
	removecachecmd := exec.Command("python", "./scripts/remove_cache.py", filelocation, "1")

	m.Lock.Lock()
	m.PythonStatus[fileuid] = make(map[util.Task]*PythonTaskStatus[fileidtype])
	m.PythonStatus[fileuid][util.Dbg] = &PythonTaskStatus[fileidtype]{Calltype: util.Rpc, Cmd: nil, State: util.Idle, Uid: fileuid}
	m.PythonStatus[fileuid][util.Sctp] = &PythonTaskStatus[fileidtype]{Calltype: util.Cmd, Cmd: idscmd, State: util.Idle, Uid: fileuid}
	m.PythonStatus[fileuid][util.Delete] = &PythonTaskStatus[fileidtype]{Calltype: util.Cmd, Cmd: removecachecmd, State: util.Idle, Uid: fileuid}
	m.Lock.Unlock()
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Delete(fileid fileidtype) {
	m.Forcestop(fileid)
	m.Lock.Lock()
	delete(m.PythonStatus, fileid)
	m.Lock.Unlock()
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Get(uid fileidtype) (map[util.Task]*PythonTaskStatus[fileidtype], bool) {
	m.Lock.RLock()
	v, ok := m.PythonStatus[uid]
	m.Lock.RUnlock()
	return v, ok
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) SetState(uid fileidtype, cmdname util.Task, state util.State) bool {
	m.Lock.Lock()
	v, ok := m.PythonStatus[uid][cmdname]
	if ok {
		v.State = state
	}
	m.Lock.Unlock()
	return ok
}
func (m *PythonServiceStatusManager[useridtype, fileidtype]) Forcestop(fileid fileidtype) {
	fmt.Println("forcestop", fileid)
	taskstatus, _ := m.Get(fileid)

	if taskstatus[util.Dbg].Calltype == util.Cmd && taskstatus[util.Dbg].Cmd.Process != nil {
		if err := taskstatus[util.Dbg].Cmd.Process.Kill(); err != nil {
			fmt.Println("kill dbg process error:", err)
		}
	}
	if taskstatus[util.Sctp].Calltype == util.Cmd && taskstatus[util.Sctp].Cmd.Process != nil {
		if err := taskstatus[util.Sctp].Cmd.Process.Kill(); err != nil {
			fmt.Println("kill ids process error:", err)
		}
	}
	if taskstatus[util.Delete].Calltype == util.Cmd && taskstatus[util.Delete].Cmd.Process != nil {
		if err := taskstatus[util.Delete].Cmd.Process.Kill(); err != nil {
			fmt.Println("kill removecache process error:", err)
		}
	}
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Start(filestatus *file.FileStatus[fileidtype, useridtype], taskname util.Task) error {
	currenttask := m.PythonStatus[filestatus.Uid][taskname]
	if currenttask.Calltype == util.Cmd {
		return currenttask.Cmd.Start()
	} else if currenttask.Calltype == util.Rpc {
		var jsondata = make(map[string]interface{})
		jsondata["function"] = fmt.Sprintf("%v", taskname)
		jsondata["fileuid"] = fmt.Sprintf("%v", filestatus.Uid)
		jsondata["useruid"] = fmt.Sprintf("%v", filestatus.Useruid)
		m.PythonServer.WriteJSON(jsondata)
		fmt.Println("sent rpc request:", jsondata)
	}
	return nil
}
