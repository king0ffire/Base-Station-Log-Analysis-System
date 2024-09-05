package topmanager

import (
	"fmt"
	"os/exec"
	"sync"
	"webapp/service/lowermanager"
	"webapp/util"

	"github.com/sirupsen/logrus"
)

type PythonStatusManager[useridtype comparable, fileidtype comparable] interface {
	Add(*lowermanager.FileStatus[useridtype, fileidtype], string)
	Delete(*lowermanager.FileStatus[useridtype, fileidtype])
	Get(*lowermanager.FileStatus[useridtype, fileidtype]) (map[util.Task]*PythonTaskStatus[useridtype, fileidtype], bool)
	SetState(*lowermanager.FileStatus[useridtype, fileidtype], util.Task, util.State) bool
	Stop(*lowermanager.FileStatus[useridtype, fileidtype])
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
	dbgcmd := exec.Command(util.ConfigMap["python"]["python_path"], "./scripts/dbg_main.py", filelocation, "1")
	idscmd := exec.Command(util.ConfigMap["python"]["python_path"], "./scripts/sctp_main.py", filelocation, "1")
	removecachecmd := exec.Command(util.ConfigMap["python"]["python_path"], "./scripts/remove_cache.py", filelocation, "1")

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
	logrus.Debug("forcestop", fileid)
	cmdstatus, _ := m.Get(fileid)
	if err := cmdstatus[util.Dbg].Cmd.Process.Kill(); err != nil {
		logrus.Debug("kill dbg process error:", err)
	}
	if cmdstatus[util.Sctp].Cmd.Process != nil {
		if err := cmdstatus[util.Sctp].Cmd.Process.Kill(); err != nil {
			logrus.Debug("kill ids process error:", err)
		}
	}
	if cmdstatus[util.Delete].Cmd.Process != nil {
		if err := cmdstatus[util.Delete].Cmd.Process.Kill(); err != nil {
			logrus.Debug("kill removecache process error:", err)
		}
	}
}

type PythonServiceStatusManager[useridtype comparable, fileidtype comparable] struct {
	lock                      sync.RWMutex
	FileTasks                 map[fileidtype]map[util.Task]*PythonTaskStatus[useridtype, fileidtype] //一个file下有三个cmd，这三个cmd对应不同的状态
	PythonServerSocketManager *lowermanager.SocketManager
}

