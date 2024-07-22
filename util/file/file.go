// File uid
package file

import (
	"strings"
	"sync"
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

type FileStatus[fileidtype comparable, useruidtype comparable] struct {
	Filename string
	Uid      fileidtype //newlocation
	Max      int
	Current  int
	Useruid  useruidtype
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

func (m *FileStatusManager[Keytype, useruidtype]) Add(uid Keytype, filename string, current int, max int, userid useruidtype) {
	m.Lock.Lock()
	m.Filestatus[uid] = &FileStatus[Keytype, useruidtype]{Filename: filename, Uid: uid, Current: current, Max: max, Useruid: userid}
	m.Lock.Unlock()
}

func (m *FileStatusManager[Keytype, useruidtype]) Delete(uid Keytype) {
	m.Lock.Lock()
	delete(m.Filestatus, uid)
	m.Lock.Unlock()
}

func (m *FileStatusManager[Keytype, useruidtype]) Get(uid Keytype) (*FileStatus[Keytype, useruidtype], bool) {
	m.Lock.RLock()
	v, ok := m.Filestatus[uid]
	m.Lock.RUnlock()
	return v, ok
}

func (m *FileStatusManager[Keytype, useruidtype]) Set(uid Keytype, current int, max int, useruid useruidtype) bool {
	oldvalue, ok := m.Get(uid)
	if ok {
		m.Lock.Lock()
		oldvalue.Current = current
		oldvalue.Max = max
		oldvalue.Useruid = useruid
		m.Lock.Unlock()
		return true
	}
	return false
}

func (m *FileStatusManager[Keytype, useruidtype]) KeyAndValue() ([]Keytype, []*FileStatus[Keytype, useruidtype]) {
	keys := []Keytype{}
	values := []*FileStatus[Keytype, useruidtype]{}
	m.Lock.RLock()
	for k, v := range m.Filestatus {
		keys = append(keys, k)
		values = append(values, v)
	}
	m.Lock.RUnlock()
	return keys, values
}

func (m *FileStatusManager[Keytype, useruidtype]) FilterGetByFilename(filter string) []*FileStatus[Keytype, useruidtype] {
	filteredfilestatuslist := []*FileStatus[Keytype, useruidtype]{}
	for _, v := range m.Filestatus {
		if strings.Contains(v.Filename, filter) {
			filteredfilestatuslist = append(filteredfilestatuslist, v)
		}
	}
	return filteredfilestatuslist
}
