package util

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
	"webapp/util/file"
	"webapp/util/pythonmanager"
	"webapp/util/session"
	"webapp/util/socket"
)

func FileListNameFilter(FileList []string, filter string) []string {
	result := []string{}
	for _, v := range FileList {
		if strings.Contains(v, filter) {
			result = append(result, v)
		}
	}
	return result
}

/*
	func StringListToMapValue(StringList []string, TargetMap map[string]*file.FileStatus) []*file.FileStatus {
		result := []*file.FileStatus{}
		for _, v := range StringList {
			result = append(result, TargetMap[v])
		}
		return result
	}
*/

func OldFileCollection[fileidtype comparable, useruidtype socket.Socketidinterface](
	sessionmanager *session.SessionStatusManager[fileidtype, fileidtype, useruidtype],
	pythonprocessesmanager *pythonmanager.PythonProcessStatusManager[fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, fileidtype],
	fileuid fileidtype, uploadpath string, userid fileidtype) {
	if cachequeue.Len() > 99 {
		fmt.Println("deleting old file")
		filetobedelete := cachequeue.Top()
		pythonprocessesmanager.Delete(filetobedelete.Uid)
		cachequeue.Pop()
		usersession, ok := sessionmanager.Get(filetobedelete.Useruid)
		if !ok {
			fmt.Println("didnot find the user of file to be deleted")
			return
		}
		usersession.FileStatusManager.Delete(filetobedelete.Uid)
		DeleteFileFromLocal(uploadpath, fmt.Sprintf("%v", filetobedelete.Uid))
	}
}

func AddFileToMemory[sessionidtype comparable, fileidtype comparable, socketidtype socket.Socketidinterface](
	sessionmanager *session.SessionStatusManager[sessionidtype, fileidtype, socketidtype], pythonprocessesmanager *pythonmanager.PythonProcessStatusManager[fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, sessionidtype],
	fileuid fileidtype, filename string, current int, max int, userid sessionidtype) {

	sessionmanager.AddFile(userid, fileuid, filename, 0, 0, userid)
	sess, ok := sessionmanager.Get(userid)
	if !ok {
		fmt.Println("didnot find the user")
		return
	}
	f, ok := sess.FileStatusManager.Get(fileuid)
	if !ok {
		fmt.Println("didnot find the file")
		return
	}
	cachequeue.Push(f)
}

func ParseFile[sessionidtype comparable, fileidtype comparable, socketidtype socket.Socketidinterface](
	sessionmanager *session.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonprocessesmanager *pythonmanager.PythonProcessStatusManager[fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, sessionidtype], fileuid fileidtype, uploadpath string, userid sessionidtype) {
	stringuid := fmt.Sprintf("%v", fileuid)
	pythoncmd := exec.Command("python", "./scripts/main.py", filepath.Join(uploadpath, stringuid+".tar.gz"), "1")
	outpipe, _ := pythoncmd.StdoutPipe()
	pythonprocessesmanager.Add(fileuid, pythoncmd)
	if err := pythoncmd.Start(); err != nil {
		fmt.Println("start python fail:", err)
		return
	}
	pythonprocessesmanager.Set(fileuid, "running")

	scanner := bufio.NewScanner(outpipe)

	go func() {
		fmt.Println("Go thread start:", stringuid)
		defer fmt.Println("Go thread end:", stringuid)
		scanner.Scan()
		fmt.Println("Received first line of exection:", stringuid, ":", scanner.Text())
		intText, err := strconv.Atoi(scanner.Text())
		if err != nil {
			fmt.Println("text conversion:", err)
			pythonprocessesmanager.Set(fileuid, "failed")
			return
		}

		usersession, ok := sessionmanager.Get(userid)
		if !ok {
			fmt.Println("didnot find the user")
			pythonprocessesmanager.Set(fileuid, "failed")
			return
		}
		currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
		if !ok {
			fmt.Println("didnot find the file")
			pythonprocessesmanager.Set(fileuid, "failed")
			return
		}
		ok = usersession.FileStatusManager.Set(currentfilestatus.Uid, 0, intText, currentfilestatus.Useruid)
		if !ok {
			pythonprocessesmanager.Delete(fileuid)
			cachequeue.Delete(fileuid)
			DeleteFileFromLocal(uploadpath, fmt.Sprintf("%v", fileuid))
			return
		}
		_, filestatus := usersession.FileStatusManager.KeyAndValue()
		_, socketstatus := usersession.SocketstatusManager.KeyAndValue()
		AnnounceAllSocketsInUser(filestatus, socketstatus)
		for scanner.Scan() {
			fmt.Println("python outputs:", scanner.Text())
			if scanner.Text() == "sctp_finished_one" {
				ok := usersession.FileStatusManager.Set(currentfilestatus.Uid, currentfilestatus.Current+1, currentfilestatus.Max, currentfilestatus.Useruid)
				if !ok { //异常退出
					pythonprocessesmanager.Delete(fileuid)
					cachequeue.Delete(fileuid)
					DeleteFileFromLocal(uploadpath, fmt.Sprintf("%v", fileuid))
					return
				}
				_, filestatus := usersession.FileStatusManager.KeyAndValue()
				_, socketstatus := usersession.SocketstatusManager.KeyAndValue()
				AnnounceAllSocketsInUser(filestatus, socketstatus)
				if currentfilestatus.Current == currentfilestatus.Max {
					pythonprocessesmanager.Set(fileuid, "finished")
					return
				}
			}
		}
		//走到这里则python已退出
	}()
}

