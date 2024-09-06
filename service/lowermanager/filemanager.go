package lowermanager

import (
	"strings"
	"sync"
	"webapp/service/models"
	"webapp/util"
)

type FileStatusManager[useruidtype comparable, fileidtype comparable] struct {
	Filestatus map[fileidtype]*models.FileStatus[useruidtype, fileidtype]
	lock       sync.RWMutex
}

func FileNameFilter[useruidtype comparable, fileidtype comparable](files []*models.FileStatus[useruidtype, fileidtype], filter string) []*models.FileStatus[useruidtype, fileidtype] {
	result := []*models.FileStatus[useruidtype, fileidtype]{}
	for _, file := range files {
		if strings.Contains(file.Filename, filter) {
			result = append(result, file)
		}
	}
	return result
}
func NewFileStatusManager[useruidtype comparable, fileidtype comparable]() *FileStatusManager[useruidtype, fileidtype] {
	return &FileStatusManager[useruidtype, fileidtype]{Filestatus: make(map[fileidtype]*models.FileStatus[useruidtype, fileidtype])}
}
func NewFileStatus[useruidtype comparable, fileidtype comparable]() *models.FileStatus[useruidtype, fileidtype] {
	return &models.FileStatus[useruidtype, fileidtype]{Dbgstatus: &models.AnalysisStatus{State: util.Created}, Sctpstatus: &models.AnalysisStatus{State: util.Created}}
}
func (m *FileStatusManager[useruidtype, fileidtype]) Add(fileuid fileidtype, filestatus *models.FileStatus[useruidtype, fileidtype]) {
	m.lock.Lock()
	m.Filestatus[fileuid] = filestatus
	m.lock.Unlock()
}

func (m *FileStatusManager[useruidtype, fileidtype]) Delete(fileuid fileidtype) {
	m.lock.Lock()
	delete(m.Filestatus, fileuid)
	m.lock.Unlock()
}

func (m *FileStatusManager[useruidtype, fileidtype]) Get(fileuid fileidtype) (*models.FileStatus[useruidtype, fileidtype], bool) {
	m.lock.RLock()
	v, ok := m.Filestatus[fileuid]
	m.lock.RUnlock()
	return v, ok
}

func (m *FileStatusManager[useruidtype, fileidtype]) Set(fileuid fileidtype, obj *models.FileStatus[useruidtype, fileidtype]) bool {
	_, ok := m.Get(fileuid)
	if ok {
		m.lock.Lock()
		m.Filestatus[fileuid] = obj
		m.lock.Unlock()
		return true
	}
	return false
}

func (m *FileStatusManager[useruidtype, fileidtype]) KeyAndValue() ([]fileidtype, []*models.FileStatus[useruidtype, fileidtype]) {
	keys := []fileidtype{}
	values := []*models.FileStatus[useruidtype, fileidtype]{}
	m.lock.RLock()
	for k, v := range m.Filestatus {
		keys = append(keys, k)
		values = append(values, v)
	}
	m.lock.RUnlock()
	return keys, values
}

func (m *FileStatusManager[useruidtype, fileidtype]) FilterGetByFilename(filter string) []*models.FileStatus[useruidtype, fileidtype] {
	filteredfilestatuslist := []*models.FileStatus[useruidtype, fileidtype]{}
	for _, v := range m.Filestatus {
		if strings.Contains(v.Filename, filter) {
			filteredfilestatuslist = append(filteredfilestatuslist, v)
		}
	}
	return filteredfilestatuslist
}
