package service

import (
	"bufio"
	"encoding/json"
	"fmt"

	"net/http"
	"strconv"
	"time"
	"webapp/dataaccess"
	"webapp/service/lowermanager"
	"webapp/service/topmanager"
	"webapp/util"

	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

func InitFileWithDBG[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, websocketidtype],
	pythonprocessesmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype],
	fileuid fileidtype, filename string, uploadpath string, current int, max int, userid sessionidtype) {
	filestatus := lowermanager.NewFileStatus[sessionidtype, fileidtype]()
	filestatus.Dbgstatus.Lock.Lock()
	logrus.Debugf("file %v acquired lock", fileuid)

	filestatus.Filename = filename
	filestatus.Uid = fileuid
	filestatus.Useruid = userid
	AddFileToMemory(sessionmanager, pythonprocessesmanager, cachequeue, fileuid, filename, uploadpath, current, max, userid, filestatus)
	ParsedbgFile(sessionmanager, pythonprocessesmanager, cachequeue, fileuid, uploadpath, userid)
	filestatus.Dbgstatus.Lock.Unlock()
	logrus.Debugf("file %v released lock", filestatus.Uid)
}
func AddFileToMemory[sessionidtype comparable, fileidtype comparable, socketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonprocessesmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype],
	fileuid fileidtype, filename string, uploadpath string, current int, max int, userid sessionidtype, filestatus *lowermanager.FileStatus[sessionidtype, fileidtype]) {
	sessionmanager.AddFile(userid, fileuid, filestatus)
	logrus.Debugf("fileid=%v added to filemangaer of user", filestatus.Uid)
	dataaccess.DatabaseAddFileinfo(fileuid, userid)
	logrus.Debugf("fileid=%v added to database", filestatus.Uid)
	PushQueueAndDeleteOld(sessionmanager, pythonprocessesmanager, cachequeue, fileuid, uploadpath, userid, filestatus)
	logrus.Debugf("fileid=%v added to memory", filestatus.Uid)
}

func PushQueueAndDeleteOld[sessionidtype comparable, fileidtype comparable, socketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonprocessesmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype],
	fileuid fileidtype, uploadpath string, userid sessionidtype, filestatus *lowermanager.FileStatus[sessionidtype, fileidtype]) {
	filetobedeleted := cachequeue.PushAndPopWhenFull(filestatus, 5)
	logrus.Debug(fileuid, " added to cachequeue, current length: ", cachequeue.Len())
	if filetobedeleted != nil {
		filetobedeleted.Dbgstatus.Lock.Lock()
		logrus.Debugf("file %v acquired lock", filetobedeleted.Uid)
		defer func() {
			filetobedeleted.Dbgstatus.Lock.Unlock()
			logrus.Debugf("file %v released lock", filetobedeleted.Uid)
		}()
		logrus.Debug("deleting expired file: ", filetobedeleted.Uid)
		usersession, ok := sessionmanager.Get(filetobedeleted.Useruid)
		if !ok {
			logrus.Debug("didnot find the user owning file to be deleted")
		}
		ForceStopAndDeleteFile(filetobedeleted, uploadpath, usersession, pythonprocessesmanager, cachequeue)
	}
}
func ParseidsFilebyCmd[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, websocketidtype],
	pythonprocessesmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype], fileuid fileidtype, uploadpath string, userid sessionidtype) {
	stringuid := fmt.Sprintf("%v", fileuid)

	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		logrus.Debug("didnot find the user")
		return
	}
	currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
	if !ok {
		logrus.Debug("didnot find the file")
		return
	}
	currentfilestatus.Sctpstatus.Lock.Lock()
	if currentfilestatus.Dbgstatus.State != util.Finished {
		logrus.Info("the dbglog analysis is not finished, should wait")
		currentfilestatus.Sctpstatus.Lock.Unlock()
		return
	}
	currentfilestatus.Sctpstatus.State = util.Created
	currentfilestatus.Sctpstatus.Lock.Unlock()
	cmds, _ := pythonprocessesmanager.Get(currentfilestatus)
	idscmd := cmds[util.Sctp]
	removecachecmd := cmds[util.Delete]

	stdoutpipe, err := idscmd.Cmd.StdoutPipe()
	if err != nil {
		logrus.Error("get stdout pipe:", err)
	}
	stderrpipe, err := idscmd.Cmd.StderrPipe()
	if err != nil {
		logrus.Error("get stderr pipe:", err)
	}

	usersession.FileStatusManager.Set(fileuid, currentfilestatus)
	if err := idscmd.Cmd.Start(); err != nil {
		logrus.Error("start python fail:", err)
		return
	}

	go func() {
		logrus.Debug("Go thread start:", stringuid)
		defer logrus.Debug("Go thread end:", stringuid)
		defer func() {
			idscmd.Cmd.Wait()
			idscmd.State = util.Idle
			removecachecmd.Cmd.Start()
			removecachecmd.Cmd.Wait()
		}()
		currentfilestatus.Sctpstatus.State = util.Running
		idscmd.State = util.Running
		scanner := bufio.NewScanner(stdoutpipe)
		scanner.Scan()
		logrus.Debug("Received first line of ids analysis:", stringuid, ":", scanner.Text())
		intText, err := strconv.Atoi(scanner.Text())
		if err != nil {
			logrus.Error("text conversion:", err)
			currentfilestatus.Sctpstatus.State = util.Failed
			return
		}
		currentfilestatus.Sctpstatus.Maxvalue = intText
		ok = usersession.FileStatusManager.Set(currentfilestatus.Uid, currentfilestatus)
		if !ok {
			logrus.Debug("the file is already cleaned")
			currentfilestatus.Sctpstatus.State = util.Failed
			return
		}
		AnnounceAllSocketsInUser(userid, usersession)
		for scanner.Scan() {
			logrus.Debug("python outputs:", scanner.Text())
			if scanner.Text() == "sctp_finished_one" {
				currentfilestatus.Sctpstatus.Currentvalue++
				ok := usersession.FileStatusManager.Set(currentfilestatus.Uid, currentfilestatus)
				if !ok { //异常退出
					logrus.Debug("the file is already cleaned")
					currentfilestatus.Sctpstatus.State = util.Failed
					return
				}
				if currentfilestatus.Sctpstatus.Currentvalue == currentfilestatus.Sctpstatus.Maxvalue {
					currentfilestatus.Sctpstatus.State = util.Finished
					AnnounceAllSocketsInUser(userid, usersession)
					return
				}
				AnnounceAllSocketsInUser(userid, usersession)
			}
		}
		currentfilestatus.Sctpstatus.State = util.Failed
		AnnounceAllSocketsInUser(userid, usersession)
		//走到这里则python已退出
	}()
	go func() {
		scanner := bufio.NewScanner(stderrpipe)
		for scanner.Scan() {
			logrus.Info("python error:", scanner.Text())
		}
	}()
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
	userid sessionidtype,
	usersession *topmanager.SessionStatus[sessionidtype, fileidtype, websocketidtype]) {
	logrus.Debug("announce all sockets in user: ", userid)
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
		logrus.Error("userid assert string failed")
		return
	}
	sessionmanager.Add(userid)
	dataaccess.DatabaseAddUserinfo(userid)
	time.AfterFunc(48*time.Hour, func() {
		dataaccess.DatabaseDeleteUserinfo(userid)
		sessionmanager.Delete(userid)
		logrus.Debug("Delete User since expired:", userid)
	})
	logrus.Debug("Add User:", userid)
}

