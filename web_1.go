package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"webapp/util"

	"github.com/gorilla/websocket"
)

const uploadpath = "./cache"
const localhosturl = "localhost:9090"

func indexentry(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
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
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			fmt.Println("upload fail:", err)
			return
		}
		defer file.Close()

		_, err = os.Stat("./cache/" + handler.Filename)
		if err != nil {
			cachefile, err := os.Create("./cache/" + handler.Filename)
			if err != nil {
				fmt.Println("open new fail:", err)
				return
			}
			defer cachefile.Close()

			_, err = cachefile.ReadFrom(file)
			if err != nil {
				fmt.Println("cache fail:", err)
				return
			}
			util.ParseFile(handler.Filename, uploadpath)
		} else {
			fmt.Println("file already exists, skip parsing")
			max, current, existence := util.CheckFileExist(handler.Filename)
			if !existence {
				fmt.Println("file created before web init")
				util.FileStatusMapLock.Lock()
				util.FileStatusMap[handler.Filename] = &util.FileStatus{Filename: handler.Filename, Max: max, Current: current}
				util.FileStatusMapLock.Unlock()
			}
		}
		fmt.Println("redirect success")
		http.Redirect(w, r, "/uploadedfiles?filename="+handler.Filename, http.StatusSeeOther)
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

func showresults_dbg(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_dbg")
	filename := r.URL.Query().Get("filename")
	filenamewithoutextension := filename
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	csvpath := filepath.Join(uploadpath, filenamewithoutextension, "dbg.csv")
	if filename == "" {
		csvpath = ""
	}
	util.Renderbycsvfile(w, r, csvpath, 1)
}

func showresults_ids(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_ids")
	filename := r.URL.Query().Get("filename")
	filenamewithoutextension := filename
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	csvpath := filepath.Join(uploadpath, filenamewithoutextension, "ids.csv")
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
	util.SocketManagerAdd(filter, conn)
	defer util.SocketManagerDelete(conn)
	//建立连接后首先先把当前信息都导一下
	filteredfilestatuslist := []*util.FileStatus{}
	util.FileStatusMapLock.RLock()
	for k, v := range util.FileStatusMap {
		if strings.Contains(k, filter) {
			filteredfilestatuslist = append(filteredfilestatuslist, v)
		}
	}
	util.FileStatusMapLock.RUnlock()
	conn.WriteJSON(filteredfilestatuslist)
	//关闭网页后这段就会退出
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Println("websocket read error:", err)
			break
		}
	}
}

func main() {
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
	http.HandleFunc("/ws", sockethandler_withfilter)

	err := http.ListenAndServe(localhosturl, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
