package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"html/template"
	"log"
	"strconv"
	"strings"

	"github.com/ego008/goutils/json"
	"github.com/gin-gonic/gin"
)

func (h *BaseHandler) ContentPreview(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")
	curUser, _ := h.CurrentUser(c)

	if curUser.Flag < model.FlagAuthor {
		c.String(200, `{"Code":401,"Msg":"请先登录"}`)
		return
	}

	type recForm struct {
		Act     string
		Title   string
		Content string
	}

	type response struct {
		model.NormalRsp
		Html template.HTML
	}

	var rec recForm
	err := util.Bind(c, util.JSON, &rec)
	if err != nil {
		log.Println(err)
		c.String(200, `{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	if len(rec.Content) == 0 {
		c.String(200, `{"Code":400,"Msg":"内容不能为空"}`)
		return
	}

	rec.Content = strings.ReplaceAll(rec.Content, "\r\n", "\n")

	rsp := response{}
	rsp.Code = 200

	var _html string
	conLen := len(rec.Content)
	if rec.Act == "topicPreview" {
		// 检测字数
		titleLen := len(rec.Title)
		if titleLen > h.App.Cf.Site.TitleMaxLen {
			c.String(200, `{"Code":201,"Msg":"文章标题太长 `+strconv.Itoa(titleLen)+` > `+strconv.Itoa(h.App.Cf.Site.TitleMaxLen)+`"}`)
			return
		}
		if conLen > h.App.Cf.Site.TopicConMaxLen {
			c.String(200, `{"Code":201,"Msg":"主题内容太长 `+strconv.Itoa(conLen)+` > `+strconv.Itoa(h.App.Cf.Site.TopicConMaxLen)+`"}`)
			return
		}
		// 主贴预览显示摘要
		_html = util.GetDesc(rec.Content) + "<hr>"
	} else if rec.Act == "commentPreview" {
		// 检测字数
		if conLen > h.App.Cf.Site.CommentConMaxLen {
			c.String(200, `{"Code":201,"Msg":"评论内容太长 `+strconv.Itoa(conLen)+` > `+strconv.Itoa(h.App.Cf.Site.CommentConMaxLen)+`"}`)
			return
		}
	}
	_html += util.ContentFmt(rec.Content)
	rsp.Html = template.HTML(_html)

	_ = json.NewEncoder(c.Writer).Encode(rsp)
}
