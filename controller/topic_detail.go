package controller

import (
	"fmt"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"html"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"github.com/mssola/user_agent"
	"github.com/rs/xid"
	"go.etcd.io/bbolt"
)

var rangeLock = sync.Mutex{}

func (h *BaseHandler) TopicDetailPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)

	if h.App.Cf.Site.Authorized && curUser.Flag < model.FlagAuthor {
		if curUser.ID == 0 {
			c.Redirect(302, "/login")
			return
		}
		c.Redirect(302, "/setting")
		return
	}

	tid := c.Param("tid")
	tidInt, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		c.String(200, tid+" tid not found")
		//c.Redirect(302, "/")
		return
	}

	scf := h.App.Cf.Site
	db := h.App.Db

	var topic model.Topic
	var author model.User
	var node model.Node
	logoUrl := scf.MainDomain + "/static/logo_112.png"
	evn := &ybs.TopicDetailPage{}
	var safeTitle string
	var imgLst []string

	_ = db.Update(func(tx *bbolt.Tx) error {
		topic = model.TopicGetById(db, tx, tidInt)
		if topic.ID == 0 {
			// 不存在
			c.Status(http.StatusNotFound)
			c.Redirect(302, "/")
			return nil
		}
		tidByte := mdb.I2b(topic.ID)

		node, _ = model.NodeGetById(db, tx, topic.NodeId)
		// 作者
		author, _ = model.UserGetById(db, tx, topic.UserId)

		safeTitle = html.EscapeString(topic.Title)

		evn.CurrentUser = *curUser
		evn.SiteCf = scf
		evn.Title = safeTitle + " - " + scf.Name
		evn.Keywords = topic.Tags
		evn.Description = util.GetDesc(topic.Content)
		evn.Canonical = scf.MainDomain + "/t/" + tid //r.URL.Path
		evn.CloseSidebar = true
		if !scf.IsDevMod {
			evn.ShowAutoAd = true
		}

		evn.ReadMoreBreak = model.ReadMoreBreak
		evn.NodeLst = model.NodeGetAll(h.App.Mc, db, tx)

		var contentFmt string
		// get from mc
		var canRead bool
		var isPublic bool
		if topic.ReadAuthed || topic.ReadReply {
			if curUser.ID > 0 && curUser.Flag > model.FlagReview {
				if topic.ReadAuthed || curUser.Flag == model.FlagAdmin || curUser.ID == author.ID {
					canRead = true
				} else if topic.ReadReply {
					if db.HKeyExist(tx, model.TbnPostReply+strconv.FormatUint(topic.ID, 10), mdb.I2b(curUser.ID)) {
						canRead = true
					}
				}
			}
			if canRead {
				// show all
				contentFmt = util.ContentFmt(topic.Content)
				contentFmt = strings.ReplaceAll(contentFmt, `" alt="">`, `" alt="`+safeTitle+`">`)
				// find img
				imgLst = util.FindAllImgInContent(topic.Content)
			} else {
				//
				publicCon, privateCon := util.GetPublicCon(topic.Content)
				contentFmt = util.ContentFmt(publicCon)
				contentFmt = strings.ReplaceAll(contentFmt, `" alt="">`, `" alt="`+safeTitle+`">`)
				// read more tip
				if len(privateCon) > 0 {
					imgLen := util.CountAllImgInContent(privateCon)
					actName := "登录"
					if curUser.ID > 0 {
						if curUser.Flag == model.FlagForbidden {
							actName = "解禁"
						} else if curUser.Flag == model.FlagReview {
							actName = "等待审核"
						} else {
							if topic.ReadReply {
								actName = "回复"
							}
						}
					}
					contentFmt += `<p>为了防止爬虫，本主题 "` + topic.Title + `" 的发布者已设置浏览权限，需要“` + actName + `”才能继续浏览，剩余的内容包含个` + strconv.Itoa(len(privateCon)) + `字`
					if imgLen > 0 {
						contentFmt += `，其中包含` + strconv.Itoa(imgLen) + `张图片`
					}
					contentFmt += `</p>`
				}
				// find img
				imgLst = util.FindAllImgInContent(publicCon)
			}
		} else {
			isPublic = true
			mcKey := []byte("ContentFmt:" + tid)
			if mcValue, exist := util.ObjCachedGet(h.App.Mc, mcKey, nil, true); exist {
				contentFmt = mdb.B2s(mcValue)
			} else {
				contentFmt = util.ContentFmt(topic.Content)
				//
				contentFmt = strings.ReplaceAll(contentFmt, `" alt="">`, `" alt="`+safeTitle+`">`)
				util.ObjCachedSet(h.App.Mc, mcKey, contentFmt)
			}
		}

		// img list
		if len(imgLst) == 0 {
			//imgLst = append(imgLst, logoUrl) // for Json-LD
			imgLst = append(imgLst, scf.MainDomain+"/avatar/"+strconv.FormatUint(author.ID, 10)+".jpg")
		} else {
			for i, v := range imgLst {
				if strings.HasPrefix(v, "/static/") {
					imgLst[i] = scf.MainDomain + v
				}
			}
		}

		// 帖子评论数
		topic.Comments = db.HGetInt(tx, model.CommentNumTbName, tidByte)
		if topic.Comments > 0 {
			evn.CommentLst = model.GetAllTopicComment(h.App.Mc, db, tx, topic, isPublic, canRead)
		}

		evn.TopicFmt = model.TopicFmt{
			Topic:       topic,
			Name:        author.Name,
			AddTimeFmt:  util.TimeFmt(topic.AddTime, "2006-01-02 15:04"), // Jan 2TH, 2006
			EditTimeFmt: util.TimeFmt(topic.EditTime, "2006-01-02 15:04:05"),
			ContentFmt:  contentFmt,
			ClockEmoji:  util.GetTimeUnicodeClock(topic.AddTime),
			Relative:    model.TopicGetRelative(h.App.Mc, db, tx, topic.ID, topic.Tags),
		}
		evn.TopicFmt.Views, _ = db.HIncr(tx, "topic_view", tidByte, 1)
		evn.DefaultNode = node
		evn.OldTopic, evn.NewTopic = model.ArticleGetNearby(db, tx, topic.ID)
		if len(topic.Tags) > 0 {
			for _, v := range util.StringSplit(topic.Tags, ",") {
				evn.TagLst = append(evn.TagLst, model.TagFontSize{
					Name: v,
					Size: 0,
				})
			}
		}
		evn.TagCloud = model.GetTagsForSide(h.App.Mc, db, tx, showTagNum)
		evn.RangeTopicLst = rangeTopicLst[:]
		evn.RecentComment = model.CommentGetRecent(h.App.Mc, db, tx, scf.RecentCommentNum)

		// 消除站内未读
		if curUser.ID > 0 {
			tb := "user_msg:" + strconv.FormatUint(curUser.ID, 10)
			if db.HKeyExist(tx, tb, tidByte) {
				_ = db.HDel(tx, tb, tidByte)
			}
			evn.HasMsg = model.MsgCheckHasOne(db, tx, curUser.ID)
			if curUser.Flag >= model.FlagAdmin {
				evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
				evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)
			}
		}
		return nil
	})

	// Json-LD
	jsArticle := model.JsArticle{
		Type:             "Article",
		DateModified:     util.TimeFmt(topic.EditTime, "2006-01-02T15:04:05+08:00"),
		DatePublished:    util.TimeFmt(topic.AddTime, "2006-01-02T15:04:05+08:00"),
		Headline:         safeTitle,
		Description:      evn.Description,
		MainEntityOfPage: evn.Canonical,
		Image:            imgLst,
		Author: model.JsAuthor{
			Type: "Person",
			Name: author.Name,
			URL:  scf.MainDomain + "/member/" + strconv.FormatUint(author.ID, 10),
		},
		Publisher: model.JsPublisher{
			Type: "Organization",
			Name: scf.Name,
			Logo: model.JsLogo{
				Type: "ImageObject",
				URL:  scf.MainDomain + "/static/logo_112.png",
			},
		},
		Speakable: model.JsSpeakable{
			Type: "SpeakableSpecification",
			Xpath: []string{
				"/html/head/title",
				"/html/head/meta[@name='description']/@content",
			},
		},
		//CommentCount: commentCount,
		//Comment:      jsCommentLst,
	}
	commentCount := len(evn.CommentLst)
	var jsCommentLst []model.JsComment
	if commentCount > 0 {
		for _, v := range evn.CommentLst {
			commentAuthor := model.JsCommentAuthor{
				Type: "Person",
				Name: v.Name,
				Url:  scf.MainDomain + "/member/" + strconv.FormatUint(v.UserId, 10),
			}
			obj := model.JsComment{
				Type:        "Comment",
				Url:         scf.MainDomain + v.Link,
				Text:        util.GetDesc(v.Content),
				DateCreated: util.TimeFmt(v.AddTime, "2006-01-02T15:04:05+08:00"),
				Name:        strconv.FormatUint(v.ID, 10),
				Author:      commentAuthor,
				Publisher:   commentAuthor,
			}
			jsCommentLst = append(jsCommentLst, obj)
		}

		jsArticle.CommentCount = commentCount
		jsArticle.Comment = jsCommentLst
	}
	jsonLd := model.JsonLd{
		Context: "http://schema.org/",
		Graph: []interface{}{
			model.JsOrganization{
				Type: "Organization",
				Logo: logoUrl,
				URL:  scf.MainDomain,
			},
			model.JsBreadcrumbList{
				Type: "BreadcrumbList",
				ItemListElement: []model.JsItemListElement{
					{
						Type:     "ListItem",
						Position: 1,
						Name:     scf.Name,
						Item:     scf.MainDomain,
					},
					{
						Type:     "ListItem",
						Position: 2,
						Name:     node.Name,
						Item:     scf.MainDomain + "/n/" + strconv.FormatUint(node.ID, 10),
					},
				},
			},
			jsArticle,
		},
	}
	jb, _ := json.Marshal(jsonLd)
	evn.JsonLd = mdb.B2s(jb)
	// Json-LD ed

	ua := c.GetHeader("User-Agent")
	if len(ua) > 30 {
		uaCli := user_agent.New(ua)
		if !uaCli.Bot() {
			var has bool
			for _, v := range evn.RangeTopicLst {
				if v.ID == topic.ID {
					has = true
					break
				}
			}
			if !has {
				rangeLock.Lock()
				rangeTopicLst = append(rangeTopicLst, model.TopicLi{ID: topic.ID, Title: topic.Title})
				if len(rangeTopicLst) > scf.TopRateNum {
					rangeTopicLst = rangeTopicLst[1:]
				}
				rangeLock.Unlock()
			}
		}
	}

	token := h.GetCookie(c, "token")
	if len(token) == 0 {
		token := xid.New().String()
		_ = h.SetCookie(c, "token", token, 1)
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	ybs.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) TopicDetailPost(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAuthor {
		c.String(200, `{"Code":401,"Msg":"请先登录 ->"}`)
		return
	}
	tid := c.Param("tid")
	tidInt, err := strconv.ParseUint(tid, 10, 64)
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"错误 tid"}`)
		return
	}

	scf := h.App.Cf.Site
	if scf.CloseReply && curUser.Flag < model.FlagAdmin {
		c.String(200, `{"Code":403,"Msg":"评论已关闭"}`)
		return
	}

	db := h.App.Db

	_ = db.Update(func(tx *bbolt.Tx) error {

		topic := model.TopicGetById(db, tx, tidInt)
		if topic.ID == 0 {
			// 不存在
			c.String(200, `{"Code":400,"Msg":"帖子不存在"}`)
			return nil
		}

		var rec model.Comment
		err = util.Bind(c, util.JSON, &rec)
		if err != nil {
			fmt.Println(err)
			c.String(200, `{"Code":400,"Msg":"unable to read body"}`)
			return nil
		}

		rec.Content = strings.TrimSpace(rec.Content)

		contentLen := len(rec.Content)
		if contentLen == 0 {
			c.String(200, `{"Code":400,"Msg":"评论内容不能为空"}`)
			return nil
		}

		if curUser.Flag < model.FlagAdmin && contentLen > scf.TopicConMaxLen {
			msg := fmt.Sprintf(`{"Code":400,"Msg":"文章内容太长 %d > %d "}`, contentLen, scf.TopicConMaxLen)
			c.String(200, msg)
			return nil
		}

		// get ip
		clip := c.GetHeader("X-Forwarded-For")
		if len(clip) == 0 {
			clip = c.GetHeader("X-FORWARDED-FOR")
		}

		stamp := util.GetCNTM(model.TimeOffSet)
		comment := model.Comment{
			// ID:      0,
			ReplyId:  rec.ReplyId,
			TopicId:  topic.ID,
			UserId:   curUser.ID,
			Content:  rec.Content,
			ClientIp: clip,
			AddTime:  stamp,
		}

		type response struct {
			model.NormalRsp
			Tid uint64
		}
		rsp := response{}
		rsp.Code = 200

		if curUser.Flag < model.FlagTrust && scf.PostReview {
			// 非管理员+开启审核
			// 检测限制，防止机器恶意灌水
			var showStr string
			_ = db.HGetFunc(tx, "userLastReplyTime", mdb.I2b(curUser.ID), func(val []byte) error {
				if (uint64(stamp) - mdb.B2i(val)) < 20 {
					showStr = `{"Code":403,"Msg":"稍休息一下，请勿灌水"}`
					return nil
				}
				return nil
			})
			if len(showStr) > 0 {
				c.String(200, showStr)
				return nil
			}

			if model.CommentGetReviewNum(db, tx, curUser.ID) >= 10 {
				c.String(200, `{"Code":403,"Msg":"请勿灌水"}`)
				return nil
			}
			// 把jb 内容暂存到审核列表
			jb, _ := json.Marshal(comment)
			k := []byte(strconv.FormatInt(comment.AddTime, 10) + "_" + tid)
			// 给管理员
			_ = db.HSet(tx, model.CommentReviewTbName, k, jb)
			// 给个人
			_ = db.HSet(tx, model.CommentReviewTbName+":"+strconv.FormatUint(comment.UserId, 10), k, nil)
			// 记录最后请求发表时间
			_ = db.HSet(tx, "userLastReplyTime", mdb.I2b(curUser.ID), mdb.I2b(uint64(stamp)))
			rsp.Code = 201
			rsp.Msg = "* 您的评论已经提交，请耐心等管理员审核"

			// 构建邮件信息，给管理员发邮件，尽快来验证
			if scf.SendEmail {
				mailInfo := model.EmailInfo{}
				mailInfo.Key = uint64(time.Now().UTC().UnixNano())
				mailInfo.Subject = scf.Name + " " + curUser.Name + "回复帖子《" + topic.Title + "》审核 " + strconv.FormatInt(comment.AddTime, 10)
				mailInfo.Body = "这是一封系统通知邮件：" + curUser.Name + "于" + util.TimeFmt(comment.AddTime, "") + " 回帖，IP" + comment.ClientIp + " 内容摘要：<br><br>" + util.GetDesc(comment.Content) + "<br><br>请尽快前往处理 " + scf.MainDomain + "/admin/comment/review"
				model.EmailInfoUpdate(db, tx, mailInfo)
			}

			_ = json.NewEncoder(c.Writer).Encode(rsp)
			return nil
		}

		// 直接保存
		comment = model.CommentAdd(h.App.Mc, db, tx, comment)
		rsp.Tid = comment.ID
		rsp.Msg = "评论提交成功"

		_ = json.NewEncoder(c.Writer).Encode(rsp)

		return nil
	})
}
