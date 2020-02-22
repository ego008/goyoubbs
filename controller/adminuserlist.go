package controller

import (
	"encoding/json"
	"github.com/ego008/youdb"
	"github.com/rs/xid"
	"goyoubbs/model"
	"goyoubbs/util"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *BaseHandler) AdminUserList(w http.ResponseWriter, r *http.Request) {
	flag, btn, key := r.FormValue("flag"), r.FormValue("btn"), r.FormValue("key")
	if len(key) > 0 {
		_, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"key type err"}`))
			return
		}
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 99 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden}`))
		return
	}

	cmd := "hrscan"
	if btn == "prev" {
		cmd = "hscan"
	}

	db := h.App.Db

	if len(flag) == 0 {
		flag = "5"
	}

	pageInfo := model.UserListByFlag(db, cmd, "user_flag:"+flag, key, h.App.Cf.Site.PageShowNum)

	type pageData struct {
		PageData
		PageInfo model.UserPageInfo
		Flag     string
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = "用户列表"
	evn.IsMobile = tpl == "mobile"
	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "user_list"

	evn.PageInfo = pageInfo
	evn.Flag = flag

	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		token := xid.New().String()
		h.SetCookie(w, "token", token, 1)
	}

	h.Render(w, tpl, evn, "layout.html", "adminuserlist.html")
}

func (h *BaseHandler) AdminUserListPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"token cookie missed"}`))
		return
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 99 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden}`))
		return
	}

	type recForm struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	type response struct {
		normalRsp
	}

	decoder := json.NewDecoder(r.Body)
	var rec recForm
	err := decoder.Decode(&rec)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"json Decode err:` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()

	if len(rec.Name) == 0 || len(rec.Password) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"name or pw is empty"}`))
		return
	}
	nameLow := strings.ToLower(rec.Name)
	db := h.App.Db
	timeStamp := uint64(time.Now().UTC().Unix())

	if db.Hget("user_name2uid", []byte(nameLow)).State == "ok" {
		w.Write([]byte(`{"retcode":400,"retmsg":"name is exist"}`))
		return
	}

	userId, _ := db.HnextSequence("user")
	flag := 5

	uobj := model.User{
		Id:            userId,
		Name:          rec.Name,
		Password:      rec.Password,
		Flag:          flag,
		RegTime:       timeStamp,
		LastLoginTime: timeStamp,
	}

	uidStr := strconv.FormatUint(userId, 10)
	err = util.GenerateAvatar("male", rec.Name, 73, 73, "static/avatar/"+uidStr+".jpg")
	if err != nil {
		uobj.Avatar = "0"
	} else {
		uobj.Avatar = uidStr
	}

	jb, _ := json.Marshal(uobj)
	db.Hset("user", youdb.I2b(uobj.Id), jb)
	db.Hset("user_name2uid", []byte(nameLow), youdb.I2b(userId))
	db.Hset("user_flag:"+strconv.Itoa(flag), youdb.I2b(uobj.Id), []byte(""))

	rsp := response{}
	rsp.Retcode = 200
	json.NewEncoder(w).Encode(rsp)
}
