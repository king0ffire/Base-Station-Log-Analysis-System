package lowermanager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
	"webapp/util"
)

type Socket struct {
	Socket *net.Conn
	Lock   sync.Mutex
}

type SocketManager struct {
	Socket *Socket
}

func NewSocketManager() *SocketManager {
	return &SocketManager{
		Socket: NewSocket(),
	}
}

func NewSocket() *Socket {
	fmt.Println("waiting python server up")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", util.ConfigMap["socket"]["host"], util.ConfigMap["socket"]["port"]))
	for err != nil {
		time.Sleep(10 * time.Second)
		fmt.Println("waiting python server up")
		conn, err = net.Dial("tcp", fmt.Sprintf("%s:%s", util.ConfigMap["socket"]["host"], util.ConfigMap["socket"]["port"]))
	}
	fmt.Println("connect server success")
	return &Socket{
		Socket: &conn,
	}
}

func (s *Socket) NewPythonServerListener(handlefunc func(int, []byte)) {
	go func() { //new listen
		//buf := make([]byte, 1024)
		reader := bufio.NewReader(*s.Socket)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("lost connection to python server:", err)
				for i := 1; ; i++ {
					if i == 10 {
						panic("cannot reconnect to python server")
					}
					fmt.Println("reconnect to python server:", i)
					conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", util.ConfigMap["socket"]["host"], util.ConfigMap["socket"]["port"]))
					if err != nil {
						fmt.Printf("%v-th conn server failed, err:%v\n", i, err)
						time.Sleep(10 * time.Second)
						continue
					}
					s.Socket = &conn
					reader = bufio.NewReader(*s.Socket)
					break
				}
				fmt.Println("reconnect to python server success")
				continue
			}
			fmt.Println("read from python server:", line)
			handlefunc(len(line), []byte(line))
		}
	}()
}

func (s *Socket) WriteJSON(v interface{}) {
	jsondump, err := json.Marshal(v)
	if err != nil {
		fmt.Println("json dump failed:", err)
		return
	}
	jsondump = append(jsondump, '\n')
	(*s.Socket).Write(jsondump)

}
