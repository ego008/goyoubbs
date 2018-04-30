/*
MIT License

Copyright (c) 2017

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"encoding/json"
	"github.com/goxforum/xforum/pkg/model"
	"github.com/rs/xid"
	"net/http"
	"strconv"
	"strings"
)

func (h *BaseHandler) AdminLinkList(w http.ResponseWriter, r *http.Request) {
	lid := r.FormValue("lid")

	db := h.App.Db

	var lobj model.Link
	if len(lid) > 0 {
		_, err := strconv.ParseUint(lid, 10, 64)
		if err != nil {
			w.Write([]byte(`{"retcode":400,"retmsg":"key type err"}`))
			return
		}

		lobj = model.LinkGetById(db, lid)
		if lobj.Id == 0 {
			w.Write([]byte(`{"retcode":404,"retmsg":"id not found"}`))
			return
		}
	} else {
		lobj.Score = 10
	}

	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		w.Write([]byte(`{"retcode":401,"retmsg":"authored err"}`))
		return
	}
	if currentUser.Flag < 99 {
		w.Write([]byte(`{"retcode":403,"retmsg":"flag forbidden"}`))
		return
	}

	type pageData struct {
		PageData
		Items []model.Link
		Lobj  model.Link
	}

	tpl := h.CurrentTpl(r)
	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = "链接列表"
	evn.IsMobile = tpl == "mobile"
	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "user_list"

	evn.Items = model.LinkList(db, true)
	evn.Lobj = lobj

	token := h.GetCookie(r, "token")
	if len(token) == 0 {
		token := xid.New().String()
		h.SetCookie(w, "token", token, 1)
	}

	h.Render(w, tpl, evn, "layout.html", "adminlinklist.html")
}

func (h *BaseHandler) AdminLinkListPost(w http.ResponseWriter, r *http.Request) {
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

	type response struct {
		normalRsp
	}

	decoder := json.NewDecoder(r.Body)
	var rec model.Link
	err := decoder.Decode(&rec)
	if err != nil {
		w.Write([]byte(`{"retcode":400,"retmsg":"json Decode err:` + err.Error() + `"}`))
		return
	}
	defer r.Body.Close()

	rec.Name = strings.TrimSpace(rec.Name)
	rec.Url = strings.TrimSpace(rec.Url)

	if len(rec.Name) == 0 || len(rec.Url) == 0 {
		w.Write([]byte(`{"retcode":400,"retmsg":"missed arg"}`))
		return
	}

	model.LinkSet(h.App.Db, rec)

	rsp := response{}
	rsp.Retcode = 200
	json.NewEncoder(w).Encode(rsp)
}
