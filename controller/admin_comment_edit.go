package controller

import (
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"strconv"
)

func (h *BaseHandler) AdminCommentEditPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin", 302)
		return
	}

	scf := h.App.Cf.Site
	db := h.App.Db

	tid, cid := sdb.B2s(ctx.QueryArgs().Peek("tid")), sdb.B2s(ctx.QueryArgs().Peek("cid"))
	tidI, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin", 302)
		return
	}
	cidI, err := strconv.ParseUint(cid, 10, 64)
	if err != nil {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin", 302)
		return
	}

	comment := model.CommentGetById(db, tidI, cidI)

	evn := &admin.CommentEdit{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "评论修改"
	evn.PageName = "admin_comment_edit"

	author, _ := model.UserGetById(db, comment.UserId)

	if author.ID == 0 {
		author = evn.CurrentUser
	}

	evn.ReadMoreBreak = model.ReadMoreBreak
	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)
	evn.DefaultTopic = model.TopicGetById(db, comment.TopicId)
	evn.DefaultComment = model.CommentFmt{
		Comment:    comment,
		Name:       author.Name,
		AddTimeFmt: util.TimeFmt(comment.AddTime, ""),
		ContentFmt: comment.Content,
	}
	evn.DefaultUser = author

	if len(ctx.QueryArgs().Peek("back")) > 0 {
		evn.GoBack = true
	}

	evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	admin.WritePageTemplate(ctx, evn)
}
