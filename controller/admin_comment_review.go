package controller

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"strconv"
	"strings"
)

func (h *BaseHandler) AdminCommentReviewPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin", 302)
		return
	}

	scf := h.App.Cf.Site
	db := h.App.Db

	evn := &admin.CommentEdit{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "待审核评论"
	evn.PageName = "admin_comment_review"

	act := sdb.B2s(ctx.QueryArgs().Peek("act"))
	var delKey []byte // 待删除的key
	// 取待审核信息
	var rec model.Comment
	rs := db.Hscan(model.CommentReviewTbName, nil, 1)
	if rs.OK() {
		rs.KvEach(func(key, value sdb.BS) {
			delKey = key
			err := json.Unmarshal(value, &rec)
			if err != nil {
				return
			}
		})
	}

	if act == "del" {
		// 删掉管理员列表
		_ = db.Hdel(model.CommentReviewTbName, delKey)
		// 删掉个人待审核列表
		_ = db.Hdel("review_comment:"+strconv.FormatUint(rec.UserId, 10), delKey)
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/comment/review", 302)
		return
	}

	var author model.User
	if rec.UserId > 0 {
		author, _ = model.UserGetById(db, rec.UserId)
	}
	if author.ID == 0 {
		author = evn.CurrentUser
	}

	evn.ReadMoreBreak = model.ReadMoreBreak
	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)
	evn.DefaultTopic = model.TopicGetById(db, rec.TopicId)
	evn.DefaultComment = model.CommentFmt{
		Comment:    rec,
		Name:       author.Name,
		AddTimeFmt: util.TimeFmt(rec.AddTime, ""),
		ContentFmt: rec.Content,
	}
	evn.DefaultUser = author

	evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	admin.WritePageTemplate(ctx, evn)
}

// AdminCommentReviewPost 管理员编辑与审核公用
func (h *BaseHandler) AdminCommentReviewPost(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		_, _ = ctx.WriteString(`{"Code":401,"Msg":"请先登录"}`)
		return
	}

	var rec model.Comment
	err := util.Bind(ctx, util.JSON, &rec)
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	rec.Content = strings.TrimSpace(rec.Content)
	if len(rec.Content) == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"评论内容不能为空"}`)
		return
	}

	db := h.App.Db

	// edit
	var isEdit bool
	if rec.ID > 0 {
		isEdit = true
	}

	var comment model.Comment
	if isEdit {
		comment = model.CommentGetById(db, rec.TopicId, rec.ID)
		if comment.ID == 0 {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"该 id 不存在"}`)
			return
		}
		comment.Content = rec.Content
		model.CommentSet(db, comment)
		// 删缓存
		h.App.Mc.Del([]byte("CommentGetRecent"))
		h.App.Mc.Del([]byte(model.CommentTbName + strconv.FormatUint(rec.TopicId, 10)))
		_, _ = ctx.WriteString(`{"Code":200,"Msg":"成功编辑"}`)
		return
	}

	comment = rec

	type response struct {
		model.NormalRsp
		Tid uint64
	}

	rsp := response{}
	rsp.Code = 200
	// 保存(通过审核)
	comment = model.CommentAdd(h.App.Mc, db, comment)
	rsp.Tid = comment.ID

	// 删除
	reviewKey := []byte(strconv.FormatInt(comment.AddTime, 10) + "_" + strconv.FormatUint(comment.TopicId, 10))
	// 管理员
	_ = db.Hdel(model.CommentReviewTbName, reviewKey)
	// 发表者
	_ = db.Hdel(model.CommentReviewTbName+":"+strconv.FormatUint(comment.UserId, 10), reviewKey)

	_ = json.NewEncoder(ctx).Encode(rsp)
}
