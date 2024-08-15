// File uid
package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"webapp/service/util"
)

type FileCacheQueue[fileidtype comparable, useruidtype comparable] struct {
	Queue []*FileStatus[fileidtype, useruidtype]
	Lock  sync.RWMutex
}

func (m *FileCacheQueue[fileidtype, useruidtype]) Push(file *FileStatus[fileidtype, useruidtype]) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	m.Queue = append(m.Queue, file)
}

func (m *FileCacheQueue[fileidtype, useruidtype]) Pop() {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	if len(m.Queue) > 0 {
		m.Queue = m.Queue[1:]
	}
}

func (m *FileCacheQueue[fileidtype, useruidtype]) Len() int {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	return len(m.Queue)
}

func (m *FileCacheQueue[fileidtype, useruidtype]) Top() *FileStatus[fileidtype, useruidtype] {
	m.Lock.RLock()
	defer m.Lock.RUnlock()
	if len(m.Queue) > 0 {
		return m.Queue[0]
	}
	return nil
}

func (m *FileCacheQueue[fileidtype, useruidtype]) Delete(uid fileidtype) {
	m.Lock.Lock()
	defer m.Lock.Unlock()
	for i, v := range m.Queue {
		if v.Uid == uid {
			m.Queue = append(m.Queue[:i], m.Queue[i+1:]...)
			return
		}
	}
}

type AnalysisStatus struct {
	State        util.State
	Maxvalue     int
	Currentvalue int
	Lock         sync.Mutex
}

type FileStatus[fileidtype comparable, useruidtype comparable] struct {
	Filename   string
	Uid        fileidtype //newlocation
	Useruid    useruidtype
	Dbgstatus  *AnalysisStatus //no schedule, created, running, finished, failed
	Sctpstatus *AnalysisStatus
}

type FileStatusManager[Keytype comparable, useruidtype comparable] struct {
	Filestatus map[Keytype]*FileStatus[Keytype, useruidtype]
	Lock       sync.RWMutex
}

func FileNameFilter[fileidtype comparable, useruidtype comparable](files []*FileStatus[fileidtype, useruidtype], filter string) []*FileStatus[fileidtype, useruidtype] {
	result := []*FileStatus[fileidtype, useruidtype]{}
	for _, file := range files {
		if strings.Contains(file.Filename, filter) {
			result = append(result, file)
		}
	}
	return result
}
func NewManager[Keytype comparable, useruidtype comparable]() *FileStatusManager[Keytype, useruidtype] {
	return &FileStatusManager[Keytype, useruidtype]{Filestatus: make(map[Keytype]*FileStatus[Keytype, useruidtype])}
}
func NewFileStatus[fileidtype comparable, useruidtype comparable]() *FileStatus[fileidtype, useruidtype] {
	return &FileStatus[fileidtype, useruidtype]{Dbgstatus: &AnalysisStatus{State: util.Noschedule}, Sctpstatus: &AnalysisStatus{State: util.Noschedule}}
}
func (m *FileStatusManager[fileidtype, useruidtype]) Add(fileuid fileidtype, filestatus *FileStatus[fileidtype, useruidtype]) {
	m.Lock.Lock()
	m.Filestatus[fileuid] = filestatus
	m.Lock.Unlock()
}

func (m *FileStatusManager[fileidtype, useruidtype]) Delete(fileuid fileidtype) {
	m.Lock.Lock()
	delete(m.Filestatus, fileuid)
	m.Lock.Unlock()
}

func (m *FileStatusManager[fileidtype, useruidtype]) Get(fileuid fileidtype) (*FileStatus[fileidtype, useruidtype], bool) {
	m.Lock.RLock()
	v, ok := m.Filestatus[fileuid]
	m.Lock.RUnlock()
	return v, ok
}

func (m *FileStatusManager[fileidtype, useruidtype]) Set(fileuid fileidtype, obj *FileStatus[fileidtype, useruidtype]) bool {
	_, ok := m.Get(fileuid)
	if ok {
		m.Lock.Lock()
		m.Filestatus[fileuid] = obj
		m.Lock.Unlock()
		return true
	}
	return false
}

func (m *FileStatusManager[fileidtype, useruidtype]) KeyAndValue() ([]fileidtype, []*FileStatus[fileidtype, useruidtype]) {
	keys := []fileidtype{}
	values := []*FileStatus[fileidtype, useruidtype]{}
	m.Lock.RLock()
	for k, v := range m.Filestatus {
		keys = append(keys, k)
		values = append(values, v)
	}
	m.Lock.RUnlock()
	return keys, values
}

func (m *FileStatusManager[fileidtype, useruidtype]) FilterGetByFilename(filter string) []*FileStatus[fileidtype, useruidtype] {
	filteredfilestatuslist := []*FileStatus[fileidtype, useruidtype]{}
	for _, v := range m.Filestatus {
		if strings.Contains(v.Filename, filter) {
			filteredfilestatuslist = append(filteredfilestatuslist, v)
		}
	}
	return filteredfilestatuslist
}

func DeleteFileFromLocal(uploadpath string, uid string) error {
	err := os.RemoveAll(filepath.Join(uploadpath, uid))
	if err != nil {
		go func() {
			time.Sleep(time.Second * 1)
			err2 := os.RemoveAll(filepath.Join(uploadpath, uid))
			fmt.Println("retried delete file")
			if err2 != nil {
				fmt.Println("retry error:", err2)
			}
		}()
	}
	return err
}
