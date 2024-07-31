package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

type DbgItem struct {
	Id        int
	Time      string
	Errortype string
	Device    string
	Info      string
	Eventname string
}

func main() {
	var err error
	db, err = sql.Open("mysql", "root:root123@/")
	if err != nil {
		fmt.Println("fatal error:", err)
	}
	_, err = db.Exec("use webapp")
	rows, err := db.Query("select id,fileid,COALESCE(userid, '') from fileinfo")
	if err != nil {
		fmt.Printf("Error %s when select \n", err)
		return
	}
	for rows.Next() {
		var item = &DbgItem{}
		err := rows.Scan(&item.Id, &item.Time, &item.Errortype)
		if err != nil {
			fmt.Println("fatal error:", err)
		}
		fmt.Println(item)
	}

}
func AddFileinfo(fileid string) {
	var err error
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()
	_, err = db.ExecContext(ctx, "insert into fileinfo values (null,?)", fileid)
	if err != nil {
		fmt.Printf("Error %s when creating DB\n", err)
		return
	}
	fmt.Println("Added fileid:", fileid)
}
func GetByEventName(eventname string) []*DbgItem {
	var res = []*DbgItem{}
	rows, err := db.Query("select * from dbg where eventname = ?", eventname)
	if err != nil {
		fmt.Println("fatal error:", err)
		return res
	}
	defer rows.Close()

	for rows.Next() {
		var item = &DbgItem{}
		err := rows.Scan(&item.Id, &item.Time, &item.Errortype, &item.Device, &item.Info, &item.Eventname)
		if err != nil {
			fmt.Println("fatal error:", err)
		}
		res = append(res, item)
	}
	return res
}
