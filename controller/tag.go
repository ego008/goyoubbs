package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"net/url"
	"strconv"
	"strings"
)

func (h *BaseHandler) TagPage(ctx *fasthttp.RequestCtx) {
	db := h.App.Db
	scf := h.App.Cf.Site

	tagRaw := ctx.UserValue("tag").(string)
	if tag, err := url.QueryUnescape(tagRaw); err == nil {
		tagRaw = tag
	}
	tagLower := strings.ToLower(tagRaw)
	if !db.Hscan("tag:"+tagLower, nil, 1).OK() {
		// 该标签下文章数为0
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/", 302)
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

	topicPageInfo := model.GetTopicListArchives(db, cmd, "tag:"+tagLower, key, score, scf.PageShowNum)

	curUser, _ := h.CurrentUser(ctx)

	evn := &model.TagPage{}
	evn.SiteCf = scf
	evn.Title = "Tag: " + tagRaw + " - " + scf.Name
	evn.CurrentUser = curUser

	evn.Tag = tagLower
	evn.NodeLst = model.NodeGetAll(h.App.Mc, db)
	evn.TopicPageInfo = topicPageInfo
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

	model.WritePageTemplate(ctx, evn)
	ctx.SetContentType("text/html; charset=utf-8")

	//_ = h.Render(ctx, evn, "default/layout.html", "default/sidebar.html", "default/tag.html")
}