func AnnounceAllSocketsInUser[sessionidtype comparable, fileidtype comparable, socketidtype socket.Socketidinterface](
	filelist []*file.FileStatus[fileidtype, sessionidtype], socketlist []*socket.SocketStatus[socketidtype]) {
	fmt.Println("announce all sockets in user")
	for _, v := range socketlist {
		filelistfilterd := file.FileNameFilter(filelist, v.Filter)
		v.Lock.Lock()
		v.Socketid.WriteJSON(filelistfilterd)
		v.Lock.Unlock()
	}
}
func Renderbycsvfile(w http.ResponseWriter, r *http.Request, csvpath string, htmlheadertype int) {
	var t *template.Template
	var headername string
	var err error
	if htmlheadertype == 1 {
		headername = "DBG Event Count List"
		t, err = template.ParseFiles("./templates/dataanalyzer/show_dbg.html")
	} else if htmlheadertype == 2 {
		headername = "IDS Capture Infomation"
		t, err = template.ParseFiles("./templates/dataanalyzer/show_ids.html")
	}
	if err != nil {
		fmt.Println(err)
		return
	}
	if csvpath == "" {
		t.Execute(w, struct {
			Header       []string
			Data         [][]string
			Downloadlink string
			Htmlheader   string
		}{Data: [][]string{},
			Header:       []string{},
			Downloadlink: "",
			Htmlheader:   headername,
		})
		return
	}
	csvfile, err := os.Open(csvpath)
	if err != nil {
		t.Execute(w, struct {
			Header       []string
			Data         [][]string
			Downloadlink string
			Htmlheader   string
		}{Data: [][]string{},
			Header:       []string{},
			Downloadlink: "../" + strings.ReplaceAll(csvpath, "\\", "/"),
			Htmlheader:   headername,
		})
		return
	}
	defer csvfile.Close()

	csvreader := csv.NewReader(csvfile)
	csvdata, err := csvreader.ReadAll()
	if err != nil {
		fmt.Fprintln(w, "Read failed:", err)
	}

	t.Execute(w, struct {
		Header       []string
		Data         [][]string
		Downloadlink string
		Htmlheader   string
	}{Data: func() [][]string {
		if len(csvdata) > 1 {
			if htmlheadertype == 1 {
				Sorted_dbg(csvdata[1:])
			}
			return csvdata[1:]
		}
		return [][]string{}
	}(),
		Header: func() []string {
			if len(csvdata) > 0 {
				return csvdata[0]
			}
			return []string{}
		}(),
		Downloadlink: "../" + strings.ReplaceAll(csvpath, "\\", "/"),
		Htmlheader:   headername,
	})
}
func Sorted_dbg(data [][]string) {
	sort.Slice(data, func(i, j int) bool {
		num1, _ := strconv.Atoi(data[i][1])
		num2, _ := strconv.Atoi(data[j][1])
		return num1 > num2
	})
	return
}

/*
	func AnnounceAllSocketsWithFile(uid string) {
		sockets, socketstatus := socket.SocketManagerGetsAll()
		for i := range sockets {
			filter := socketstatus[i].Filter
			filesinsocket := AccessiableFileInSocket(sockets[i])
			for _, f := range filesinsocket {
				if f.Uid == uid {
					filteredfilesinsocket := file.FileNameFilter(filesinsocket, filter)
					sockets[i].WriteJSON(filteredfilesinsocket)
					break
				}
			}
		}
	}

	func AccessiableFileInSocket(conn *websocket.Conn) []*file.FileStatus {
		socketstatus, ok := socket.SocketManagerGet(conn)
		if !ok {
			fmt.Println("socket not exist")
			return []*file.FileStatus{}
		}
		connHoldingFileStatus := []*file.FileStatus{}
		for _, uid := range socketstatus.Session.Values["filename"].([]string) {
			fileStatus, ok := file.FileStatusMapGet(uid)
			if !ok {
				fmt.Println("non-existing file, failed")
				return []*file.FileStatus{}
			}
			connHoldingFileStatus = append(connHoldingFileStatus, fileStatus)
		}
		return connHoldingFileStatus
	}
*/

func DeleteFileFromLocal(uploadpath string, uid string) {
	os.RemoveAll(filepath.Join(uploadpath, uid))
}

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
