package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/ybs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) TagPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)

	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			c.Redirect(302, "/login")
			return
		}
		c.Redirect(302, "/setting")
		return
	}

	db := h.App.Db
	scf := h.App.Cf.Site

	evn := &ybs.TagPage{}

	tagRaw := c.Param("tag")
	if tag, err := url.QueryUnescape(tagRaw); err == nil {
		tagRaw = tag
	}
	tagLower := strings.ToLower(tagRaw)

	_ = db.View(func(tx *bbolt.Tx) error {
		var ok bool
		_ = db.HScanFunc(tx, "tag:"+tagLower, nil, 1, func(key, val []byte) bool {
			ok = true
			return true
		})
		if !ok {
			// 该标签下文章数为0
			c.Redirect(302, "/")
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

		topicPageInfo := model.GetTopicListArchives(db, tx, cmd, "tag:"+tagLower, key, scf.PageShowNum)

		evn.SiteCf = scf
		evn.Title = "Tag: " + tagRaw + " - " + scf.Name
		evn.CurrentUser = *curUser

		evn.Tag = tagLower
		evn.NodeLst = model.NodeGetAll(h.App.Mc, db, tx)
		evn.TopicPageInfo = topicPageInfo
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

		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	ybs.WritePageTemplate(c.Writer, evn)
}
