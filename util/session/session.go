package session

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

var key, _ = base64.RawStdEncoding.DecodeString("A+whGsPHGm88co6U+ozfFwR7oAsWaNK1VbfQxQxvTi0=")
var store = sessions.NewCookieStore(key) //securecookie.GenerateRandomKey(32))

func init() {
	store.MaxAge(0) //浏览器关闭后就啥都没了
}
func SessionInit(w http.ResponseWriter, r *http.Request) {
	//gob.Register(map[string]string{})
	session, err := store.Get(r, "session_name")
	fmt.Println(session.IsNew)
	if err != nil {
		fmt.Println("error init session:", err)
		return
	}
	if session.Values["filename"] == nil {
		session.Values["filename"] = []string{}
		session.Values["nametoid"] = map[string]string{}
		err = session.Save(r, w)
		fmt.Println("cookie init")
	} else {
		fmt.Println("cookie exist")
	}

	if err != nil {
		fmt.Println("error saving session:", err)
		return
	}
}

// 会话的可访问文件全部保存在cookie里
// false-> exist, return that uid
func SessionAddFileHistory(w http.ResponseWriter, r *http.Request, uid string, filename string) (string, bool) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		fmt.Println("error add file:", err)
		return "", true
	}
	if uid, ok := session.Values["nametoid"].(map[string]string)[filename]; ok {
		return uid, false
	}
	session.Values["filename"] = append(session.Values["filename"].([]string), uid)
	session.Values["nametoid"].(map[string]string)[filename] = uid
	err = session.Save(r, w)
	if err != nil {
		fmt.Println("error saving session:", err)
		return "", true
	}
	return "", true
}
func SessionDeleteAllFileHistory(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		fmt.Println("error delete file:", err)
		return
	}
	session.Values["filename"] = []string{}
	session.Values["nametoid"] = map[string]string{}
	err = session.Save(r, w)
	if err != nil {
		fmt.Println("error saving session:", err)
	}
}

func SessionFileHistoryFilter(r *http.Request) ([]string, error) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		return nil, err
	}
	return session.Values["filename"].([]string), err
}

func SessionGet(r *http.Request) *sessions.Session {
	session, err := store.Get(r, "session_name")
	if err != nil {
		fmt.Println("error getting session:", err)
		return nil
	}
	return session
}
