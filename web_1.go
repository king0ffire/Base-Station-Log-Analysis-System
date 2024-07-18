package main

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"webapp/util"
	"webapp/util/file"
	"webapp/util/session"
	"webapp/util/socket"

	"github.com/gorilla/websocket"
)

const uploadpath = "./cache"
const localpost = ":9090"

func indexentry(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		session.SessionInit(w, r)

		t, err := template.ParseFiles("./templates/home/index.html")
		if err != nil {
			fmt.Println(err)
			return
		}
		t.Execute(w, struct {
			URL string
		}{
			URL: localhosturl,
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

		uid, created := file.MultiPartFileSaver(uploadpath, &multipartfile, multiparthandler)
		if !created {
			fmt.Println("created failed:", created)
		}
		existuid, ok := session.SessionAddFileHistory(w, r, uid, multiparthandler.Filename)
		if !ok {
			fmt.Println("file exist, use old:", existuid)
		} else {
			util.ParseFile(uid, uploadpath)
		}
		//同用户内不允许重名文件
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
			URL: localhosturl,
		})

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
		URL: localhosturl,
	})
}
func showresults_dbg(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_dbg")
	filename := r.URL.Query().Get("filename")
	nametoid := session.SessionGet(r).Values["nametoid"].(map[string]string)
	uid, ok := nametoid[filename]
	if !ok {
		fmt.Println("access file u not have")
		render404(w, r)
		return
	}
	csvpath := filepath.Join(uploadpath, uid, "dbg.csv")
	if filename == "" {
		csvpath = ""
	}
	util.Renderbycsvfile(w, r, csvpath, 1)
}

func showresults_ids(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_ids")
	filename := r.URL.Query().Get("filename")
	nametoid := session.SessionGet(r).Values["nametoid"].(map[string]string)
	uid, ok := nametoid[filename]
	if !ok {
		fmt.Println("access file u not have")
		render404(w, r)
		return
	}
	csvpath := filepath.Join(uploadpath, uid, "ids.csv")
	if filename == "" {
		csvpath = ""
	}
	util.Renderbycsvfile(w, r, csvpath, 2)
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
		URL:      localhosturl,
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
	socket.SocketManagerAdd(filter, session.SessionGet(r), conn)
	defer socket.SocketManagerDelete(conn)
	//建立连接后首先先把当前信息都导一下
	filelist, err := session.SessionFileHistoryFilter(r) //从当前的cookie中提取文件列表
	if err != nil {
		fmt.Println("filtering retrieve fail:", err)
		return
	}
	for _, uid := range filelist {
		if _, ok := file.FileStatusMapGet(uid); ok {
			util.AnnounceAllSocketsWithFile(uid)
		}
	}
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("websocket read error:", err)
			break
		}
		if string(message) == "clearcache" {
			util.Clearmycache(w, r, uploadpath)
			conn.WriteJSON([]interface{}{})
			fmt.Println("Empty Json returned")
		}
	}
}

func clearcache(w http.ResponseWriter, r *http.Request) {
	fmt.Println("clear cache")
	util.Clearmycache(w, r, uploadpath)
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

	err := http.ListenAndServe(localpost, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
