package util

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
)

var key, _ = base64.RawStdEncoding.DecodeString("A+whGsPHGm88co6U+ozfFwR7oAsWaNK1VbfQxQxvTi0=")
var store = sessions.NewCookieStore(key) //securecookie.GenerateRandomKey(32))

func init() {
	store.MaxAge(0) //浏览器关闭后就啥都没了
	store.Options.Secure = false
	store.Options.SameSite = http.SameSiteLaxMode
}

func GenerateNewId(w http.ResponseWriter, r *http.Request, cook *sessions.Session) {
	cook.Values["id"] = strings.ReplaceAll(uuid.New().String(), "-", "_")
	//strconv.FormatInt(time.Now().UnixNano(), 10)
	err := cook.Save(r, w)
	if err != nil {
		logrus.Error("error saving cookie:", err)
	}
	logrus.Debug("cookie init")
}

/*
// 会话的可访问文件全部保存在cookie里
// false-> exist, return that uid

	func CookieAdd(w http.ResponseWriter, r *http.Request, uid string, filename string) (string, bool) {
		session, err := store.Get(r, "session_name")
		if err != nil {
			logrus.Debug("error add file:", err)
			return "", true
		}
		if uid, ok := session.Values["nametoid"].(map[string]string)[filename]; ok {
			return uid, false
		}
		session.Values["filename"] = append(session.Values["filename"].([]string), uid)
		session.Values["nametoid"].(map[string]string)[filename] = uid
		err = session.Save(r, w)
		if err != nil {
			logrus.Debug("error saving session:", err)
			return "", true
		}
		return "", true
	}
*/
func CookieDeleteAll(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "session_name")
	if err != nil {
		logrus.Error("error trying delete file:", err)
		return
	}
	session.Values["filename"] = []string{}
	session.Values["nametoid"] = map[string]string{}
	err = session.Save(r, w)
	if err != nil {
		logrus.Error("error saving session:", err)
	}
}

func CookieGet(r *http.Request) *sessions.Session {
	session, err := store.Get(r, "session_name")
	if err != nil {
		logrus.Error("error getting session:", err)
		return nil
	}
	return session
}
