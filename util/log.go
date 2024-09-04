package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&CustomFormatter{})
	logfile, err := os.OpenFile("golang.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.SetOutput(logfile)
}

type CustomFormatter struct{}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05.000000")
	level := entry.Level.String()
	var caller string
	if entry.HasCaller() {
		caller = fmt.Sprintf("%s:%d", filepath.Base(entry.Caller.File), entry.Caller.Line)
	}
	log := fmt.Sprintf("%s %-5s %-25s   %s\n", timestamp, level, caller, entry.Message)
	return []byte(log), nil
}
