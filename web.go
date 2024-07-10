package main

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
)

const uploadpath = "./cache"

type FileStatus struct {
	Max     int
	Current int
}

var FileStatusMap = make(map[string]*FileStatus)

func webentry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("webentry")
	if r.Method == "GET" {
		t, _ := template.ParseFiles("upload.html")
		t.Execute(w, nil)
	}
}

func upload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("upload")
	if r.Method == "POST" {
		fmt.Println("收到上传的POST请求")
		r.ParseMultipartForm(64 << 20) //64Mbits
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

			parsethefile(handler.Filename)

		}
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
	go func() {
		fmt.Println("Go thread start:", Filename)
		scanner.Scan()
		fmt.Println("Received first line of exection:", Filename, ":", scanner.Text())
		intText, err := strconv.Atoi(scanner.Text())
		if err != nil {
			fmt.Println("text conversion:", err)
			return
		}
		FileStatusMap[Filename] = &FileStatus{Max: intText, Current: 0}
		for scanner.Scan() {
			fmt.Println("python outputs:", scanner.Text())
			if scanner.Text() == "sctp_finished_one" {
				(*FileStatusMap[Filename]).Current++
			}
		}
		fmt.Println("Go thread end:", Filename)
	}()
}

func checkstatus(Filename string) (int, int) {
	value, existance := FileStatusMap[Filename]
	if existance {
		return value.Current, value.Max
	}
	return 0, 0
}

func showresults_dbg(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_dbg")
	filename := r.URL.Query().Get("filename")
	filenamewithoutextension := filename
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	csvpath := filepath.Join(uploadpath, filenamewithoutextension, "dbg.csv")
	renderbycsvfile(w, r, csvpath)
}
func renderbycsvfile(w http.ResponseWriter, r *http.Request, csvpath string) {
	csvfile, err := os.Open(csvpath)
	if err != nil {
		fmt.Fprintln(w, "Ah-oh:", csvpath)
		fmt.Fprintln(w, "Ah-oh:", err)
		return
	}
	defer csvfile.Close()

	csvreader2 := csv.NewReader(csvfile)
	csvdata2, err := csvreader2.ReadAll()
	if err != nil {
		fmt.Fprintln(w, "Read failed:", err)
	}

	t, _ := template.ParseFiles("show.html")
	t.Execute(w, struct {
		Header1      []string
		Data1        [][]string
		Downloadlink string
	}{Data1: csvdata2,
		Header1:      []string{"Event name", "Counts"},
		Downloadlink: "../" + strings.ReplaceAll(csvpath, "\\", "/"),
	})
}

func showresults_ids(w http.ResponseWriter, r *http.Request) {
	fmt.Println("showresults_ids")
	filename := r.URL.Query().Get("filename")
	filenamewithoutextension := filename
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	filenamewithoutextension = strings.TrimSuffix(filenamewithoutextension, path.Ext(filenamewithoutextension))
	csvpath := filepath.Join(uploadpath, filenamewithoutextension, "ids.csv")
	renderbycsvfile(w, r, csvpath)
}
func parseornot(w http.ResponseWriter, r *http.Request) {
	fmt.Println("parseornot")
	t, _ := template.ParseFiles("parseornot.html")
	t.Execute(w, r.URL.Query().Get("filename"))
}
func main() {
	http.Handle("/cache/", http.StripPrefix("/cache/", http.FileServer(http.Dir("cache"))))
	http.HandleFunc("/", webentry)
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/results/dbg", showresults_dbg)
	http.HandleFunc("/results/ids", showresults_ids)
	http.HandleFunc("/parseornot", parseornot)
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
