package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) MemberNamePage(c *gin.Context) {
	// 内容 @用户名 的链接
	uNameRaw := strings.TrimSpace(c.Param("uname"))
	if uName, err := url.QueryUnescape(uNameRaw); err == nil {
		uNameRaw = uName
	}

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {

		user, err := model.UserGetByName(h.App.Db, tx, uNameRaw)
		if err != nil {
			unUint64, err := strconv.ParseUint(uNameRaw, 10, 64)
			if err == nil {
				var code int
				user, code = model.UserGetById(h.App.Db, tx, unUint64)
				if code != 1 {
					c.Status(http.StatusNotFound)
					return nil
				}
			} else {
				c.Status(http.StatusNotFound)
				return nil
			}
		}
		c.Redirect(302, "/member/"+strconv.FormatUint(user.ID, 10))
		return nil
	})
}

func (h *BaseHandler) MemberPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)

	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			c.Redirect(302, "/login")
			return
		}
		c.Redirect(302, "/setting")
		return
	}

	uid := strings.TrimSpace(c.Param("uid"))
	uidInt, err := strconv.ParseUint(uid, 10, 64)
	if err != nil {
		// 不是数字，取用户名
		c.Status(http.StatusNotFound)
		c.String(200, uid+" uid not found")
		return
	}

	db := h.App.Db
	evn := &ybs.MemberPage{}
	_ = db.View(func(tx *bbolt.Tx) error {

		user, code := model.UserGetById(db, tx, uidInt)
		if code != 1 {
			c.Status(http.StatusNotFound)
			c.String(200, uid+" user not found")
			return nil
		}

		btn, key, score := c.Query("btn"), c.Query("key"), c.Query("score")
		if len(key) > 0 {
			_, err := strconv.ParseUint(key, 10, 64)
			if err != nil {
				c.Redirect(302, "/")
				return nil
			}
		}
		if len(score) > 0 {
			_, err := strconv.ParseUint(score, 10, 64)
			if err != nil {
				c.Redirect(302, "/")
				return nil
			}
		}

		cmd := "zrscan"
		if btn == "prev" {
			cmd = "zscan"
		}

		var titleText string
		lstType := c.Query("type")
		if lstType == "comment" {
			titleText = "评论的主题"
		} else {
			titleText = "发表的主题"
			lstType = "topic"
		}

		scf := h.App.Cf.Site

		//topicPageInfo := model.GetTopicList(db, cmd, model.TbnPostUpdate, key, score, scf.PageShowNum)
		tbName := "user_" + lstType + ":" + strconv.FormatUint(user.ID, 10)
		topicPageInfo := model.GetTopicList(db, tx, cmd, tbName, key, score, scf.PageShowNum)

		evn.SiteCf = scf
		evn.Title = "会员: " + user.Name + " 最近" + titleText + " - " + scf.Name
		evn.CurrentUser = *curUser

		evn.NodeLst = model.NodeGetAll(h.App.Mc, db, tx)
		evn.TopicPageInfo = topicPageInfo
		evn.UserFmt = model.UserFmt{
			User:       user,
			RegTimeFmt: util.TimeFmt(int64(user.RegTime), "2006-01-02 15:04"),
		}
		evn.TopicNum = db.HGetInt(tx, model.TbnUserTopicNum, mdb.I2b(user.ID))
		evn.CommentNum = db.HGetInt(tx, model.TbnUserCommentNum, mdb.I2b(user.ID))
		if lstType == "comment" {
			evn.TopicPageInfo.TotalNum = evn.CommentNum
		} else {
			evn.TopicPageInfo.TotalNum = evn.TopicNum
		}
		evn.LstType = lstType
		evn.TitleText = titleText

		if curUser.ID == user.ID {
			if lstType == "comment" {
				evn.CommentReviewLst = model.CommentGetReview(db, tx, curUser.ID)
			} else {
				evn.TopicLst = model.TopicGetV2Review(db, tx, curUser.ID)
			}
		}

		evn.TagCloud = model.GetTagsForSide(h.App.Mc, db, tx, showTagNum)
		evn.RangeTopicLst = rangeTopicLst[:]
		evn.RecentComment = model.CommentGetRecent(h.App.Mc, db, tx, scf.RecentCommentNum)

		if curUser.ID > 0 {
			evn.HasMsg = model.MsgCheckHasOne(db, tx, curUser.ID)
			if curUser.Flag >= model.FlagAdmin {
				evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
				evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)
			}
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Status(http.StatusOK)
		ybs.WritePageTemplate(c.Writer, evn)

		return nil
	})

}
