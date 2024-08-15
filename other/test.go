package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

func main() {
	// 1、与服务端建立连接

	conn, err := net.Dial("tcp", "127.0.0.1:9091")
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return
	}

	fmt.Println("conn server success")
	s := make(map[string]string)
	s["function"] = "dbg"
	s["filelocation"] = "../../Log_20240618_092153.tar.gz"
	jsondump, err := json.Marshal(s)
	startT := time.Now()
	conn.Write(jsondump)

	buf := make([]byte, 1024)
	reader := bufio.NewReader(conn)
	fmt.Println("send success")
	n, err := reader.Read(buf)
	tc := time.Since(startT)
	fmt.Printf("time cost = %v\n", tc)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(buf)
	var resmap map[string]interface{}
	err = json.Unmarshal(buf[:n], &resmap)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resmap)

}
