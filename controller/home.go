package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/views/ybs"
	"strconv"
)

func (h *BaseHandler) HomePage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)

	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/login", 302)
			return
		}
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/setting", 302)
		return
	}

	btn, key, score := sdb.B2s(ctx.FormValue("btn")), sdb.B2s(ctx.FormValue("key")), sdb.B2s(ctx.FormValue("score"))
	if len(key) > 0 {
		_, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/", 302)
			return
		}
	}
	if len(score) > 0 {
		_, err := strconv.ParseUint(score, 10, 64)
		if err != nil {
			ctx.Redirect(h.App.Cf.Site.MainDomain+"/", 302)
			return
		}
	}

	cmd := "zrscan"
	if btn == "prev" {
		cmd = "zscan"
	}

	db := h.App.Db
	scf := h.App.Cf.Site

	topicPageInfo := model.GetTopicList(db, cmd, "topic_update", key, score, scf.PageShowNum)
	//topicPageInfo := model.GetTopicListSortById(db, cmd, "topic_update", key, score, scf.PageShowNum)

	evn := &ybs.HomePage{}
	evn.SiteCf = scf
	evn.Title = scf.Name
	evn.CurrentUser = curUser

	evn.SiteInfo = model.GetSiteInfo(db)
	evn.DefaultNode = model.Node{ID: 1}
	evn.NodeLst = model.NodeGetAll(h.App.Mc, db)
	evn.TopicPageInfo = topicPageInfo
	evn.TagCloud = model.GetTagsForSide(h.App.Mc, db, 100)
	evn.RangeTopicLst = rangeTopicLst[:]
	evn.RecentComment = model.CommentGetRecent(h.App.Mc, db, scf.RecentCommentNum)
	evn.LinkLst = model.LinkList(h.App.Mc, h.App.Db, false)

	if curUser.ID > 0 {
		evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
		if curUser.Flag >= model.FlagAdmin {
			evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
			evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)
		}
	}

	ctx.SetContentType("text/html; charset=utf-8")
	ybs.WritePageTemplate(ctx, evn)
}
