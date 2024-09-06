package service

import (
	"bufio"
	"fmt"
	"path/filepath"
	"webapp/dataaccess"
	"webapp/service/lowermanager"
	"webapp/service/models"
	"webapp/service/topmanager"
	"webapp/util"

	"github.com/sirupsen/logrus"
)

func OldFileCollection[sessionidtype comparable, fileidtype comparable, websocketidtype models.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, websocketidtype],
	pythonmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *lowermanager.ServerCacheQueue[sessionidtype, fileidtype],
	fileuid fileidtype, uploadpath string, userid sessionidtype) {
	logrus.Debug("during processing ", fileuid, ", current cache queue length: ", cachequeue.Len())
	if cachequeue.Len() > 5 {
		logrus.Debug("deleting expired file")
		filetobedelete := cachequeue.Pop()
		usersession, ok := sessionmanager.Get(filetobedelete.Useruid)
		if !ok {
			logrus.Debug("didnot find the user owning file to be deleted")
		}
		ForceStopAndDeleteFile(filetobedelete, uploadpath, usersession, pythonmanager, cachequeue)
	}
}

/*
func ScheduleForceStopAndDeleteFile[sessionidtype comparable, fileidtype comparable, websocketidtype lowermanager.WebSocketID](
	filestatus *lowermanager.FileStatus[sessionidtype, fileidtype], uploadpath string,
	usersession *topmanager.SessionStatus[sessionidtype, fileidtype, websocketidtype],
	pythonmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *topmanager.ServerCacheQueue[sessionidtype, fileidtype]) {
	pythonmanager.Stop(filestatus)
	logrus.Debug("scheduled stop for: ", filestatus.Uid)
}*/

func ForceStopAndDeleteFile[sessionidtype comparable, fileidtype comparable, websocketidtype models.WebSocketID](
	filestatus *models.FileStatus[sessionidtype, fileidtype], uploadpath string,
	usersession *topmanager.SessionStatus[sessionidtype, fileidtype, websocketidtype],
	pythonmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *lowermanager.ServerCacheQueue[sessionidtype, fileidtype]) { //should use with lock filestatus
	logrus.Debugf("force stop and delete file %v", filestatus.Uid)
	pythonmanager.Stop(filestatus)
	pythonmanager.Delete(filestatus)
	logrus.Debugf("file %v removed from python process manager", filestatus.Uid)
	cachequeue.Delete(filestatus.Uid)
	logrus.Debugf("file %v removed from cache queue", filestatus.Uid)
	dataaccess.DatabaseDeleteFileinfo(filestatus.Uid)
	dataaccess.DatabaseDeletedbgitemstable(filestatus.Uid)
	logrus.Debugf("file %v deleted from database", filestatus.Uid)
	usersession.FileStatusManager.Delete(filestatus.Uid)
	err := dataaccess.DeleteDirFromLocal(uploadpath, fmt.Sprintf("%v", filestatus.Uid))
	if err != nil {
		logrus.Errorf("delete %v directory error:%s\n", filestatus.Uid, err)
	}
	err = dataaccess.DeleteFileFromLocal(uploadpath, fmt.Sprintf("%v", filestatus.Uid))
	if err != nil {
		logrus.Errorf("delete %v file error:%s\n", filestatus.Uid, err)
	}
}

func ParsedbgFile[sessionidtype comparable, fileidtype comparable, socketidtype models.WebSocketID](
	sessionmanager *topmanager.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonstatusmanager topmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *lowermanager.ServerCacheQueue[sessionidtype, fileidtype], fileuid fileidtype, uploadpath string, userid sessionidtype) {

	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		logrus.Debug("didnot find the user for file: ", fileuid)
		return
	}
	currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
	if !ok {
		logrus.Debug("didnot find the file: ", fileuid)
		return
	}

	stringuid := fmt.Sprintf("%v", fileuid)
	pythonstatusmanager.Add(currentfilestatus, filepath.Join(uploadpath, stringuid+".tar.gz")) //三个命令一起初始化了
	taskstatus, _ := pythonstatusmanager.Get(currentfilestatus)
	dbgtaskstatus := taskstatus[util.Dbg]

	for _, v := range taskstatus {
		if v.State == util.Terminated {
			logrus.Debugf("the task %s is scheduled termination", v.Calltype)
			return
		}
	}
	logrus.Debugf("checking file task mode for file: %v", fileuid)
	if dbgtaskstatus.Calltype == util.Cmd {
		stdoutpipe, err := dbgtaskstatus.Cmd.StdoutPipe()
		if err != nil {
			logrus.Errorf("error in getting stdout pipe for file: %v", fileuid)
		}
		stderrpipe, err := dbgtaskstatus.Cmd.StderrPipe()
		if err != nil {
			logrus.Errorf("error in getting stderr pipe for file: %v", fileuid)
		}
		currentfilestatus.Dbgstatus.State = util.Created
		usersession.FileStatusManager.Set(fileuid, currentfilestatus)
		pythonstatusmanager.Start(currentfilestatus, util.Dbg)
		dbgtaskstatus.State = util.Running
		currentfilestatus.Dbgstatus.State = util.Running
		AnnounceAllSocketsInUser(userid, usersession)
		go func() {
			logrus.Debug("Go thread start:", stringuid)
			defer logrus.Debug("Go thread end:", stringuid)
			defer func() {
				dbgtaskstatus.Cmd.Wait()
				dbgtaskstatus.State = util.Idle
			}()
			scanner := bufio.NewScanner(stdoutpipe)
			for scanner.Scan() {
				logrus.Debug("python outputs:", scanner.Text())
				if scanner.Text() == "dbg analysis success" {
					currentfilestatus.Dbgstatus.State = util.Finished
					AnnounceAllSocketsInUser(userid, usersession)
					return
				}
			}
			currentfilestatus.Dbgstatus.State = util.Failed
			AnnounceAllSocketsInUser(userid, usersession)
		}()
		go func() {
			scanner := bufio.NewScanner(stderrpipe)
			for scanner.Scan() {
				logrus.Info("python error outputs:", scanner.Text())
			}
		}()
	} else if dbgtaskstatus.Calltype == util.Rpc {
		logrus.Debug("the rpc call preparation: ", fileuid)
		pythonstatusmanager.Start(currentfilestatus, util.Dbg)
		dbgtaskstatus.State = util.Running
		currentfilestatus.Dbgstatus.State = util.Running
		AnnounceAllSocketsInUser(userid, usersession)
	}
}
