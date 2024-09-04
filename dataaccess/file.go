package dataaccess

import (
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func DeleteDirFromLocal(uploadpath string, uid string) error { //cache下的文件和文件夹的删除
	err := os.RemoveAll(filepath.Join(uploadpath, uid))
	logrus.Debug("first delete file directory: ", uid)
	if err != nil {
		logrus.Debug("first delete file failed:", err)
		go func() {
			time.Sleep(time.Second * 10)
			err2 := os.RemoveAll(filepath.Join(uploadpath, uid))
			logrus.Debug("retried delete file directory: ", uid)
			if err2 != nil {
				logrus.Debug("retry error:", err2)
			}
		}()
	}
	return err
}

func DeleteFileFromLocal(uploadpath string, uid string) error { //cache下的文件和文件夹的删除
	err := os.Remove(filepath.Join(uploadpath, uid+".tar.gz"))
	logrus.Debug("first delete file:", uid)
	if err != nil {
		logrus.Debug("first delete file failed:", err)
		go func() {
			time.Sleep(time.Second * 10)
			err2 := os.RemoveAll(filepath.Join(uploadpath, uid+".tar.gz"))
			logrus.Debug("retried delete file:", uid)
			if err2 != nil {
				logrus.Debug("retry error:", err2)
			}
		}()
	}
	return err
}

func MultiPartFileSaver(savepath string, file *multipart.File, handler *multipart.FileHeader) (string, bool) {
	uid := strings.ReplaceAll(uuid.New().String(), "-", "_") //strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err := os.Stat(filepath.Join(savepath, uid+".tar.gz"))
	logrus.Debug("newfile path:", filepath.Join(savepath, uid+".tar.gz"))
	if err != nil {
		cachefile, err := os.Create(filepath.Join(savepath, uid+".tar.gz"))
		if err != nil {
			logrus.Debug("create new fail:", err)
			return "", false
		}
		defer cachefile.Close()

		_, err = cachefile.ReadFrom(*file)
		if err != nil {
			logrus.Debug("cache fail:", err)
			return "", false
		}

		return uid, true
	} else {
		logrus.Debug("file already exists, fatal error, skip parsing")
		return "", false
		/*
			max, current, existence := util.CheckFileExist(handler.Filename)
			if !existence {
				fmt.Println("file created before web init")
				util.FileStatusMapLock.Lock()
				util.FileStatusMap[handler.Filename] = &util.FileStatus{Filename: handler.Filename, Max: max, Current: current}
				util.FileStatusMapLock.Unlock()
			}*/
	}
}
