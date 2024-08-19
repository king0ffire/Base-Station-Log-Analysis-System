package service

import (
	"bufio"
	"encoding/json"
	"fmt"

	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"webapp/dataaccess"
	"webapp/service/lowermanager"
	"webapp/service/topmanager"
	"webapp/util"

	"github.com/gorilla/sessions"
)

func OldFileCollection[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, websocketidtype],
	pythonmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype],
	fileuid fileidtype, uploadpath string, userid sessionidtype) {
	if cachequeue.Len() > 200 {
		fmt.Println("deleting expired file")
		filetobedelete := cachequeue.Top()
		usersession, ok := sessionmanager.Get(filetobedelete.Useruid)
		if !ok {
			fmt.Println("didnot find the user owning file to be deleted")
		}
		ForceStopAndDeleteFile(filetobedelete, uploadpath, usersession, pythonmanager, cachequeue)
	}
}

func ForceStopAndDeleteFile[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	filestatus *lowermanager.FileStatus[sessionidtype, fileidtype], uploadpath string,
	usersession *topmanager.SessionStatus[sessionidtype, fileidtype, websocketidtype],
	pythonmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype]) {

	pythonmanager.Delete(filestatus)
	cachequeue.Delete(filestatus.Uid)
	dataaccess.DatabaseDeleteFileinfo(filestatus.Uid)
	dataaccess.DatabaseDeletedbgitemstable(filestatus.Uid)
	usersession.FileStatusManager.Delete(filestatus.Uid)
	err := dataaccess.DeleteDirFromLocal(uploadpath, fmt.Sprintf("%v", filestatus.Uid))
	if err != nil {
		fmt.Println("delete local file and directory error:", err)
	}
}

func AddFileToMemory[sessionidtype comparable, fileidtype comparable, socketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonprocessesmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype],
	fileuid fileidtype, filename string, current int, max int, userid sessionidtype) {

	filestatus := lowermanager.NewFileStatus[sessionidtype, fileidtype]()
	filestatus.Filename = filename
	filestatus.Uid = fileuid
	filestatus.Useruid = userid
	sessionmanager.AddFile(userid, fileuid, filestatus)
	dataaccess.DatabaseAddFileinfo(fileuid, userid)
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

func ParseidsFilebyCmd[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, websocketidtype],
	pythonprocessesmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype], fileuid fileidtype, uploadpath string, userid sessionidtype) {
	stringuid := fmt.Sprintf("%v", fileuid)

	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		fmt.Println("didnot find the user")
		return
	}
	currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
	if !ok {
		fmt.Println("didnot find the file")
		return
	}

	cmds, _ := pythonprocessesmanager.Get(currentfilestatus)
	idscmd := cmds[util.Sctp]
	removecachecmd := cmds[util.Delete]

	outpipe, _ := idscmd.Cmd.StdoutPipe()
	idscmd.Cmd.Stderr = os.Stderr

	currentfilestatus.Sctpstatus.Lock.Lock()
	if currentfilestatus.Dbgstatus.State != util.Finished {
		fmt.Println("the dbglog analysis is not finished, should wait")
		return
	}
	if currentfilestatus.Sctpstatus.State != util.Noschedule {
		fmt.Println("the sctp analysis is being processed")
		return
	}
	currentfilestatus.Sctpstatus.State = util.Created
	currentfilestatus.Sctpstatus.Lock.Unlock()
	usersession.FileStatusManager.Set(fileuid, currentfilestatus)
	if err := idscmd.Cmd.Start(); err != nil {
		fmt.Println("start python fail:", err)
		return
	}

	go func() {
		fmt.Println("Go thread start:", stringuid)
		defer fmt.Println("Go thread end:", stringuid)
		defer func() {
			idscmd.Cmd.Wait()
			idscmd.State = util.Idle
			removecachecmd.Cmd.Start()
			removecachecmd.Cmd.Wait()
		}()
		currentfilestatus.Sctpstatus.State = util.Running
		idscmd.State = util.Running
		scanner := bufio.NewScanner(outpipe)
		scanner.Scan()
		fmt.Println("Received first line of ids analysis:", stringuid, ":", scanner.Text())
		intText, err := strconv.Atoi(scanner.Text())
		if err != nil {
			fmt.Println("text conversion:", err)
			currentfilestatus.Sctpstatus.State = util.Failed
			return
		}
		currentfilestatus.Sctpstatus.Maxvalue = intText
		ok = usersession.FileStatusManager.Set(currentfilestatus.Uid, currentfilestatus)
		if !ok {
			fmt.Println("the file is already cleaned")
			currentfilestatus.Sctpstatus.State = util.Failed
			return
		}
		AnnounceAllSocketsInUser(usersession)
		for scanner.Scan() {
			fmt.Println("python outputs:", scanner.Text())
			if scanner.Text() == "sctp_finished_one" {
				currentfilestatus.Sctpstatus.Currentvalue++
				ok := usersession.FileStatusManager.Set(currentfilestatus.Uid, currentfilestatus)
				if !ok { //异常退出
					fmt.Println("the file is already cleaned")
					currentfilestatus.Sctpstatus.State = util.Failed
					return
				}
				if currentfilestatus.Sctpstatus.Currentvalue == currentfilestatus.Sctpstatus.Maxvalue {
					currentfilestatus.Sctpstatus.State = util.Finished
					AnnounceAllSocketsInUser(usersession)
					return
				}
				AnnounceAllSocketsInUser(usersession)
			}
		}
		currentfilestatus.Sctpstatus.State = util.Failed
		AnnounceAllSocketsInUser(usersession)
		//走到这里则python已退出
	}()
}
func ParsedbgFile[sessionidtype comparable, fileidtype comparable, socketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonstatusmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype], fileuid fileidtype, uploadpath string, userid sessionidtype) {

	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		fmt.Println("didnot find the user")
		return
	}
	currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
	if !ok {
		fmt.Println("didnot find the file")
		return
	}

	stringuid := fmt.Sprintf("%v", fileuid)
	pythonstatusmanager.Add(currentfilestatus, filepath.Join(uploadpath, stringuid+".tar.gz")) //三个命令一起初始化了
	taskstatus, _ := pythonstatusmanager.Get(currentfilestatus)
	dbgtaskstatus := taskstatus[util.Dbg]

	if dbgtaskstatus.Calltype == util.Cmd {
		outpipe, _ := dbgtaskstatus.Cmd.StdoutPipe()
		dbgtaskstatus.Cmd.Stderr = os.Stderr

		currentfilestatus.Dbgstatus.State = util.Created
		usersession.FileStatusManager.Set(fileuid, currentfilestatus)
		pythonstatusmanager.Start(currentfilestatus, util.Dbg)
		dbgtaskstatus.State = util.Running
		currentfilestatus.Dbgstatus.State = util.Running
		AnnounceAllSocketsInUser(usersession)
		go func() {
			fmt.Println("Go thread start:", stringuid)
			defer fmt.Println("Go thread end:", stringuid)
			defer func() {
				dbgtaskstatus.Cmd.Wait()
				dbgtaskstatus.State = util.Idle
			}()
			scanner := bufio.NewScanner(outpipe)
			for scanner.Scan() {
				fmt.Println("python outputs:", scanner.Text())
				if scanner.Text() == "dbg analysis success" {
					currentfilestatus.Dbgstatus.State = util.Finished
					AnnounceAllSocketsInUser(usersession)
					return
				}
			}
			currentfilestatus.Dbgstatus.State = util.Failed
			AnnounceAllSocketsInUser(usersession)
		}()
	} else if dbgtaskstatus.Calltype == util.Rpc {
		dbgtaskstatus.State = util.Running
		currentfilestatus.Dbgstatus.State = util.Created
		pythonstatusmanager.Start(currentfilestatus, util.Dbg)
		currentfilestatus.Dbgstatus.State = util.Running
		AnnounceAllSocketsInUser(usersession)
	}

}

