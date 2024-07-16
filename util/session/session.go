package session

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
)

var key, _ = base64.RawStdEncoding.DecodeString("A+whGsPHGm88co6U+ozfFwR7oAsWaNK1VbfQxQxvTi0=")
var store = sessions.NewCookieStore(key) //securecookie.GenerateRandomKey(32))

func SessionInit(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		fmt.Println("error getting session:", err)
		return
	}
	session.Values["filename"] = []string{}
	session.Options.MaxAge = 0
	err = session.Save(r, w)
	if err != nil {
		fmt.Println("error saving session:", err)
		return
	}
}

func SessionAddFileHistory(w http.ResponseWriter, r *http.Request, filename string) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		fmt.Println("error add file:", err)
		return
	}
	session.Values["filename"] = append(session.Values["filename"].([]string), filename)
	err = session.Save(r, w)
	if err != nil {
		fmt.Println("error saving session:", err)
		return
	}
}

func SessionDeleteAllFileHistory(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		fmt.Println("error delete file:", err)
		return
	}
	session.Values["filename"] = []string{}
}
func SessionFileHistoryFilter(r *http.Request) ([]string, error) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		return nil, err
	}
	return session.Values["filename"].([]string), err
}