func ConstructJSONHandle[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	pythonmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype],
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, websocketidtype],
	cache_location string) func(int, []byte) {
	return func(n int, jsondump []byte) {
		var jsondata map[string]interface{}
		err := json.Unmarshal(jsondump[:n], &jsondata)
		if err != nil {
			logrus.Error("json unmarshal:", err)
			return
		}
		useruid := jsondata["useruid"].(sessionidtype)
		fileuid := jsondata["fileuid"].(fileidtype)
		usersession, ok := sessionmanager.Get(useruid)
		if !ok {
			logrus.Debug("fileid ", fileuid, ":useruid ", useruid, "; userid not exist")
			dataaccess.DatabaseDeleteFileinfo(useruid)
			dataaccess.DatabaseDeletedbgitemstable(useruid)
			err := dataaccess.DeleteDirFromLocal(cache_location, fmt.Sprintf("%v", useruid))
			if err != nil {
				logrus.Errorf("delete %v directory error:%s\n", useruid, err)
			}
			err = dataaccess.DeleteFileFromLocal(cache_location, fmt.Sprintf("%v", useruid))
			if err != nil {
				logrus.Errorf("delete %v file error:%s\n", useruid, err)
			}
			return
		}
		defer AnnounceAllSocketsInUser(useruid, usersession)

		currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
		if !ok {
			logrus.Debug("fileid ", fileuid, ":useruid ", useruid, "; fileid not exist")
			cachequeue.Delete(fileuid)
			dataaccess.DatabaseDeleteFileinfo(fileuid)
			dataaccess.DatabaseDeletedbgitemstable(fileuid)
			usersession.FileStatusManager.Delete(fileuid)
			err := dataaccess.DeleteDirFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			if err != nil {
				logrus.Errorf("delete %v directory error:%s\n", fileuid, err)
			}
			err = dataaccess.DeleteFileFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			if err != nil {
				logrus.Errorf("delete %v file error:%s\n", fileuid, err)
			}
			return
		}
		if currentfilestatus.Dbgstatus.State == util.Terminated || currentfilestatus.Sctpstatus.State == util.Terminated {
			if currentfilestatus.Dbgstatus.State != util.Running && currentfilestatus.Sctpstatus.State != util.Running {
				logrus.Debug("scheduled termination for: ", fileuid)
				pythonmanager.Delete(currentfilestatus)
				cachequeue.Delete(fileuid)
				dataaccess.DatabaseDeleteFileinfo(fileuid)
				dataaccess.DatabaseDeletedbgitemstable(fileuid)
				usersession.FileStatusManager.Delete(fileuid)
				err := dataaccess.DeleteDirFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
				if err != nil {
					logrus.Errorf("delete %v directory error:%s\n", fileuid, err)
				}
				err = dataaccess.DeleteFileFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
				if err != nil {
					logrus.Errorf("delete %v file error:%s\n", fileuid, err)
				}
				return
			}
		}
		if jsondata["state"] == "Success" {
			if jsondata["task"] == "Dbg" {
				pythonmanager.SetState(currentfilestatus, util.Dbg, util.Idle)
				currentfilestatus.Dbgstatus.State = util.Finished
				usersession.FileStatusManager.Set(fileuid, currentfilestatus)
			}
		} else {
			logrus.Debug("file ", fileuid, " might be deleted earlier")
			pythonmanager.Delete(currentfilestatus)
			cachequeue.Delete(fileuid)
			dataaccess.DatabaseDeleteFileinfo(fileuid)
			dataaccess.DatabaseDeletedbgitemstable(fileuid)
			usersession.FileStatusManager.Delete(fileuid)
			err := dataaccess.DeleteDirFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			if err != nil {
				logrus.Errorf("delete %v directory error:%s\n", fileuid, err)
			}
			err = dataaccess.DeleteFileFromLocal(cache_location, fmt.Sprintf("%v", fileuid))
			if err != nil {
				logrus.Errorf("delete %v file error:%s\n", fileuid, err)
			}
		}
	}
}
