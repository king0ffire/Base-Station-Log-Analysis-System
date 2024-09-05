package dataaccess

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"webapp/util"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

var db *sql.DB

type DbgItem struct {
	Id        int
	Time      string
	Errortype string
	Device    string
	Info      string
	Eventname string
	Fileid    string
}

func init() {
	var err error
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/",
		util.ConfigMap["database"]["user"],
		util.ConfigMap["database"]["password"],
		util.ConfigMap["database"]["host"],
		util.ConfigMap["database"]["port"]))
	if err != nil {
		logrus.Fatal("fatal error:", err)
	}
	ctx, cancelfunc := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelfunc()
	_, err = db.ExecContext(ctx, "drop database if exists webapp")
	if err != nil {
		logrus.Fatalf("Error %s when drop DB\n", err)
	}
	_, err = db.ExecContext(ctx, "create database webapp")
	if err != nil {
		logrus.Fatalf("Error %s when creating DB\n", err)
	}

	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		util.ConfigMap["database"]["user"],
		util.ConfigMap["database"]["password"],
		util.ConfigMap["database"]["host"],
		util.ConfigMap["database"]["port"],
		util.ConfigMap["database"]["database"]))
	if err != nil {
		logrus.Fatal("fatal error:", err)
	}
	ctx, cancelfunc = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelfunc()
	_, err = db.ExecContext(ctx,
		"create table userinfo (id int auto_increment primary key, userid VARCHAR(255), unique(userid))")
	if err != nil {
		logrus.Fatalf("Error %s when creating table\n", err)
		return
	}
	_, err = db.ExecContext(ctx,
		"create table fileinfo (id int auto_increment primary key, fileid VARCHAR(255),unique(fileid), userid VARCHAR(255), foreign key (userid) references userinfo(userid) on delete cascade)")
	if err != nil {
		logrus.Fatalf("Error %s when creating table\n", err)
		return
	}
	logrus.Debug("database is inited")
}
func DatabaseDeleteUserinfo[useridtype comparable](userid useridtype) {
	var err error

	_, err = db.Exec("delete from userinfo where userid = ?", userid)
	if err != nil {
		logrus.Debugf("Error %s when delete userinfo\n", err)
		return
	}
}
func DatabaseAddUserinfo[useridtype comparable](userid useridtype) {
	var err error

	_, err = db.Exec("insert into userinfo values (null,?)", userid)
	if err != nil {
		logrus.Debugf("Error %s when add userinfo\n", err)
		return
	}
	logrus.Debug("Added userid to db:", userid)
}
func DatabaseDeleteFileinfo[fileidtype comparable](fileid fileidtype) {
	var err error

	_, err = db.Exec("delete from fileinfo where fileid = ?", fileid)
	if err != nil {
		logrus.Debugf("Error %s when delete fileinfo\n", err)
		return
	}
}

func DatabaseDeletedbgitemstable[fileidtype comparable](fileid fileidtype) {
	var err error

	_, err = db.Exec("drop table dbgitems_" + fmt.Sprintf("%v", fileid))
	if err != nil {
		logrus.Debugf("Error %s, when drop table dbgitems_%s\n", err, fmt.Sprintf("%v", fileid))
		return
	}
}
func DatabaseAddFileinfo[fileidtype comparable, useridtype comparable](fileid fileidtype, userid useridtype) {
	var err error

	_, err = db.Exec("insert into fileinfo values (null,?,?)", fileid, userid)
	if err != nil {
		logrus.Debugf("Error %s when %s insert fileinfo %s\n", err, fmt.Sprintf("%v", userid), fmt.Sprintf("%v", fileid))
		return
	}
	logrus.Debug("insert fileinfo table:", fileid)
}
func DatabaseGetByEventName[fileidtype comparable](fileid fileidtype, eventname string) []*DbgItem {
	var res = []*DbgItem{}
	rows, err := db.Query("select * from dbgitems_"+fmt.Sprintf("%v", fileid)+" where event = ?", eventname)
	if err != nil {
		logrus.Debug("fatal error when finding event from target tabel:", err)
		return res
	}
	defer rows.Close()

	for rows.Next() {
		var item = &DbgItem{}
		err := rows.Scan(&item.Id, &item.Time, &item.Errortype, &item.Device, &item.Info, &item.Eventname, &item.Fileid)
		if err != nil {
			logrus.Error("fatal error:", err)
		}
		res = append(res, item)
	}
	return res
}
