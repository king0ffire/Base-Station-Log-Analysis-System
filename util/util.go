package util

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type FileStatus struct {
	Filename string
	Max      int
	Current  int
}

var FileStatusMap = make(map[string]*FileStatus) //for ids file
var SocketStatusManager = make(map[*websocket.Conn]string)
var FileStatusMapLock sync.RWMutex
var SocketStatusManagerLock sync.RWMutex

func CheckFileExist(filename string) (int, int, bool) {
	FileStatusMapLock.RLock()
	value, err := FileStatusMap[filename]
	FileStatusMapLock.RUnlock()
	if err {
		return value.Current, value.Max, err
	}
	return 0, 0, err
}
func ParseFile(Filename string, Uploadpath string) {
	pythoncmd := exec.Command("python", "./scripts/main.py", filepath.Join(Uploadpath, Filename))
	outpipe, _ := pythoncmd.StdoutPipe()
	if err := pythoncmd.Start(); err != nil {
		fmt.Println("start python fail:", err)
		return
	}
	scanner := bufio.NewScanner(outpipe)
	FileStatusMapLock.Lock()
	FileStatusMap[Filename] = &FileStatus{Filename: Filename, Max: 0, Current: 0}
	FileStatusMapLock.Unlock()
	go func() {
		fmt.Println("Go thread start:", Filename)
		scanner.Scan()
		fmt.Println("Received first line of exection:", Filename, ":", scanner.Text())
		intText, err := strconv.Atoi(scanner.Text())
		if err != nil {
			fmt.Println("text conversion:", err)
			return
		}
		FileStatusMapLock.Lock()
		FileStatusMap[Filename].Max = intText
		FileStatusMapLock.Unlock()
		AnnounceAllSocketsWithFilter()
		for scanner.Scan() {
			fmt.Println("python outputs:", scanner.Text())
			if scanner.Text() == "sctp_finished_one" {
				FileStatusMapLock.Lock()
				FileStatusMap[Filename].Current++
				FileStatusMapLock.Unlock()
				AnnounceAllSocketsWithFilter()
			}
		}
		fmt.Println("Go thread end:", Filename)
	}()
}

func Sorted_dbg(data [][]string) {
	sort.Slice(data, func(i, j int) bool {
		num1, _ := strconv.Atoi(data[i][1])
		num2, _ := strconv.Atoi(data[j][1])
		return num1 > num2
	})
	return
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

func SocketManagerAdd(filter string, conn *websocket.Conn) {
	SocketStatusManagerLock.Lock()
	SocketStatusManager[conn] = filter
	SocketStatusManagerLock.Unlock()
}
func SocketManagerDelete(conn *websocket.Conn) {
	SocketStatusManagerLock.Lock()
	delete(SocketStatusManager, conn)
	SocketStatusManagerLock.Unlock()
}

func AnnounceAllSocketsWithFilter() {
	SocketStatusManagerLock.RLock()
	for conn, filter := range SocketStatusManager {
		filteredfilestatuslist := []*FileStatus{}
		FileStatusMapLock.RLock()
		for k, v := range FileStatusMap {
			if strings.Contains(k, filter) {
				filteredfilestatuslist = append(filteredfilestatuslist, v)
			}
		}
		FileStatusMapLock.RUnlock()
		conn.WriteJSON(filteredfilestatuslist)
	}
	SocketStatusManagerLock.RUnlock()
}
