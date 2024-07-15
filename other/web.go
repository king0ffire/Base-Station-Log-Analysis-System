package web_old

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

const uploadpath = "./cache"
const localhosturl = "localhost:9090"

type FileStatus struct {
	Max     int
	Current int
}

var FileStatusMap = make(map[string]*FileStatus) //for ids file
var socketStatusManager = make(map[*websocket.Conn]string)
var lock sync.RWMutex
var lock2 sync.RWMutex

func webentry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("webentry")
	if r.Method == "GET" {
		t, _ := template.ParseFiles("upload_mod.html")
		t.Execute(w, struct {
			URL string
		}{
			URL: localhosturl,
		})
	}
}

func indexentry(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles("./templates/home/index.html")
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
		if true { //err != nil {
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
			parsethefile(handler.Filename)
		}
		fmt.Println("redirect success")
		http.Redirect(w, r, "/parseornot?filename="+handler.Filename, http.StatusSeeOther)
	}
}

func parsethefile(Filename string) {
	pythoncmd := exec.Command("python", "./scripts/main.py", filepath.Join(uploadpath, Filename))
	outpipe, _ := pythoncmd.StdoutPipe()
	if err := pythoncmd.Start(); err != nil {
		fmt.Println("start python fail:", err)
		return
	}
	scanner := bufio.NewScanner(outpipe)
	lock.Lock()
	FileStatusMap[Filename] = &FileStatus{Max: 0, Current: 0}
	lock.Unlock()
	//announceSocketsOfFile(Filename)
	go func() {
		fmt.Println("Go thread start:", Filename)
		scanner.Scan()
		fmt.Println("Received first line of exection:", Filename, ":", scanner.Text())
		intText, err := strconv.Atoi(scanner.Text())
		if err != nil {
			fmt.Println("text conversion:", err)
			return
		}
		lock.Lock()
		FileStatusMap[Filename].Max = intText
		lock.Unlock()
		announceAllSocketsWithFilter()
		for scanner.Scan() {
			fmt.Println("python outputs:", scanner.Text())
			if scanner.Text() == "sctp_finished_one" {
				lock.Lock()
				FileStatusMap[Filename].Current++
				lock.Unlock()
				announceAllSocketsWithFilter()
			}
		}
		fmt.Println("Go thread end:", Filename)
	}()
}

func showresults_dbg(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_dbg")
	filename := r.URL.Query().Get("filename")
	filenamewithoutextension := filename
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	csvpath := filepath.Join(uploadpath, filenamewithoutextension, "dbg.csv")
	renderbycsvfile(w, r, csvpath, 1)
}

func showresults_ids(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_ids")
	filename := r.URL.Query().Get("filename")
	filenamewithoutextension := filename
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	csvpath := filepath.Join(uploadpath, filenamewithoutextension, "ids.csv")
	renderbycsvfile(w, r, csvpath, 2)
}
func renderbycsvfile(w http.ResponseWriter, r *http.Request, csvpath string, htmlheadertype int) {
	csvfile, err := os.Open(csvpath)
	if err != nil {
		fmt.Fprintln(w, "Ah-oh:", csvpath)
		fmt.Fprintln(w, "Ah-oh:", err)
		return
	}
	defer csvfile.Close()

	csvreader := csv.NewReader(csvfile)
	csvdata, err := csvreader.ReadAll()
	if err != nil {
		fmt.Fprintln(w, "Read failed:", err)
	}
	var headername string
	if htmlheadertype == 1 {
		headername = "DBG Event Count List"
	} else if htmlheadertype == 2 {
		headername = "IDS Capture Infomation"
	}
	t, _ := template.ParseFiles("show_mod.html")
	t.Execute(w, struct {
		Header       []string
		Data         [][]string
		Downloadlink string
		Htmlheader   string
	}{Data: func() [][]string {
		if len(csvdata) > 1 {
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

func parseornot(w http.ResponseWriter, r *http.Request) {
	fmt.Println("parseornot")

	t, _ := template.ParseFiles("parseornot_mod.html")

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
	socketManagerAdd(filter, conn)
	defer socketManagerDelete(conn)
	//建立连接后首先先把当前信息都导一下
	filteredfilestatuslist := []*FileStatus{}
	lock.RLock()
	for k, v := range FileStatusMap {
		if strings.Contains(k, filter) {
			filteredfilestatuslist = append(filteredfilestatuslist, v)
		}
	}
	lock.RUnlock()
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
func socketManagerAdd(filter string, conn *websocket.Conn) {
	lock2.Lock()
	socketStatusManager[conn] = filter
	lock2.Unlock()
}
func socketManagerDelete(conn *websocket.Conn) {
	lock2.Lock()
	delete(socketStatusManager, conn)
	lock2.Unlock()
}

/*
func socketJSONwriter(filename string, conn *websocket.Conn) {
	lock.RLock()
	value, existance := FileStatusMap[filename]
	lock.RUnlock()
	if existance {
		conn.WriteJSON(value)
	} else {
		fmt.Println("No map information")
		os.Stat(filepath.Join(uploadpath, filename))
	}
}


	func sockethandler(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}
		filename := string(p)
		socketEventadd(filename, conn)
		defer socketEventdelete(filename, conn)
		socketJSONwriter(filename, conn)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Println("read error:", err)
				break
			}
		}
	}

	func socketEventadd(filename string, conn *websocket.Conn) {
		lock.RLock()
		_, existance := FileStatusMap[filename]
		lock.RUnlock()
		if existance {
			lock.Lock()
			FileStatusMap[filename].Conns = append(FileStatusMap[filename].Conns, conn)
			lock.Unlock()
			fmt.Println("add success")
		}
	}

	func socketEventdelete(filename string, conn *websocket.Conn) {
		fmt.Println("delete success")
		lock.Lock()
		conns := FileStatusMap[filename].Conns
		for i, c := range conns {
			if c == conn {
				conns = append(conns[:i], conns[i+1:]...)
				break
			}
		}
		lock.Unlock()
	}

func announceSocketsOfFile(filename string) {
	lock.RLock()
	conns := FileStatusMap[filename].Conns
	lock.RUnlock()
	for _, c := range conns {
		socketJSONwriter(filename, c)
	}
}*/

func announceAllSocketsWithFilter() {
	lock2.RLock()
	for conn, filter := range socketStatusManager {
		filteredfilestatuslist := []*FileStatus{}
		lock.RLock()
		for k, v := range FileStatusMap {
			if strings.Contains(k, filter) {
				filteredfilestatuslist = append(filteredfilestatuslist, v)
			}
		}
		lock.RUnlock()
		conn.WriteJSON(filteredfilestatuslist)
	}
	lock2.RUnlock()
}

func testmain() {
	http.Handle("/cache/", http.StripPrefix("/cache/", http.FileServer(http.Dir("cache"))))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/template/", http.StripPrefix("/template/", http.FileServer(http.Dir("template"))))
	http.HandleFunc("/", indexentry)
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/results/dbg", showresults_dbg)
	http.HandleFunc("/results/ids", showresults_ids)
	http.HandleFunc("/parseornot", parseornot)
	http.HandleFunc("/ws", sockethandler_withfilter)

	err := http.ListenAndServe(localhosturl, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
