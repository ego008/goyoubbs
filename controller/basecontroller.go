package controller

import (
	"encoding/json"
	"errors"
	"github.com/ego008/goyoubbs/model"
	"github.com/ego008/goyoubbs/system"
	"github.com/ego008/youdb"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var mobileRegexp = regexp.MustCompile(`Mobile|iP(hone|od|ad)|Android|BlackBerry|IEMobile|Kindle|NetFront|Silk-Accelerated|(hpw|web)OS|Fennec|Minimo|Opera M(obi|ini)|Blazer|Dolfin|Dolphin|Skyfire|Zune`)

type (
	BaseHandler struct {
		App *system.Application
	}

	PageData struct {
		SiteCf        *system.SiteConf
		Title         string
		Keywords      string
		Description   string
		IsMobile      bool
		CurrentUser   model.User
		PageName      string // index/post_add/post_detail/...
		ShowPostTopAd bool
		ShowPostBotAd bool
		ShowSideAd    bool
		HotNodes      []model.CategoryMini
		NewestNodes   []model.CategoryMini
	}
	normalRsp struct {
		Retcode int    `json:"retcode"`
		Retmsg  string `json:"retmsg"`
	}
)

func (h *BaseHandler) Render(w http.ResponseWriter, tpl string, data interface{}, tplPath ...string) error {
	if len(tplPath) == 0 {
		return errors.New("File path can not be empty ")
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Server", "GoYouBBS")

	tplDir := h.App.Cf.Main.ViewDir + "/" + tpl + "/"
	tmpl := template.New("youbbs")
	for _, tpath := range tplPath {
		tmpl = template.Must(tmpl.ParseFiles(tplDir + tpath))
	}
	err := tmpl.Execute(w, data)

	return err
}

func (h *BaseHandler) CurrentUser(w http.ResponseWriter, r *http.Request) (model.User, error) {
	var user model.User
	ssValue := h.GetCookie(r, "SessionID")
	if len(ssValue) == 0 {
		return user, errors.New("SessionID cookie not found ")
	}
	z := strings.Split(ssValue, ":")
	uid := z[0]
	sessionID := z[1]

	rs := h.App.Db.Hget("user", youdb.DS2b(uid))
	if rs.State == "ok" {
		json.Unmarshal(rs.Data[0], &user)
		if sessionID == user.Session {
			h.SetCookie(w, "SessionID", ssValue, 365)
			return user, nil
		}
	}

	return user, errors.New("user not found")
}

func (h *BaseHandler) SetCookie(w http.ResponseWriter, name, value string, days int) error {
	encoded, err := h.App.Sc.Encode(name, value)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    encoded,
		Path:     "/",
		Secure:   h.App.Cf.Main.CookieSecure,
		HttpOnly: h.App.Cf.Main.CookieHttpOnly,
		Expires:  time.Now().UTC().AddDate(0, 0, days),
	})
	return err
}

func (h *BaseHandler) GetCookie(r *http.Request, name string) string {
	if cookie, err := r.Cookie(name); err == nil {
		var value string
		if err = h.App.Sc.Decode(name, cookie.Value, &value); err == nil {
			return value
		}
	}
	return ""
}

func (h *BaseHandler) DelCookie(w http.ResponseWriter, name string) {
	if len(name) > 0 {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Secure:   h.App.Cf.Main.CookieSecure,
			HttpOnly: h.App.Cf.Main.CookieHttpOnly,
			Expires:  time.Unix(0, 0),
		})
	}
}

func (h *BaseHandler) CurrentTpl(r *http.Request) string {
	tpl := "desktop"
	//tpl := "mobile"

	cookieTpl := h.GetCookie(r, "tpl")
	if len(cookieTpl) > 0 {
		if cookieTpl == "desktop" || cookieTpl == "mobile" {
			return cookieTpl
		}
	}

	ua := r.Header.Get("User-Agent")
	if len(ua) < 6 {
		return tpl
	}
	if mobileRegexp.MatchString(ua) {
		return "mobile"
	}
	return tpl
}
