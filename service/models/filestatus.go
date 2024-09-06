package models

import (
	"sync"
	"webapp/util"
)

type AnalysisStatus struct {
	State        util.State
	Maxvalue     int
	Currentvalue int
	Lock         sync.Mutex
}

type FileStatus[useruidtype comparable, fileidtype comparable] struct {
	Filename   string
	Uid        fileidtype //newlocation
	Useruid    useruidtype
	Dbgstatus  *AnalysisStatus //no schedule, created, running, finished, failed
	Sctpstatus *AnalysisStatus
}
