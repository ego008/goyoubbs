package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"strings"
)

func (h *BaseHandler) SearchPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)

	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
			return
		}
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/setting", 302)
		return
	}

	//if curUser.ID == 0 {
	//	ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
	//	return
	//}

	q := strings.TrimSpace(sdb.B2s(ctx.FormValue("q")))
	if len(q) == 0 {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/", 302)
		return
	}

	scf := h.App.Cf.Site

	qLow := strings.ToLower(q)

	where := "title"
	if strings.HasPrefix(qLow, "c:") {
		where = "content"
		qLow = strings.TrimSpace(qLow[2:])
		if len(qLow) == 0 {
			ctx.Redirect(scf.MainDomain+"/", fasthttp.StatusSeeOther)
			return
		}
	}

	db := h.App.Db

	var pageInfo model.TopicPageInfo
	mcKey := []byte("search:" + where + ":" + qLow)
	if _, exist := util.ObjCachedGet(h.App.Mc, mcKey, &pageInfo, false); !exist {
		pageInfo = model.SearchTopicList(h.App.Mc, db, q, scf.PageShowNum)
		// set to mc
		util.ObjCachedSet(h.App.Mc, mcKey, pageInfo)
	}

	evn := &ybs.SearchPage{}
	evn.SiteCf = scf
	evn.Title = "搜索: " + q + " - " + scf.Name
	evn.CurrentUser = curUser

	evn.Q = q
	evn.NodeLst = model.NodeGetAll(h.App.Mc, db)
	evn.TopicPageInfo = pageInfo
	evn.TagCloud = model.GetTagsForSide(h.App.Mc, db, 100)
	evn.RangeTopicLst = rangeTopicLst[:]
	evn.RecentComment = model.CommentGetRecent(h.App.Mc, db, scf.RecentCommentNum)

	if curUser.ID > 0 {
		evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
		if curUser.Flag >= model.FlagAdmin {
			evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
			evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)
		}
	}

	ybs.WritePageTemplate(ctx, evn)
	ctx.SetContentType("text/html; charset=utf-8")
}