func NewPythonServiceStatusManager[useridtype comparable, fileidtype comparable]() *PythonServiceStatusManager[useridtype, fileidtype] {
	m := &PythonServiceStatusManager[useridtype, fileidtype]{FileTasks: make(map[fileidtype]map[util.Task]*PythonTaskStatus[useridtype, fileidtype])}
	m.PythonServerSocketManager = lowermanager.NewSocketManager()
	return m
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Add(filestatus *lowermanager.FileStatus[useridtype, fileidtype], filelocation string) {
	//dbgcmd := exec.Command("python", "./scripts/dbg_main.py", filelocation, "0")
	idscmd := exec.Command(util.ConfigMap["python"]["python_path"], "./scripts/sctp_main.py", filelocation, "1")
	removecachecmd := exec.Command(util.ConfigMap["python"]["python_path"], "./scripts/remove_cache.py", filelocation, "1")

	m.lock.Lock()
	m.FileTasks[filestatus.Uid] = make(map[util.Task]*PythonTaskStatus[useridtype, fileidtype])
	m.FileTasks[filestatus.Uid][util.Dbg] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Rpc, Cmd: nil, State: util.Idle, Uid: filestatus.Uid, Useruid: filestatus.Useruid}
	m.FileTasks[filestatus.Uid][util.Sctp] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Cmd, Cmd: idscmd, State: util.Idle, Uid: filestatus.Uid, Useruid: filestatus.Useruid}
	m.FileTasks[filestatus.Uid][util.Delete] = &PythonTaskStatus[useridtype, fileidtype]{Calltype: util.Cmd, Cmd: removecachecmd, State: util.Idle, Uid: filestatus.Uid, Useruid: filestatus.Useruid}
	m.lock.Unlock()
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Delete(filestatus *lowermanager.FileStatus[useridtype, fileidtype]) {
	m.lock.Lock()
	delete(m.FileTasks, filestatus.Uid)
	m.lock.Unlock()
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Get(filestatus *lowermanager.FileStatus[useridtype, fileidtype]) (map[util.Task]*PythonTaskStatus[useridtype, fileidtype], bool) {
	m.lock.RLock()
	v, ok := m.FileTasks[filestatus.Uid]
	m.lock.RUnlock()
	return v, ok
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) SetState(filestatus *lowermanager.FileStatus[useridtype, fileidtype], task util.Task, state util.State) bool {
	m.lock.Lock()
	v, ok := m.FileTasks[filestatus.Uid][task]
	if ok {
		v.State = state
	}
	m.lock.Unlock()
	return ok
}
func (m *PythonServiceStatusManager[useridtype, fileidtype]) Stop(filestatus *lowermanager.FileStatus[useridtype, fileidtype]) {
	logrus.Debug("forcing stop:", filestatus.Uid)
	taskstatus, ok := m.Get(filestatus)
	if !ok {
		logrus.Debug("file not found:", filestatus.Uid)
		return
	}
	if taskstatus[util.Dbg].Calltype == util.Cmd && taskstatus[util.Dbg].Cmd.Process != nil {
		if err := taskstatus[util.Dbg].Cmd.Process.Kill(); err != nil {
			logrus.Debug("kill dbg process error:", err)
		}
	} else if taskstatus[util.Dbg].Calltype == util.Rpc {
		var jsondata = make(map[string]interface{})
		jsondata["task"] = fmt.Sprintf("%v", util.Dbg)
		jsondata["action"] = fmt.Sprintf("%v", util.Stop)
		jsondata["fileuid"] = fmt.Sprintf("%v", filestatus.Uid)
		jsondata["useruid"] = fmt.Sprintf("%v", filestatus.Useruid)
		(m.PythonServerSocketManager.Socket).WriteJSON(jsondata)
		filestatus.Dbgstatus.State = util.Terminated
	}
	if taskstatus[util.Sctp].Calltype == util.Cmd && taskstatus[util.Sctp].Cmd.Process != nil {
		if err := taskstatus[util.Sctp].Cmd.Process.Kill(); err != nil {
			logrus.Debug("kill ids process error:", err)
		}
	}
	if taskstatus[util.Delete].Calltype == util.Cmd && taskstatus[util.Delete].Cmd.Process != nil {
		if err := taskstatus[util.Delete].Cmd.Process.Kill(); err != nil {
			logrus.Debug("kill removecache process error:", err)
		}
	}
}

func (m *PythonServiceStatusManager[useridtype, fileidtype]) Start(filestatus *lowermanager.FileStatus[useridtype, fileidtype], task util.Task) error {
	currenttask, ok := m.FileTasks[filestatus.Uid][task]
	if !ok {
		logrus.Debug("task not found:", filestatus.Uid, ":", task)
		return fmt.Errorf("task not found when trying start %v", filestatus.Uid)
	}
	if currenttask.Calltype == util.Cmd {
		return currenttask.Cmd.Start()
	} else if currenttask.Calltype == util.Rpc {
		var jsondata = make(map[string]interface{})
		jsondata["task"] = fmt.Sprintf("%v", task)
		jsondata["fileuid"] = fmt.Sprintf("%v", filestatus.Uid)
		jsondata["useruid"] = fmt.Sprintf("%v", filestatus.Useruid)
		jsondata["action"] = util.Start
		m.PythonServerSocketManager.Socket.WriteJSON(jsondata)
		logrus.Debug("sent rpc request:", jsondata)
	}
	return nil
}
