package controller

import (
	"github.com/ego008/goutils/json"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"html/template"
	"log"
	"strconv"
	"strings"
)

func (h *BaseHandler) ContentPreview(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json; charset=UTF-8")
	curUser, _ := h.CurrentUser(ctx)

	if curUser.Flag < model.FlagAuthor {
		_, _ = ctx.WriteString(`{"Code":401,"Msg":"请先登录"}`)
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
	err := util.Bind(ctx, util.JSON, &rec)
	if err != nil {
		log.Println(err)
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	if len(rec.Content) == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"内容不能为空"}`)
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
			_, _ = ctx.WriteString(`{"Code":201,"Msg":"文章标题太长 ` + strconv.Itoa(titleLen) + ` > ` + strconv.Itoa(h.App.Cf.Site.TitleMaxLen) + `"}`)
			return
		}
		if conLen > h.App.Cf.Site.TopicConMaxLen {
			_, _ = ctx.WriteString(`{"Code":201,"Msg":"主题内容太长 ` + strconv.Itoa(conLen) + ` > ` + strconv.Itoa(h.App.Cf.Site.TopicConMaxLen) + `"}`)
			return
		}
		// 主贴预览显示摘要
		_html = util.GetDesc(rec.Content) + "<hr>"
	} else if rec.Act == "commentPreview" {
		// 检测字数
		if conLen > h.App.Cf.Site.CommentConMaxLen {
			_, _ = ctx.WriteString(`{"Code":201,"Msg":"评论内容太长 ` + strconv.Itoa(conLen) + ` > ` + strconv.Itoa(h.App.Cf.Site.CommentConMaxLen) + `"}`)
			return
		}
	}
	_html += util.ContentFmt(rec.Content)
	rsp.Html = template.HTML(_html)

	_ = json.NewEncoder(ctx).Encode(rsp)
}
