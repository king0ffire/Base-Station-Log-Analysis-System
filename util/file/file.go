package file

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileStatus struct {
	Filename string
	Uid      string //newlocation
	Max      int
	Current  int
}

var fileStatusMap = make(map[string]*FileStatus) //uid to file
var fileStatusMapLock sync.RWMutex

func MultiPartFileSaver(savepath string, file *multipart.File, handler *multipart.FileHeader) (string, bool) {
	uid := strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err := os.Stat(filepath.Join(savepath, uid+".tar.gz"))
	fmt.Println("newfile path:", filepath.Join(savepath, uid+".tar.gz"))
	if err != nil {
		cachefile, err := os.Create(filepath.Join(savepath, uid+".tar.gz"))
		if err != nil {
			fmt.Println("create new fail:", err)
			return "", false
		}
		defer cachefile.Close()

		_, err = cachefile.ReadFrom(*file)
		if err != nil {
			fmt.Println("cache fail:", err)
			return "", false
		}

		FileStatusMapAdd(uid, handler.Filename, 0, 0)
		return uid, true
	} else {
		fmt.Println("file already exists, fatal error, skip parsing")
		return "", false
		/*
			max, current, existence := util.CheckFileExist(handler.Filename)
			if !existence {
				fmt.Println("file created before web init")
				util.FileStatusMapLock.Lock()
				util.FileStatusMap[handler.Filename] = &util.FileStatus{Filename: handler.Filename, Max: max, Current: current}
				util.FileStatusMapLock.Unlock()
			}*/
	}

}
func FileStatusMapNameFilter(filter string) []*FileStatus {
	filteredfilestatuslist := []*FileStatus{}
	for _, v := range fileStatusMap {
		if strings.Contains(v.Filename, filter) {
			filteredfilestatuslist = append(filteredfilestatuslist, v)
		}
	}
	return filteredfilestatuslist
}

func FileNameFilter(files []*FileStatus, filter string) []*FileStatus {
	result := []*FileStatus{}
	for _, file := range files {
		if strings.Contains(file.Filename, filter) {
			result = append(result, file)
		}
	}
	return result
}

func FileStatusMapSet(uid string, current int, max int) bool {
	oldvalue, ok := FileStatusMapGet(uid)
	if ok {
		fileStatusMapLock.Lock()
		oldvalue.Current = current
		oldvalue.Max = max
		fileStatusMapLock.Unlock()
		return true
	}
	return false
}

func FileStatusMapGet(uid string) (*FileStatus, bool) {
	fileStatusMapLock.RLock()
	v, ok := fileStatusMap[uid]
	fileStatusMapLock.RUnlock()
	return v, ok
}

func FileStatusMapDelete(uid string) {
	fileStatusMapLock.Lock()
	delete(fileStatusMap, uid)
	fileStatusMapLock.Unlock()
}

func FileStatusMapAdd(uid string, filename string, current int, max int) {
	value := &FileStatus{Filename: filename, Uid: uid, Current: current, Max: max}
	fileStatusMapLock.Lock()
	fileStatusMap[uid] = value
	fileStatusMapLock.Unlock()
}
