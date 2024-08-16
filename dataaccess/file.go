package dataaccess

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func DeleteFileFromLocal(uploadpath string, uid string) error { //cache下的文件和文件夹的删除
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
