package controller

import (
	"goyoubbs/model"
	"goyoubbs/views/ybs"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) NodePage(c *gin.Context) {
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

	nidStr := c.Param("nid")
	nid, err := strconv.ParseUint(nidStr, 10, 64)
	if err != nil {
		c.Redirect(302, "/")
		return
	}
	node, code := model.NodeGetById(db, nid)
	if code != 1 {
		c.Redirect(302, "/")
		return
	}

	btn, key, score := c.Query("btn"), c.Query("key"), c.Query("score")
	if len(key) > 0 {
		_, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			c.Redirect(302, "/")
			return
		}
	}
	if len(score) > 0 {
		_, err := strconv.ParseUint(score, 10, 64)
		if err != nil {
			c.Redirect(302, "/")
			return
		}
	}

	cmd := "zrscan"
	if btn == "prev" {
		cmd = "zscan"
	}

	// sort id
	//topicPageInfo := model.GetTopicListArchives(db, cmd, "topic_node:"+nidStr, key, score, scf.PageShowNum)
	// sort comment add time
	topicPageInfo := model.GetTopicList(db, cmd, "topic_update:"+nidStr, key, score, scf.PageShowNum)

	//log.Println(topicPageInfo)

	evn := &ybs.NodePage{}
	evn.SiteCf = scf
	evn.Title = "Category: " + node.Name + " - " + scf.Name
	evn.CurrentUser = *curUser

	evn.DefaultNode = node
	evn.NodeLst = model.NodeGetAll(h.App.Mc, db)
	evn.TopicPageInfo = topicPageInfo
	evn.TagCloud = model.GetTagsForSide(h.App.Mc, db, showTagNum)
	evn.RangeTopicLst = rangeTopicLst[:]
	evn.RecentComment = model.CommentGetRecent(h.App.Mc, db, scf.RecentCommentNum)

	if curUser.ID > 0 {
		evn.HasMsg = model.MsgCheckHasOne(db, curUser.ID)
		if curUser.Flag >= model.FlagAdmin {
			evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db)
			evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db)
		}
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	ybs.WritePageTemplate(c.Writer, evn)
}
