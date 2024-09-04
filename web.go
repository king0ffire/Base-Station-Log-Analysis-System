package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"

	"net/http"
	"os"
	"path/filepath"
	"strings"
	"webapp/dataaccess"
	"webapp/service"
	"webapp/service/topmanager"
	"webapp/util"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

var uploadpath = util.ConfigMap["file"]["cache_path"]
var localport = fmt.Sprintf(":%s", util.ConfigMap["webapp"]["port"])

var sessionmanager = topmanager.NewSessionStatusManager[string, string, *websocket.Conn]()
var cachequeue = &topmanager.ServerCacheQueue[string, string]{}
var pythonstatusmanager *topmanager.PythonServiceStatusManager[string, string]

func init() {

	// var pythonprocessesmanager = pythonmanager.NewManager[string]()
	/*
		cmd := exec.Command("python", "./scripts/server.py", uploadpath)
		outpipe, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println("error:", err)
		}
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Start()
		if err != nil {
			fmt.Println("start python error:", err)
		}
		scanner := bufio.NewScanner(outpipe)
		scanner.Scan()
		fmt.Println("python output:", scanner.Text())
	*/
	pythonstatusmanager = topmanager.NewPythonServiceStatusManager[string, string]()
	(*pythonstatusmanager.PythonServerSocketManager.Socket).NewPythonServerListener(service.ConstructJSONHandle(pythonstatusmanager, cachequeue, sessionmanager, uploadpath))
	logrus.Info("python service listening!")
}

func indexentry(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		cook := util.CookieGet(r)
		if cook.IsNew {
			service.NewUserintoMemory(w, r, cook, sessionmanager)
		} else {
			logrus.Debug("cookie exist on brower")
			userid, ok := util.CookieGet(r).Values["id"].(string)
			if !ok {
				logrus.Debug("userid assert string failed")
				return
			}
			_, ok = sessionmanager.Get(userid)
			if !ok {
				logrus.Debug("but session expired on backend")
				service.NewUserintoMemory(w, r, cook, sessionmanager)
			}
		}
		t, err := template.ParseFiles("./templates/home/index.html")
		if err != nil {
			logrus.Error(err)
			return
		}
		t.Execute(w, struct {
			URL string
		}{
			URL: localport,
		})
	}
}
func upload(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("upload page request received")
	if r.Method == "POST" {
		logrus.Debug("收到上传的POST请求")
		r.ParseMultipartForm(256 << 20) //32MB
		multipartfile, multiparthandler, err := r.FormFile("uploadfile")
		if err != nil {
			logrus.Error("upload fail:", err)
			return
		}
		defer multipartfile.Close()
		userid, ok := util.CookieGet(r).Values["id"].(string)
		if !ok {
			logrus.Error("userid not found")
			return
		}
		fileuid, created := dataaccess.MultiPartFileSaver(uploadpath, &multipartfile, multiparthandler)
		if !created {
			logrus.Error("created failed:", created)
			return
		}
		service.InitFileWithDBG(sessionmanager, pythonstatusmanager, cachequeue, fileuid, multiparthandler.Filename, uploadpath, 0, 0, userid)
		http.Redirect(w, r, "/uploadedfiles?filename="+multiparthandler.Filename, http.StatusSeeOther)
		logrus.Debug("redirect success")
	} else {
		t, err := template.ParseFiles("./templates/upload/upload.html")
		if err != nil {
			logrus.Error(err)
			return
		}
		t.Execute(w, struct {
			URL string
		}{
			URL: localport,
		})

	}
}

func uploadedfiles(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("uploadedfiles")

	t, err := template.ParseFiles("./templates/dataanalyzer/basedata.html")
	if err != nil {
		logrus.Info(err)
		return
	}
	t.Execute(w, struct {
		URL      string
		Filename string
	}{
		URL:      localport,
		Filename: r.URL.Query().Get("filename"),
	})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func sockethandler_withfilter(w http.ResponseWriter, r *http.Request) { //初始话一个websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer conn.Close()
	//升级后从中读出filter内容
	_, p, err := conn.ReadMessage()
	if err != nil {
		logrus.Error("read error:", err)
		return
	}
	filter := string(p)
	userid, ok := util.CookieGet(r).Values["id"].(string)
	if !ok {
		logrus.Debug("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		logrus.Debug("usersession not found")
		return
	}
	logrus.Debug("current user id: ", userid)
	usersession.WebSocketstatusManager.Add(conn, filter, util.CookieGet(r))
	defer usersession.WebSocketstatusManager.Delete(conn)
	currentsocket, ok := usersession.WebSocketstatusManager.Get(conn)
	if !ok {
		logrus.Debug("socket not found")
		return
	}
	service.AnnounceAllSocketsInUser(userid, usersession)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logrus.Debug("websocket read error:", err)
			break
		}
		if string(message) == "clearcache" {
			cleardata(w, r)
			currentsocket.Lock.Lock()
			conn.WriteJSON([]interface{}{})
			currentsocket.Lock.Unlock()
			logrus.Info("Empty Json returned to ", userid)
		} else {
			idstobeparsed := string(message)
			logrus.Info("start to parse ids:", idstobeparsed)
			service.ParseidsFilebyCmd(sessionmanager, pythonstatusmanager, cachequeue, idstobeparsed, uploadpath, userid)
		}
	}
}

