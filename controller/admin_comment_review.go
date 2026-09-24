package controller

import (
	"bytes"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminCommentReviewPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	scf := h.App.Cf.Site
	db := h.App.Db

	evn := &admin.CommentEdit{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "待审核评论"
	evn.PageName = "admin_comment_review"

	act := c.Query("act")
	var delKey []byte // 待删除的key
	// 取待审核信息
	var rec model.Comment

	_ = db.View(func(tx *bbolt.Tx) error {
		_ = db.HScanFunc(tx, model.CommentReviewTbName, nil, 1, func(key, val []byte) bool {
			delKey = bytes.Clone(key)
			_ = json.Unmarshal(val, &rec)
			return true
		})
		return nil
	})

	if act == "del" {
		_ = db.Update(func(tx *bbolt.Tx) error {
			// 删掉管理员列表
			_ = db.HDel(tx, model.CommentReviewTbName, delKey)
			// 删掉个人待审核列表
			_ = db.HDel(tx, "review_comment:"+strconv.FormatUint(rec.UserId, 10), delKey)
			return nil
		})
		c.Redirect(302, "/admin/comment/review")
		return
	}

	var author model.User

	_ = db.View(func(tx *bbolt.Tx) error {
		if rec.UserId > 0 {
			author, _ = model.UserGetById(db, tx, rec.UserId)
		}
		if author.ID == 0 {
			author = evn.CurrentUser
		}

		evn.ReadMoreBreak = model.ReadMoreBreak
		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)
		evn.DefaultTopic = model.TopicGetById(db, tx, rec.TopicId)
		rec.Content = html.EscapeString(rec.Content) // 转义
		evn.DefaultComment = model.CommentFmt{
			Comment:    rec,
			Name:       author.Name,
			AddTimeFmt: util.TimeFmt(rec.AddTime, ""),
			ContentFmt: rec.Content,
		}
		evn.DefaultUser = author

		evn.HasMsg = model.MsgCheckHasOne(db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)

		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}

// AdminCommentReviewPost 管理员编辑与审核公用
func (h *BaseHandler) AdminCommentReviewPost(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.String(200, `{"Code":401,"Msg":"请先登录"}`)
		return
	}

	var rec model.Comment
	err := util.Bind(c, util.JSON, &rec)
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	rec.Content = strings.TrimSpace(rec.Content)
	if len(rec.Content) == 0 {
		c.String(200, `{"Code":400,"Msg":"评论内容不能为空"}`)
		return
	}

	db := h.App.Db

	// edit
	var isEdit bool
	if rec.ID > 0 {
		isEdit = true
	}

	type response struct {
		model.NormalRsp
		Tid uint64
	}

	rsp := response{}
	rsp.Code = 200

	_ = db.Update(func(tx *bbolt.Tx) error {
		var comment model.Comment
		if isEdit {
			comment = model.CommentGetById(db, tx, rec.TopicId, rec.ID)
			if comment.ID == 0 {
				c.String(200, `{"Code":400,"Msg":"该 id 不存在"}`)
				return nil
			}
			comment.Content = rec.Content
			model.CommentSet(db, tx, comment)
			// 删缓存
			h.App.Mc.Del([]byte("CommentGetRecent"))
			h.App.Mc.Del([]byte(model.CommentTbName + strconv.FormatUint(rec.TopicId, 10)))
			c.String(200, `{"Code":200,"Msg":"成功编辑"}`)
			return nil
		}

		comment = rec

		// 保存(通过审核)
		comment = model.CommentAdd(h.App.Mc, db, tx, comment)
		rsp.Tid = comment.ID

		// 删除
		reviewKey := []byte(strconv.FormatInt(comment.AddTime, 10) + "_" + strconv.FormatUint(comment.TopicId, 10))
		// 管理员
		_ = db.HDel(tx, model.CommentReviewTbName, reviewKey)
		// 发表者
		_ = db.HDel(tx, model.CommentReviewTbName+":"+strconv.FormatUint(comment.UserId, 10), reviewKey)
		return nil
	})

	_ = json.NewEncoder(c.Writer).Encode(rsp)
}
