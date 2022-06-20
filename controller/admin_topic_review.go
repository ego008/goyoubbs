package controller

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/views/ybs"
	"strconv"
)

func (h *BaseHandler) AdminTopicReviewPage(ctx *fasthttp.RequestCtx) {
	curUser, _ := h.CurrentUser(ctx)
	if curUser.Flag < model.FlagAdmin {
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin", 302)
		return
	}

	scf := h.App.Cf.Site

	evn := &ybs.AdminTopicAdd{}
	evn.CurrentUser = curUser
	evn.SiteCf = scf
	evn.Title = "待审核帖子"
	evn.PageName = "admin_topic_review"

	act := sdb.B2s(ctx.FormValue("act"))
	var delKey []byte // 待删除的key
	// 取待审核信息
	var rec model.Topic
	rs := h.App.Db.Hscan(model.TopicReviewTbName, nil, 1)
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
		_ = h.App.Db.Hdel(model.TopicReviewTbName, delKey)
		// 删掉个人待审核列表
		_ = h.App.Db.Hdel("review_topic:"+strconv.FormatUint(rec.UserId, 10), delKey)
		ctx.Redirect(h.App.Cf.Site.MainDomain+"/admin/topic/review", 302)
		return
	}

	var author model.User
	if rec.UserId > 0 {
		author, _ = model.UserGetById(h.App.Db, rec.UserId)
	}
	if author.ID == 0 {
		author = curUser
	}

	evn.DefaultTopic = model.Topic{
		NodeId:  rec.NodeId,
		UserId:  author.ID,
		Title:   rec.Title,
		Content: rec.Content,
		AddTime: rec.AddTime,
	}
	if evn.DefaultTopic.NodeId == 0 {
		evn.DefaultTopic.NodeId = 1
	}

	evn.DefaultUser = author
	// evn.DefaultNode, _ = model.NodeGetById(h.App.Db, evn.DefaultTopic.NodeId)
	// evn.DefaultNode = model.Node{}
	evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db)

	// evn.UserLst = model.UserGetAllAdmin(h.App.Db)
	evn.UserLst = []model.User{author}

	evn.HasMsg = model.MsgCheckHasOne(h.App.Db, curUser.ID)
	evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
	evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)

	ctx.SetContentType("text/html; charset=utf-8")
	ybs.WritePageTemplate(ctx, evn)
}
