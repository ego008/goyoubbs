package controller

import (
	"github.com/ego008/goyoubbs/model"
	"net/http"
	"strings"
)

func (h *BaseHandler) SearchDetail(w http.ResponseWriter, r *http.Request) {
	currentUser, _ := h.CurrentUser(w, r)
	if currentUser.Id == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	q := r.FormValue("q")

	if len(q) == 0 {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	qLow := strings.ToLower(q)

	db := h.App.Db

	where := "title"
	if strings.HasPrefix(qLow, "c:") {
		where = "content"
		qLow = qLow[2:]
	}

	pageInfo := model.ArticleSearchList(db, where, qLow, h.App.Cf.Site.PageShowNum)

	type pageData struct {
		PageData
		Q        string
		PageInfo model.ArticlePageInfo
	}

	tpl := h.CurrentTpl(r)

	evn := &pageData{}
	evn.SiteCf = h.App.Cf.Site
	evn.Title = qLow + " - " + h.App.Cf.Site.Name
	evn.IsMobile = tpl == "mobile"

	evn.CurrentUser = currentUser
	evn.ShowSideAd = true
	evn.PageName = "category_detail"
	evn.HotNodes = model.CategoryHot(db, 20)
	evn.NewestNodes = model.CategoryNewest(db, 20)

	evn.Q = qLow
	evn.PageInfo = pageInfo

	h.Render(w, tpl, evn, "layout.html", "search.html")
}
