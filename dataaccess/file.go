package dataaccess

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func DeleteDirFromLocal(uploadpath string, uid string) error { //cache下的文件和文件夹的删除
	err := os.RemoveAll(filepath.Join(uploadpath, uid))
	if err != nil {
		go func() {
			time.Sleep(time.Second * 1)
			err2 := os.RemoveAll(filepath.Join(uploadpath, uid))
			fmt.Println("retried delete file")
			if err2 != nil {
				fmt.Println("retry error:", err2)
			}
		}()
	}
	err = os.Remove(filepath.Join(uploadpath, uid+".tar.gz"))
	if err != nil {
		fmt.Println("delete tar.gz error:", err)
	}
	return err
}

func DeleteFileFromLocal(uploadpath string, uid string) error { //cache下的文件和文件夹的删除
	err := os.Remove(filepath.Join(uploadpath, uid+".tar.gz"))
	if err != nil {
		go func() {
			time.Sleep(time.Second * 1)
			err2 := os.RemoveAll(filepath.Join(uploadpath, uid))
			fmt.Println("retried delete file")
			if err2 != nil {
				fmt.Println("retry error:", err2)
			}
		}()
	}
	err = os.Remove(filepath.Join(uploadpath, uid+".tar.gz"))
	if err != nil {
		fmt.Println("delete tar.gz error:", err)
	}
	return err
}

func MultiPartFileSaver(savepath string, file *multipart.File, handler *multipart.FileHeader) (string, bool) {
	uid := strings.ReplaceAll(uuid.New().String(), "-", "_") //strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err := os.Stat(filepath.Join(savepath, uid+".tar.gz"))
	fmt.Println("newfile path:", filepath.Join(savepath, uid+".tar.gz"))
	if err != nil {
		cachefile, err := os.Create(filepath.Join(savepath, uid+".tar.gz"))
		if err != nil {
			fmt.Println("create new fail:", err)
			return "", false
		}
		defer cachefile.Close()

		_, err = cachefile.ReadFrom(*file)
		if err != nil {
			fmt.Println("cache fail:", err)
			return "", false
		}

		return uid, true
	} else {
		fmt.Println("file already exists, fatal error, skip parsing")
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
