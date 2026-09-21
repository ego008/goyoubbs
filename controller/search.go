package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) SearchPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)

	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			c.Redirect(302, "/login")
			return
		}
		c.Redirect(302, "/setting")
		return
	}

	//if curUser.ID == 0 {
	//	c.Redirect(302,h.App.Cf.Site.MainDomain+"/login")
	//	return
	//}

	q := strings.TrimSpace(c.Query("q"))
	if len(q) == 0 {
		c.Redirect(302, "/")
		return
	}

	scf := h.App.Cf.Site

	qLow := strings.ToLower(q)

	where := "title"
	if strings.HasPrefix(qLow, "c:") {
		where = "content"
		qLow = strings.TrimSpace(qLow[2:])
		if len(qLow) == 0 {
			c.Redirect(302, scf.MainDomain+"/")
			return
		}
	}

	db := h.App.Db

	var pageInfo model.TopicPageInfo
	mcKey := []byte("search:" + where + ":" + qLow)

	evn := &ybs.SearchPage{}
	_ = db.View(func(tx *bbolt.Tx) error {
		if _, exist := util.ObjCachedGet(h.App.Mc, mcKey, &pageInfo, false); !exist {
			pageInfo = model.SearchTopicList(h.App.Mc, db, tx, q, scf.PageShowNum)
			// set to mc
			util.ObjCachedSet(h.App.Mc, mcKey, pageInfo)
		}

		evn.SiteCf = scf
		evn.Title = "搜索: " + q + " - " + scf.Name
		evn.CurrentUser = *curUser

		evn.Q = q
		evn.NodeLst = model.NodeGetAll(h.App.Mc, db, tx)
		evn.TopicPageInfo = pageInfo
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