/*
func AnnounceAllSocketsInUser[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](

		filelist []*file.FileStatus[fileidtype, sessionidtype], socketlist []*websocketmanager.SocketStatus[socketidtype]) {
		fmt.Println("announce all sockets in user")
		for _, v := range socketlist {
			filelistfilterd := file.FileNameFilter(filelist, v.Filter)
			v.Lock.Lock()
			v.Socketid.WriteJSON(filelistfilterd)
			v.Lock.Unlock()
		}
	}
*/
func AnnounceAllSocketsInUser[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	usersession *topmanager.SessionStatus[sessionidtype, fileidtype, websocketidtype]) {
	fmt.Println("announce all sockets in user")
	_, userholdfilestatus := usersession.FileStatusManager.KeyAndValue()
	_, userholdsocketstatus := usersession.WebSocketstatusManager.KeyAndValue()
	for _, v := range userholdsocketstatus {
		filelistfilterd := lowermanager.FileNameFilter(userholdfilestatus, v.Filter)
		v.Lock.Lock()
		v.Socketid.WriteJSON(filelistfilterd)
		v.Lock.Unlock()
	}
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

func NewUserintoMemory[socketidtype lowermanager.WebSocketID, fileidtype comparable](w http.ResponseWriter, r *http.Request,
	cook *sessions.Session, sessionmanager *topmanager.SessionStatusManager[string, fileidtype, socketidtype]) {
	util.GenerateNewId(w, r, cook)
	userid, ok := util.CookieGet(r).Values["id"].(string)
	if !ok {
		fmt.Println("userid assert string failed")
		return
	}
	sessionmanager.Add(userid)
	dataaccess.DatabaseAddUserinfo(userid)
	time.AfterFunc(48*time.Hour, func() {
		dataaccess.DatabaseDeleteUserinfo(userid)
		sessionmanager.Delete(userid)
		fmt.Println("Delete User since expired:", userid)
	})
	fmt.Println("Add User:", userid)
}

func ConstructJSONHandle[sessionidtype comparable, fileidtype comparable, socketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonstatusmanager topmanager.PythonStatusManager[sessionidtype, fileidtype], cache_location string) func(int, []byte) {
	return func(n int, jsondump []byte) {
		var jsondata map[string]interface{}
		err := json.Unmarshal(jsondump[:n], &jsondata)
		if err != nil {
			fmt.Println("json unmarshal:", err)
			return
		}
		useruid := jsondata["useruid"].(sessionidtype)
		fileuid := jsondata["fileuid"].(fileidtype)
		usersession, ok := sessionmanager.Get(useruid)
		if !ok {
			fmt.Println("useruid not exist")
			dataaccess.DeleteDirFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			dataaccess.DeleteFileFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			return
		}
		currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
		if !ok {
			fmt.Println("fileuid not exist")
			dataaccess.DeleteDirFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			dataaccess.DeleteFileFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			return
		}
		if jsondata["state"] == "Success" {
			if jsondata["task"] == "Dbg" {
				pythonstatusmanager.SetState(currentfilestatus, util.Dbg, util.Idle)
				currentfilestatus.Dbgstatus.State = util.Finished
				usersession.FileStatusManager.Set(fileuid, currentfilestatus)
				AnnounceAllSocketsInUser(usersession)
			}
		} else {
			dataaccess.DeleteDirFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			dataaccess.DeleteFileFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
		}

	}
}
