package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"webapp/service/accounting"
	"webapp/util"
)

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
