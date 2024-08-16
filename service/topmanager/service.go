package topmanager

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"

	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
	"webapp/service/accounting"
	"webapp/service/cookie"
	"webapp/service/database"
	"webapp/service/file"
	"webapp/service/pythonmanager"
	"webapp/service/session"
	"webapp/service/util"
	"webapp/service/websocketmanager"

	"github.com/gorilla/sessions"
)

func OldFileCollection[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](
	sessionmanager *session.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonmanager pythonmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, sessionidtype],
	fileuid fileidtype, uploadpath string, userid sessionidtype) {
	if cachequeue.Len() > 200 {
		fmt.Println("deleting expired file")
		filetobedelete := cachequeue.Top()
		usersession, ok := sessionmanager.Get(filetobedelete.Useruid)
		if !ok {
			fmt.Println("didnot find the user owning file to be deleted")
		}
		ForceStopAndDeleteFile(fileuid, userid, uploadpath, usersession, pythonmanager, cachequeue)
	}
}

func ForceStopAndDeleteFile[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](
	fileid fileidtype, userid sessionidtype, uploadpath string,
	usersession *session.SessionStatus[sessionidtype, fileidtype, socketidtype],
	pythonmanager pythonmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, sessionidtype]) {

	pythonmanager.Delete(userid, fileid)
	cachequeue.Delete(fileid)
	database.DeleteFileinfo(fileid)
	database.Deletedbgitemstable(fileid)
	usersession.FileStatusManager.Delete(fileid)
	err := file.DeleteFileFromLocal(uploadpath, fmt.Sprintf("%v", fileid))
	if err != nil {
		fmt.Println("delete local file and directory error:", err)
	}
}

