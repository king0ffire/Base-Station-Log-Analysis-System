package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"webapp/util"
	"webapp/util/cookie"
	"webapp/util/database"
	"webapp/util/file"
	"webapp/util/pythonmanager"
	"webapp/util/session"
	"webapp/util/socket"

	"github.com/gorilla/websocket"
)

const uploadpath = "./cache"
const localport = ":9090"

var sessionmanager = session.NewManager[string, string, *websocket.Conn]()
var cachequeue = &file.FileCacheQueue[string, string]{}
var pythonprocessesmanager = pythonmanager.NewManager[string]()

func indexentry(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		isnew := cookie.CookieInit(w, r) //这个浏览器的唯一身份在此定义
		userid, ok := cookie.CookieGet(r).Values["id"].(string)
		if !ok {
			fmt.Println("userid not found")
			return
		}
		if isnew {
			sessionmanager.Add(userid)
			database.AddUserinfo(userid)
			fmt.Println("Add User:", userid)
		}
		t, err := template.ParseFiles("./templates/home/index.html")
		if err != nil {
			fmt.Println(err)
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
	fmt.Println("upload")
	if r.Method == "POST" {
		fmt.Println("收到上传的POST请求")
		r.ParseMultipartForm(256 << 20) //32MB
		multipartfile, multiparthandler, err := r.FormFile("uploadfile")
		if err != nil {
			fmt.Println("upload fail:", err)
			return
		}
		defer multipartfile.Close()
		userid, ok := cookie.CookieGet(r).Values["id"].(string)
		if !ok {
			fmt.Println("userid not found")
			return
		}
		fileuid, created := util.MultiPartFileSaver(uploadpath, &multipartfile, multiparthandler)
		if !created {
			fmt.Println("created failed:", created)
			return
		}
		util.OldFileCollection(sessionmanager, pythonprocessesmanager, cachequeue, fileuid, uploadpath, userid)
		util.AddFileToMemory(sessionmanager, pythonprocessesmanager, cachequeue, fileuid, multiparthandler.Filename, 0, 0, userid)
		/*
			existuid, ok := session.SessionAddFileHistory(w, r, uid, multiparthandler.Filename)
			if !ok {
				fmt.Println("file exist, use old:", existuid)
			} else {
				util.ParseFile(uid, uploadpath)
			}*/
		//同用户内允许重名文件
		util.ParseFile(sessionmanager, pythonprocessesmanager, cachequeue, fileuid, uploadpath, userid)

		http.Redirect(w, r, "/uploadedfiles?filename="+multiparthandler.Filename, http.StatusSeeOther)
		fmt.Println("redirect success")
	} else {
		t, err := template.ParseFiles("./templates/upload/upload.html")
		if err != nil {
			fmt.Println(err)
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
	fmt.Println("uploadedfiles")

	t, err := template.ParseFiles("./templates/dataanalyzer/basedata.html")
	if err != nil {
		fmt.Println(err)
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

func sockethandler_withfilter(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	//升级后从中读出filter内容
	_, p, err := conn.ReadMessage()
	if err != nil {
		log.Println("read error:", err)
		return
	}
	filter := string(p)
	userid, ok := cookie.CookieGet(r).Values["id"].(string)
	if !ok {
		fmt.Println("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		fmt.Println("usersession not found")
		return
	}
	fmt.Println("current user id: ", userid)
	usersession.SocketstatusManager.Add(conn, filter, cookie.CookieGet(r))
	defer usersession.SocketstatusManager.Delete(conn)
	currentsocket, ok := usersession.SocketstatusManager.Get(conn)
	if !ok {
		fmt.Println("socket not found")
		return
	}
	_, filelist := usersession.FileStatusManager.KeyAndValue()
	socketlist := []*socket.SocketStatus[*websocket.Conn]{currentsocket}
	fmt.Println("filelist length", len(filelist))
	util.AnnounceAllSocketsInUser(filelist, socketlist)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("websocket read error:", err)
			break
		}
		if string(message) == "clearcache" {
			clearcache(w, r)
			currentsocket.Lock.Lock()
			conn.WriteJSON([]interface{}{})
			currentsocket.Lock.Unlock()
			fmt.Println("Empty Json returned")
		}
	}
}

func clearcache(w http.ResponseWriter, r *http.Request) {
	fmt.Println("clear cache")
	userid, ok := cookie.CookieGet(r).Values["id"].(string)
	if !ok {
		fmt.Println("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		fmt.Println("usersession not found")
		return
	}
	_, userholdingfiles := usersession.FileStatusManager.KeyAndValue()
	for _, v := range userholdingfiles {
		pythonprocessesmanager.Delete(v.Uid)
		cachequeue.Delete(v.Uid)
		usersession.FileStatusManager.Delete(v.Uid)
		util.DeleteFileFromLocal(uploadpath, v.Uid)
		fmt.Println("cleared cache:", v.Uid)
	}
}

func render404(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./templates/404.html")
	if err != nil {
		fmt.Println(err)
		return
	}
	t.Execute(w, struct {
		URL string
	}{
		URL: localport,
	})
}
func showresults_dbg(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_dbg")
	fileid := r.URL.Query().Get("fileid")
	userid, ok := cookie.CookieGet(r).Values["id"].(string)
	if !ok {
		fmt.Println("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		fmt.Println("usersession not found")
		return
	}
	currentfilestatus, ok := usersession.FileStatusManager.Get(fileid)
	if !ok {
		fmt.Println("file status not found")
		return
	}
	csvpath := filepath.Join(uploadpath, fileid, "dbg.csv")
	csvpath_acc := filepath.Join(uploadpath, fileid, "accounting.csv")
	if fileid == "" {
		csvpath = ""
	}
	util.Renderbydbgfile(w, r, csvpath, csvpath_acc, currentfilestatus.Filename)
}

func showresults_ids(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_ids")
	fileid := r.URL.Query().Get("fileid")
	userid, ok := cookie.CookieGet(r).Values["id"].(string)
	if !ok {
		fmt.Println("userid not found")
		return
	}
	usersession, ok := sessionmanager.Get(userid)
	if !ok {
		fmt.Println("usersession not found")
		return
	}
	_, ok = usersession.FileStatusManager.Get(fileid)
	if !ok {
		fmt.Println("file status not found")
		return
	}
	csvpath := filepath.Join(uploadpath, fileid, "ids.csv")
	if fileid == "" {
		csvpath = ""
	}
	util.Renderbyidsfile(w, r, csvpath)
}

func showdbgitembyeventname(w http.ResponseWriter, r *http.Request) {
	fmt.Println("dbg open socket")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	defer fmt.Println("dbg close socket")
	//升级后从中读出filter内容
	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}
		var jsonmessage map[string]interface{}
		json.Unmarshal(p, &jsonmessage)
		fileid := jsonmessage["fileid"].(string)
		eventfilter := jsonmessage["eventname"].(string)

		userid, ok := cookie.CookieGet(r).Values["id"].(string)
		if !ok {
			fmt.Println("userid not found")
			return
		}
		usersession, ok := sessionmanager.Get(userid)
		if !ok {
			fmt.Println("usersession not found")
			return
		}
		fmt.Println("current user id: ", userid)
		currentfilestatus, ok := usersession.FileStatusManager.Get(fileid)
		if !ok {
			fmt.Println("file status not found")
			return
		}

		//some query eventfilter and fileid
		filtereddbgitems := database.GetByEventName(currentfilestatus.Uid, eventfilter)
		conn.WriteJSON(filtereddbgitems)
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
	http.HandleFunc("/clearcache", clearcache)
	http.HandleFunc("/ws", sockethandler_withfilter)
	http.HandleFunc("/dbgitembyeventfilter_ws", showdbgitembyeventname)

	err := http.ListenAndServe(localport, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
