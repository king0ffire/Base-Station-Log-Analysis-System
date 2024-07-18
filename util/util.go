package util

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"webapp/util/file"
	"webapp/util/pythonmanager"
	"webapp/util/session"
	"webapp/util/socket"

	"github.com/gorilla/websocket"
)

func FileListNameFilter(FileList []string, filter string) []string {
	result := []string{}
	for _, v := range FileList {
		if strings.Contains(v, filter) {
			result = append(result, v)
		}
	}
	return result
}
func StringListToMapValue(StringList []string, TargetMap map[string]*file.FileStatus) []*file.FileStatus {
	result := []*file.FileStatus{}
	for _, v := range StringList {
		result = append(result, TargetMap[v])
	}
	return result
}

func ParseFile(uid string, uploadpath string) {
	pythoncmd := exec.Command("python", "./scripts/main.py", filepath.Join(uploadpath, uid+".tar.gz"), "1")
	outpipe, _ := pythoncmd.StdoutPipe()
	if err := pythoncmd.Start(); err != nil {
		fmt.Println("start python fail:", err)
		return
	}
	pythonmanager.PythonProcessStatusMapAdd(uid, pythoncmd)
	scanner := bufio.NewScanner(outpipe)

	go func() {
		fmt.Println("Go thread start:", uid)
		defer fmt.Println("Go thread end:", uid)
		defer DeleteFileAll(uploadpath, uid)
		defer pythonmanager.PythonProcessStatusMapDelete(uid)

		scanner.Scan()
		fmt.Println("Received first line of exection:", uid, ":", scanner.Text())
		intText, err := strconv.Atoi(scanner.Text())
		if err != nil {
			fmt.Println("text conversion:", err)
			return
		}
		ok := file.FileStatusMapSet(uid, 0, intText)
		if !ok {
			Forcestop(uploadpath, uid)
			return
		}
		AnnounceAllSocketsWithFile(uid)
		for scanner.Scan() {
			fmt.Println("python outputs:", scanner.Text())
			if scanner.Text() == "sctp_finished_one" {
				value, existance := file.FileStatusMapGet(uid)
				if !existance {
					Forcestop(uploadpath, uid)
					return
				}
				ok := file.FileStatusMapSet(uid, value.Current+1, value.Max)
				if !ok {
					Forcestop(uploadpath, uid)
					return
				}
				AnnounceAllSocketsWithFile(uid)
			}
		}
	}()
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
func Sorted_dbg(data [][]string) {
	sort.Slice(data, func(i, j int) bool {
		num1, _ := strconv.Atoi(data[i][1])
		num2, _ := strconv.Atoi(data[j][1])
		return num1 > num2
	})
	return
}

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

func Clearmycache(w http.ResponseWriter, r *http.Request, uploadpath string) {
	oldsession := session.SessionGet(r)
	filelist := oldsession.Values["filename"].([]string)
	for _, v := range filelist {
		DeleteFileAll(uploadpath, v)
		fmt.Println("cleared cache:", v)
	}
}

func DeleteFileAll(uploadpath string, uid string) {

	os.RemoveAll(filepath.Join(uploadpath, uid))
	file.FileStatusMapDelete(uid)
}

func Forcestop(uploadpath string, uid string) {
	fmt.Println("cache might be cleared, stop", uid)
	cmdstatus, _ := pythonmanager.PythonProcessStatusMapGet(uid)
	if err := cmdstatus.Cmd.Process.Kill(); err != nil {
		fmt.Println("kill python process:", err)
	}
}