func AddFileToMemory[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](
	sessionmanager *session.SessionStatusManager[sessionidtype, fileidtype, socketidtype], pythonprocessesmanager pythonmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, sessionidtype],
	fileuid fileidtype, filename string, current int, max int, userid sessionidtype) {

	filestatus := file.NewFileStatus[fileidtype, sessionidtype]()
	filestatus.Filename = filename
	filestatus.Uid = fileuid
	filestatus.Useruid = userid
	sessionmanager.AddFile(userid, fileuid, filestatus)
	database.AddFileinfo(fileuid, userid)
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

func ParseidsFilebyCmd[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](
	sessionmanager *session.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonprocessesmanager pythonmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, sessionidtype], fileuid fileidtype, uploadpath string, userid sessionidtype) {
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

	cmds, _ := pythonprocessesmanager.Get(fileuid)
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
func ParsedbgFile[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](
	sessionmanager *session.SessionStatusManager[sessionidtype, fileidtype, socketidtype],
	pythonstatusmanager pythonmanager.PythonStatusManager[sessionidtype, fileidtype],
	cachequeue *file.FileCacheQueue[fileidtype, sessionidtype], fileuid fileidtype, uploadpath string, userid sessionidtype) {

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
	pythonstatusmanager.Add(userid, fileuid, filepath.Join(uploadpath, stringuid+".tar.gz")) //三个命令一起初始化了
	taskstatus, _ := pythonstatusmanager.Get(fileuid)
	dbgtaskstatus := taskstatus[util.Dbg]

	if dbgtaskstatus.Calltype == util.Cmd {
		outpipe, _ := dbgtaskstatus.Cmd.StdoutPipe()
		dbgtaskstatus.Cmd.Stderr = os.Stderr

		currentfilestatus.Dbgstatus.State = util.Created
		usersession.FileStatusManager.Set(fileuid, currentfilestatus)
		pythonstatusmanager.Start(userid, fileuid, util.Dbg)
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
		pythonstatusmanager.Start(userid, fileuid, util.Dbg)
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
func AnnounceAllSocketsInUser[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](
	usersession *session.SessionStatus[sessionidtype, fileidtype, socketidtype]) {
	fmt.Println("announce all sockets in user")
	_, userholdfilestatus := usersession.FileStatusManager.KeyAndValue()
	_, userholdsocketstatus := usersession.SocketstatusManager.KeyAndValue()
	for _, v := range userholdsocketstatus {
		filelistfilterd := file.FileNameFilter(userholdfilestatus, v.Filter)
		v.Lock.Lock()
		v.Socketid.WriteJSON(filelistfilterd)
		v.Lock.Unlock()
	}
}

type IDSTemplateDatastruct struct {
	Header       []string
	Data         [][]string
	Downloadlink string
	Htmlheader   string
	Filename     string
}

func Renderbyidsfile(w http.ResponseWriter, r *http.Request, csvpath string, filename string) {
	var t *template.Template
	var headername string
	var err error
	headername = "IDS Event Count List"
	t, err = template.ParseFiles("./templates/dataanalyzer/show_ids.html")

	if err != nil {
		fmt.Println(err)
		return
	}
	IDSTemplateData := IDSTemplateDatastruct{
		Header:       []string{},
		Data:         [][]string{},
		Downloadlink: "",
		Htmlheader:   headername,
		Filename:     filename,
	}
	if csvpath == "" {
		t.Execute(w, IDSTemplateData)
		return
	}
	IDSTemplateData.Downloadlink = "../" + strings.ReplaceAll(csvpath, "\\", "/")
	csvfile, err := os.Open(csvpath)
	if err != nil {
		t.Execute(w, IDSTemplateData)
		return
	}
	defer csvfile.Close()

	csvreader := csv.NewReader(csvfile)
	csvdata, err := csvreader.ReadAll()
	if err != nil {
		t.Execute(w, IDSTemplateData)
		fmt.Println("csvreader.ReadAll() error:", err)
		return
	}

	if len(csvdata) > 0 {
		IDSTemplateData.Header = csvdata[0]
	}
	if len(csvdata) > 1 {
		IDSTemplateData.Data = csvdata[1:]
	}
	t.Execute(w, IDSTemplateData)
}

type DBGTemplateDatastruct struct {
	Header       []string
	Data         [][]string
	Downloadlink string
	Htmlheader   string
	Numbers      [][]int
	Rates        []string
	Categories   map[string]*accounting.Categoryinfo
	Filename     string
}

func Renderbydbgfile(w http.ResponseWriter, r *http.Request, csvpath string, csvpath_acc string, filename string) {
	var t *template.Template
	var headername string
	var err error
	headername = "DBG Event Count List"
	t, err = template.ParseFiles("./templates/dataanalyzer/show_dbg.html")
	if err != nil {
		fmt.Println(err)
		return
	}

	Numbers := make([][]int, 4)
	for i := range Numbers {
		Numbers[i] = make([]int, 2)
	}
	Rates := make([]string, 4)
	for i, _ := range Rates {
		Rates[i] = "0"
	}
	Categories := make(map[string]*accounting.Categoryinfo)
	DBGTemplateData := DBGTemplateDatastruct{
		Data:         [][]string{},
		Header:       []string{},
		Downloadlink: "",
		Htmlheader:   headername,
		Numbers:      Numbers,
		Rates:        Rates,
		Categories:   Categories,
		Filename:     filename,
	}
	if csvpath == "" {
		fmt.Println("no file selected")
		t.Execute(w, DBGTemplateData)
		return
	}

	csvfile, err := os.Open(csvpath)
	DBGTemplateData.Downloadlink = "../" + strings.ReplaceAll(csvpath, "\\", "/")
	if err != nil {
		fmt.Println("csv open failed")
		t.Execute(w, DBGTemplateData)
		return
	}
	defer csvfile.Close()

	accountingfile, err := os.Open(csvpath_acc)
	if err != nil {
		fmt.Println("acc open failed")
		t.Execute(w, DBGTemplateData)
		return
	}
	defer accountingfile.Close()

	csvreader := csv.NewReader(csvfile)
	csvdata, err := csvreader.ReadAll()
	if err != nil {
		fmt.Println(w, "Read failed:", err)
	}
	if len(csvdata) > 1 {
		util.Sortdata(csvdata[1:])
		DBGTemplateData.Data = csvdata[1:]
	}

	if len(csvdata) > 0 {
		DBGTemplateData.Header = csvdata[0]
	}
	accreader := csv.NewReader(accountingfile)
	tempNumbers, err := accreader.ReadAll()
	if err != nil {
		fmt.Println("read failed", err)
	}
	for i, strrow := range tempNumbers[:4] {
		for j, str := range strrow {
			DBGTemplateData.Numbers[i][j], _ = strconv.Atoi(str)
		}
	}
	for i, v := range Numbers {
		if v[1] != 0 {
			DBGTemplateData.Rates[i] = fmt.Sprintf("%.4f", float64(v[0])/float64(v[1]))
		} else {
			DBGTemplateData.Rates[i] = "1"
		}
	}

	Category := []string{"UE接入", "S1切换入", "S1切换出", "未分类"}
	for _, v := range Category {
		DBGTemplateData.Categories[v] = accounting.NewCategoryinfo(v)
	}

	tagsofevent := []string{}
	for _, v := range DBGTemplateData.Data {
		err := json.Unmarshal([]byte(strings.ReplaceAll(v[2], "'", "\"")), &tagsofevent)
		if err != nil {
			fmt.Println(v[2])
			fmt.Println("json unmarshal failed", err)
			return
		}
		for i, _ := range tagsofevent {
			eventcount, err := strconv.Atoi(v[1])
			if err != nil {
				fmt.Println("strconv failed", err)
			}
			DBGTemplateData.Categories[tagsofevent[i]].AddEvent(v[0], eventcount)
			DBGTemplateData.Categories[tagsofevent[i]].Count += eventcount
		}
	}
	for _, v := range DBGTemplateData.Categories {
		v.SortEvent()
	}
	t.Execute(w, DBGTemplateData)
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

func NewUserintoMemory[fileidtype comparable, socketidtype websocketmanager.Socketidinterface](w http.ResponseWriter, r *http.Request,
	cook *sessions.Session, sessionmanager *session.SessionStatusManager[string, fileidtype, socketidtype]) {
	cookie.GenerateNewId(w, r, cook)
	userid, ok := cookie.CookieGet(r).Values["id"].(string)
	if !ok {
		fmt.Println("userid assert string failed")
		return
	}
	sessionmanager.Add(userid)
	database.AddUserinfo(userid)
	time.AfterFunc(48*time.Hour, func() {
		database.DeleteUserinfo(userid)
		sessionmanager.Delete(userid)
		fmt.Println("Delete User since expired:", userid)
	})
	fmt.Println("Add User:", userid)
}

func ConstructJSONHandle[sessionidtype comparable, fileidtype comparable, socketidtype websocketmanager.Socketidinterface](
	sessionmanager *session.SessionStatusManager[sessionidtype, fileidtype, socketidtype], pythonstatusmanager pythonmanager.PythonStatusManager[sessionidtype, fileidtype]) func(int, []byte) {
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
			return
		}
		currentfilestatus, ok := usersession.FileStatusManager.Get(fileuid)
		if !ok {
			fmt.Println("fileuid not exist")
			return
		}
		if jsondata["state"] == "success" {
			if jsondata["function"] == "Dbg" {
				pythonstatusmanager.SetState(fileuid, util.Dbg, util.Idle)
				currentfilestatus.Dbgstatus.State = util.Finished
				usersession.FileStatusManager.Set(fileuid, currentfilestatus)
				AnnounceAllSocketsInUser(usersession)
			}
		}
	}
}
