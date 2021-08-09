package controller

import (
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"html/template"
)

func (h *BaseHandler) FeedHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/atom+xml; charset=utf-8")

	scf := h.App.Cf.Site

	var feed = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
	<title>` + scf.Name + `</title>
	<link rel="self" type="application/atom+xml" href="` + scf.MainDomain + `/feed"/>
	<link rel="hub" href="https://pubsubhubbub.appspot.com"/>
	<updated>{{.Update}}</updated>
	<id>` + scf.MainDomain + `/feed</id>
	<author>
		<name>` + scf.Name + `</name>
	</author>
	{{range $_, $item := .Items}}
	<entry>
		<title>{{$item.Title}}</title>
		<id>` + scf.MainDomain + `/t/{{$item.ID}}</id>
		<link rel="alternate" type="text/html" href="` + scf.MainDomain + `/t/{{$item.ID}}" />
		<published>{{$item.AddTimeFmt}}</published>
		<updated>{{$item.EditTimeFmt}}</updated>
		<content type="text/plain">
		  {{$item.Des}}
		</content>
    </entry>
	{{end}}
</feed>
`

	db := h.App.Db

	items := model.TopicGetForFeed(db, 20)

	var upDate string
	if len(items) > 0 {
		upDate = items[0].AddTimeFmt
	}

	t := template.Must(template.New("feed").Parse(feed))
	_ = t.Execute(ctx, struct {
		Update string
		Items  []model.TopicFeed
	}{
		Update: upDate,
		Items:  items,
	})
}
