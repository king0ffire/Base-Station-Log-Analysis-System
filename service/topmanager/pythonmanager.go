package topmanager

import (
	"fmt"
	"os/exec"
	"sync"
	"webapp/service/lowermanager"
	"webapp/util"
)

type PythonStatusManager[useridtype comparable, fileidtype comparable] interface {
	Add(*lowermanager.FileStatus[useridtype, fileidtype], string)
	Delete(*lowermanager.FileStatus[useridtype, fileidtype])
	Get(*lowermanager.FileStatus[useridtype, fileidtype]) (map[util.Task]*PythonTaskStatus[useridtype, fileidtype], bool)
	SetState(*lowermanager.FileStatus[useridtype, fileidtype], util.Task, util.State) bool
	Forcestop(*lowermanager.FileStatus[useridtype, fileidtype])
	Start(*lowermanager.FileStatus[useridtype, fileidtype], util.Task) error
}

type PythonTaskStatus[useruidtype comparable, fileidtype comparable] struct {
	Calltype    util.Calltype //"cmd", "rpc"
	Task        util.Task
	State       util.State // idle, running
	Cmd         *exec.Cmd
	Uid         fileidtype
	Useruid     useruidtype
	Createdtime string
}

type PythonCmdStatusManager[useridtype comparable, fileidtype comparable] struct {
	Lock         sync.RWMutex
	PythonStatus map[fileidtype]map[util.Task]*PythonTaskStatus[useridtype, fileidtype] //一个file下有三个cmd，这三个cmd对应不同的状态
}

func NewPythonCmdStatusManager[useridtype comparable, fileidtype comparable]() *PythonCmdStatusManager[useridtype, fileidtype] {
	return &PythonCmdStatusManager[useridtype, fileidtype]{PythonStatus: make(map[fileidtype]map[util.Task]*PythonTaskStatus[useridtype, fileidtype])}
}

func (m *PythonCmdStatusManager[useridtype, fileidtype]) Add(fileuid fileidtype, filelocation string) {
	dbgcmd := exec.Command("python", "./scripts/dbg_main.py", filelocation, "1")
	idscmd := exec.Command("python", "./scripts/sctp_main.py", filelocation, "1")
	removecachecmd := exec.Command("python", "./scripts/remove_cache.py", filelocation, "1")

	m.Lock.Lock()
	m.PythonStatus[fileuid] = make(map[util.Task]*PythonTaskStatus[useridtype, fileidtype])
	m.PythonStatus[fileuid][util.Dbg] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Cmd, Cmd: dbgcmd, State: util.Idle, Uid: fileuid}
	m.PythonStatus[fileuid][util.Sctp] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Cmd, Cmd: idscmd, State: util.Idle, Uid: fileuid}
	m.PythonStatus[fileuid][util.Delete] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Cmd, Cmd: removecachecmd, State: util.Idle, Uid: fileuid}
	m.Lock.Unlock()
}

func (m *PythonCmdStatusManager[useridtype, fileidtype]) Delete(fileid fileidtype) {
	m.Forcestop(fileid)
	m.Lock.Lock()
	delete(m.PythonStatus, fileid)
	m.Lock.Unlock()
}

func (m *PythonCmdStatusManager[useridtype, fileidtype]) Get(uid fileidtype) (map[util.Task]*PythonTaskStatus[useridtype, fileidtype], bool) {
	m.Lock.RLock()
	v, ok := m.PythonStatus[uid]
	m.Lock.RUnlock()
	return v, ok
}

func (m *PythonCmdStatusManager[useridtype, fileidtype]) SetState(uid fileidtype, cmdname util.Task, state util.State) bool {
	m.Lock.Lock()
	v, ok := m.PythonStatus[uid][cmdname]
	if ok {
		v.State = state
	}
	m.Lock.Unlock()
	return ok
}
func (m *PythonCmdStatusManager[useridtype, fileidtype]) Forcestop(fileid fileidtype) {
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
	PythonStatus map[fileidtype]map[util.Task]*PythonTaskStatus[useridtype, fileidtype] //一个file下有三个cmd，这三个cmd对应不同的状态
	PythonServer *PythonServerSocketManager
}

func NewPythonServiceStatusManager[useridtype comparable, fileidtype comparable]() *PythonServiceStatusManager[useridtype, fileidtype] {
	m := &PythonServiceStatusManager[useridtype, fileidtype]{PythonStatus: make(map[fileidtype]map[util.Task]*PythonTaskStatus[useridtype, fileidtype])}
	m.PythonServer = NewPythonServerSocketManager()
	return m
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Add(useruid useridtype, fileuid fileidtype, filelocation string) {
	//dbgcmd := exec.Command("python", "./scripts/dbg_main.py", filelocation, "0")
	idscmd := exec.Command("python", "./scripts/sctp_main.py", filelocation, "1")
	removecachecmd := exec.Command("python", "./scripts/remove_cache.py", filelocation, "1")

	m.Lock.Lock()
	m.PythonStatus[fileuid] = make(map[util.Task]*PythonTaskStatus[useridtype, fileidtype])
	m.PythonStatus[fileuid][util.Dbg] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Rpc, Cmd: nil, State: util.Idle, Uid: fileuid, Useruid: useruid}
	m.PythonStatus[fileuid][util.Sctp] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Cmd, Cmd: idscmd, State: util.Idle, Uid: fileuid, Useruid: useruid}
	m.PythonStatus[fileuid][util.Delete] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Cmd, Cmd: removecachecmd, State: util.Idle, Uid: fileuid, Useruid: useruid}
	m.Lock.Unlock()
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Delete(useruid useridtype, fileuid fileidtype) {
	m.Forcestop(useruid, fileuid)
	m.Lock.Lock()
	delete(m.PythonStatus, fileuid)
	m.Lock.Unlock()
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Get(uid fileidtype) (map[util.Task]*PythonTaskStatus[useridtype, fileidtype], bool) {
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
func (m *PythonServiceStatusManager[useridtype, fileidtype]) Forcestop(useruid useridtype, fileuid fileidtype) {
	fmt.Println("forcing stop:", fileuid)
	taskstatus, ok := m.Get(fileuid)
	if !ok {
		fmt.Println("file not found:", fileuid)
		return
	}
	if taskstatus[util.Dbg].Calltype == util.Cmd && taskstatus[util.Dbg].Cmd.Process != nil {
		if err := taskstatus[util.Dbg].Cmd.Process.Kill(); err != nil {
			fmt.Println("kill dbg process error:", err)
		}
	} else if taskstatus[util.Dbg].Calltype == util.Rpc {
		var jsondata = make(map[string]interface{})
		jsondata["function"] = fmt.Sprintf("Stop:%v", util.Dbg)
		jsondata["fileuid"] = fmt.Sprintf("%v", fileuid)
		jsondata["useruid"] = fmt.Sprintf("%v", useruid)
		m.PythonServer.WriteJSON(jsondata)
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

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Start(useruid useridtype, fileuid fileidtype, taskname util.Task) error {
	currenttask := m.PythonStatus[fileuid][taskname]
	if currenttask.Calltype == util.Cmd {
		return currenttask.Cmd.Start()
	} else if currenttask.Calltype == util.Rpc {
		var jsondata = make(map[string]interface{})
		jsondata["function"] = fmt.Sprintf("%v", taskname)
		jsondata["fileuid"] = fmt.Sprintf("%v", fileuid)
		jsondata["useruid"] = fmt.Sprintf("%v", useruid)
		m.PythonServer.WriteJSON(jsondata)
		fmt.Println("sent rpc request:", jsondata)
	}
	return nil
}