func cleardata(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("clear cache")
	userid, ok := util.CookieGet(r).Values["id"].(string)
	if !ok {
		logrus.Debug("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		logrus.Debug("usersession not found")
		return
	}
	_, userholdingfiles := usersession.FileStatusManager.KeyAndValue()
	for _, v := range userholdingfiles {
		v.Dbgstatus.Lock.Lock()
		logrus.Debugf("file %v acquired lock", v.Uid)
		service.ForceStopAndDeleteFile(v, uploadpath, usersession, pythonstatusmanager, cachequeue)
		v.Dbgstatus.Lock.Unlock()
		logrus.Debugf("file %v released lock", v.Uid)
		logrus.Debug("cleared cache:", v.Uid)
	}
}

func render404(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./templates/404.html")
	if err != nil {
		logrus.Debug(err)
		return
	}
	t.Execute(w, struct {
		URL string
	}{
		URL: localport,
	})
}
func showresults_dbg(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("showresults_dbg")
	fileid := r.URL.Query().Get("fileid")
	userid, ok := util.CookieGet(r).Values["id"].(string)
	if !ok {
		logrus.Debug("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		logrus.Debug("usersession not found")
		return
	}
	currentfilestatus, ok := usersession.FileStatusManager.Get(fileid)
	if !ok {
		logrus.Debug("file status not found")
		return
	}
	csvpath := filepath.Join(uploadpath, fileid, "dbg.csv")
	csvpath_acc := filepath.Join(uploadpath, fileid, "accounting.csv")
	if fileid == "" {
		csvpath = ""
	}
	service.Renderbydbgfile(w, r, csvpath, csvpath_acc, currentfilestatus.Filename)
}

func showresults_ids(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("showresults_ids")
	fileid := r.URL.Query().Get("fileid")
	userid, ok := util.CookieGet(r).Values["id"].(string)
	if !ok {
		logrus.Debug("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		logrus.Debug("usersession not found")
		return
	}
	filestatus, ok := usersession.FileStatusManager.Get(fileid)
	if !ok {
		logrus.Debug("file status not found")
		return
	}
	csvpath := filepath.Join(uploadpath, fileid, "ids.csv")
	if fileid == "" {
		csvpath = ""
	}
	service.Renderbyidsfile(w, r, csvpath, filestatus.Filename)
}

func showdbgitembyeventname(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("dbg open socket")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer conn.Close()
	defer logrus.Debug("dbg close socket")
	//升级后从中读出filter内容
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			logrus.Debug("read error:", err)
			return
		}
		var jsonmessage map[string]interface{}
		json.Unmarshal(p, &jsonmessage)
		fileid := jsonmessage["fileid"].(string)
		eventfilter := jsonmessage["eventname"].(string)

		userid, ok := util.CookieGet(r).Values["id"].(string)
		if !ok {
			logrus.Debug("userid not found")
			return
		}
		usersession, ok := sessionmanager.Get(userid)
		if !ok {
			logrus.Debug("usersession not found")
			return
		}
		logrus.Debug("current user id: ", userid)
		currentfilestatus, ok := usersession.FileStatusManager.Get(fileid)
		if !ok {
			logrus.Debug("file status not found")
			return
		}

		//some query eventfilter and fileid
		filtereddbgitems := dataaccess.DatabaseGetByEventName(currentfilestatus.Uid, eventfilter)
		conn.WriteJSON(filtereddbgitems)
	}
}

func concurrencyTest(w http.ResponseWriter, r *http.Request) {
	userid, ok := util.CookieGet(r).Values["id"].(string)
	if !ok {
		logrus.Debug("userid not found")
		return
	}
	fileuid := strings.ReplaceAll(uuid.New().String(), "-", "_") //strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err := os.Stat(filepath.Join(uploadpath, fileuid+".tar.gz"))
	logrus.Debug("newfile path:", filepath.Join(uploadpath, fileuid+".tar.gz"))
	if err != nil {
		cachefile, err := os.Create(filepath.Join(uploadpath, fileuid+".tar.gz"))
		if err != nil {
			logrus.Debug("create new fail:", err)
		}
		defer cachefile.Close()
		file, err := os.Open("./example/Log_20240618_092153.tar.gz")
		if err != nil {
			logrus.Debug("open file fail:", err)
		}
		_, err = cachefile.ReadFrom(file)
		if err != nil {
			logrus.Debug("cache fail:", err)
		}
		service.InitFileWithDBG(sessionmanager, pythonstatusmanager, cachequeue, fileuid, "Log_20240618_092153.tar.gz", uploadpath, 0, 0, userid)

		http.Redirect(w, r, "/uploadedfiles?filename="+"Log_20240618_092153.tar.gz", http.StatusSeeOther)
		logrus.Debug("redirect success")
	}
}

func main() {
	gob.Register(map[string]string{})
	http.HandleFunc("/cache/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".csv") {
			w.Header().Set("Content-Type", "text/csv")
		}
		http.StripPrefix("/cache", http.FileServer(http.Dir("cache"))).ServeHTTP(w, r)
	})
	//http.Handle("/cache/", http.StripPrefix("/cache/", http.FileServer(http.Dir("cache"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/template/", http.StripPrefix("/template/", http.FileServer(http.Dir("template"))))
	http.HandleFunc("/", indexentry)
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/results/dbg", showresults_dbg)
	http.HandleFunc("/results/ids", showresults_ids)
	http.HandleFunc("/uploadedfiles", uploadedfiles)
	http.HandleFunc("/clearcache", cleardata)
	http.HandleFunc("/ws", sockethandler_withfilter)
	http.HandleFunc("/dbgitembyeventfilter_ws", showdbgitembyeventname)
	http.HandleFunc("/concurrencytest", concurrencyTest)
	logrus.Debug("server start")
	err := http.ListenAndServe(localport, nil)
	if err != nil {
		logrus.Error("ListenAndServe: ", err)
	}

}
