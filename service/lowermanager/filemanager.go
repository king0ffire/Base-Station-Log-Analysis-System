package lowermanager

import (
	"strings"
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

type FileStatusManager[useruidtype comparable, fileidtype comparable] struct {
	Filestatus map[fileidtype]*FileStatus[useruidtype, fileidtype]
	Lock       sync.RWMutex
}

func FileNameFilter[useruidtype comparable, fileidtype comparable](files []*FileStatus[useruidtype, fileidtype], filter string) []*FileStatus[useruidtype, fileidtype] {
	result := []*FileStatus[useruidtype, fileidtype]{}
	for _, file := range files {
		if strings.Contains(file.Filename, filter) {
			result = append(result, file)
		}
	}
	return result
}
func NewFileStatusManager[useruidtype comparable, fileidtype comparable]() *FileStatusManager[useruidtype, fileidtype] {
	return &FileStatusManager[useruidtype, fileidtype]{Filestatus: make(map[fileidtype]*FileStatus[useruidtype, fileidtype])}
}
func NewFileStatus[useruidtype comparable, fileidtype comparable]() *FileStatus[useruidtype, fileidtype] {
	return &FileStatus[useruidtype, fileidtype]{Dbgstatus: &AnalysisStatus{State: util.Noschedule}, Sctpstatus: &AnalysisStatus{State: util.Noschedule}}
}
func (m *FileStatusManager[useruidtype, fileidtype]) Add(fileuid fileidtype, filestatus *FileStatus[useruidtype, fileidtype]) {
	m.Lock.Lock()
	m.Filestatus[fileuid] = filestatus
	m.Lock.Unlock()
}

func (m *FileStatusManager[useruidtype, fileidtype]) Delete(fileuid fileidtype) {
	m.Lock.Lock()
	delete(m.Filestatus, fileuid)
	m.Lock.Unlock()
}

func (m *FileStatusManager[useruidtype, fileidtype]) Get(fileuid fileidtype) (*FileStatus[useruidtype, fileidtype], bool) {
	m.Lock.RLock()
	v, ok := m.Filestatus[fileuid]
	m.Lock.RUnlock()
	return v, ok
}

func (m *FileStatusManager[useruidtype, fileidtype]) Set(fileuid fileidtype, obj *FileStatus[useruidtype, fileidtype]) bool {
	_, ok := m.Get(fileuid)
	if ok {
		m.Lock.Lock()
		m.Filestatus[fileuid] = obj
		m.Lock.Unlock()
		return true
	}
	return false
}

func (m *FileStatusManager[useruidtype, fileidtype]) KeyAndValue() ([]fileidtype, []*FileStatus[useruidtype, fileidtype]) {
	keys := []fileidtype{}
	values := []*FileStatus[useruidtype, fileidtype]{}
	m.Lock.RLock()
	for k, v := range m.Filestatus {
		keys = append(keys, k)
		values = append(values, v)
	}
	m.Lock.RUnlock()
	return keys, values
}

func (m *FileStatusManager[useruidtype, fileidtype]) FilterGetByFilename(filter string) []*FileStatus[useruidtype, fileidtype] {
	filteredfilestatuslist := []*FileStatus[useruidtype, fileidtype]{}
	for _, v := range m.Filestatus {
		if strings.Contains(v.Filename, filter) {
			filteredfilestatuslist = append(filteredfilestatuslist, v)
		}
	}
	return filteredfilestatuslist
}
