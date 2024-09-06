package models

import (
	"os/exec"
	"webapp/util"
)

type PythonTaskStatus[useruidtype comparable, fileidtype comparable] struct {
	Calltype    util.Calltype //"cmd", "rpc"
	Task        util.Task
	State       util.State // idle, running
	Cmd         *exec.Cmd
	Uid         fileidtype
	Useruid     useruidtype
	Createdtime string
}
