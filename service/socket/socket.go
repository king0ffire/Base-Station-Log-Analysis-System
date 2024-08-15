package socket

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type PythonServerBySocket struct {
	Socket *net.Conn
	Lock   sync.Mutex
}

func NewSocket() *PythonServerBySocket {
	fmt.Println("conn server")
	conn, err := net.Dial("tcp", "127.0.0.1:9091")
	if err != nil {
		fmt.Printf("conn server failed, err:%v\n", err)
		return nil
	}
	fmt.Println("conn server success")
	return &PythonServerBySocket{
		Socket: &conn,
	}
}

func NewPythonServerListener(conn *net.Conn, handlefunc func(int, []byte)) {
	go func() { //new listen
		buf := make([]byte, 1024)
		reader := bufio.NewReader(*conn)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				fmt.Println("lost connection to python server:", err)
				return
			}
			fmt.Println("read from python server:", string(buf[:n]))
			handlefunc(n, buf)
		}
	}()
}

func (s *PythonServerBySocket) WriteJSON(v interface{}) {
	jsondump, err := json.Marshal(v)
	if err != nil {
		fmt.Println("json dump failed:", err)
		return
	}
	jsondump = append(jsondump, '\n')
	(*s.Socket).Write(jsondump)

}
