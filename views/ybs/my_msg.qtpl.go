// Code generated by qtc from "my_msg.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line views/ybs/my_msg.qtpl:1
package ybs

//line views/ybs/my_msg.qtpl:1
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line views/ybs/my_msg.qtpl:1
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line views/ybs/my_msg.qtpl:1
func (p *MyMsg) StreamMainBody(qw422016 *qt422016.Writer) {
//line views/ybs/my_msg.qtpl:1
	qw422016.N().S(`
<div class="index">

    <h1>`)
//line views/ybs/my_msg.qtpl:4
	qw422016.E().S(p.Title)
//line views/ybs/my_msg.qtpl:4
	qw422016.N().S(`</h1>
    <p class="bot-line">有人在下面帖子回复里 @ 了你，请及时前往查看</p>

    `)
//line views/ybs/my_msg.qtpl:7
	for _, item := range p.TopicPageInfo.Items {
//line views/ybs/my_msg.qtpl:7
		qw422016.N().S(`
    <article>

        <header>
            <a href="/member/`)
//line views/ybs/my_msg.qtpl:11
		qw422016.N().DUL(item.UserId)
//line views/ybs/my_msg.qtpl:11
		qw422016.N().S(`" rel="nofollow"><img alt="`)
//line views/ybs/my_msg.qtpl:11
		qw422016.E().S(item.AuthorName)
//line views/ybs/my_msg.qtpl:11
		qw422016.N().S(` avatar" src="/static/avatar/`)
//line views/ybs/my_msg.qtpl:11
		qw422016.N().DUL(item.UserId)
//line views/ybs/my_msg.qtpl:11
		qw422016.N().S(`.jpg" class="avatar"></a>
            <h1><a href="/t/`)
//line views/ybs/my_msg.qtpl:12
		qw422016.N().DUL(item.ID)
//line views/ybs/my_msg.qtpl:12
		qw422016.N().S(`" rel="bookmark" title="Permanent Link to `)
//line views/ybs/my_msg.qtpl:12
		qw422016.E().S(item.Title)
//line views/ybs/my_msg.qtpl:12
		qw422016.N().S(`">`)
//line views/ybs/my_msg.qtpl:12
		qw422016.E().S(item.Title)
//line views/ybs/my_msg.qtpl:12
		qw422016.N().S(`</a></h1>
            <p class="meta">
                <a href="/n/`)
//line views/ybs/my_msg.qtpl:14
		qw422016.N().DUL(item.NodeId)
//line views/ybs/my_msg.qtpl:14
		qw422016.N().S(`">`)
//line views/ybs/my_msg.qtpl:14
		qw422016.E().S(item.NodeName)
//line views/ybs/my_msg.qtpl:14
		qw422016.N().S(`</a>
                <a href="/member/`)
//line views/ybs/my_msg.qtpl:15
		qw422016.N().DUL(item.UserId)
//line views/ybs/my_msg.qtpl:15
		qw422016.N().S(`" rel="nofollow">`)
//line views/ybs/my_msg.qtpl:15
		qw422016.E().S(item.AuthorName)
//line views/ybs/my_msg.qtpl:15
		qw422016.N().S(`</a>
                <time datetime="`)
//line views/ybs/my_msg.qtpl:16
		qw422016.E().S(item.AddTimeFmt)
//line views/ybs/my_msg.qtpl:16
		qw422016.N().S(`" pubdate data-updated="true">`)
//line views/ybs/my_msg.qtpl:16
		qw422016.E().S(item.EditTimeFmt)
//line views/ybs/my_msg.qtpl:16
		qw422016.N().S(`</time>
                `)
//line views/ybs/my_msg.qtpl:17
		if item.Comments > 0 {
//line views/ybs/my_msg.qtpl:17
			qw422016.N().S(`
                <a class="right count" href="/t/`)
//line views/ybs/my_msg.qtpl:18
			qw422016.N().DUL(item.ID)
//line views/ybs/my_msg.qtpl:18
			qw422016.N().S(`#r`)
//line views/ybs/my_msg.qtpl:18
			qw422016.N().DUL(item.Comments)
//line views/ybs/my_msg.qtpl:18
			qw422016.N().S(`" title="Comment on `)
//line views/ybs/my_msg.qtpl:18
			qw422016.E().S(item.Title)
//line views/ybs/my_msg.qtpl:18
			qw422016.N().S(`" rel="nofollow">`)
//line views/ybs/my_msg.qtpl:18
			qw422016.N().DUL(item.Comments)
//line views/ybs/my_msg.qtpl:18
			qw422016.N().S(`</a>
                `)
//line views/ybs/my_msg.qtpl:19
		}
//line views/ybs/my_msg.qtpl:19
		qw422016.N().S(`
            </p>
        </header>

    </article>

    `)
//line views/ybs/my_msg.qtpl:25
	}
//line views/ybs/my_msg.qtpl:25
	qw422016.N().S(`

    <div class="top-line">
    `)
//line views/ybs/my_msg.qtpl:28
	if len(p.TopicPageInfo.Items) == 10 {
//line views/ybs/my_msg.qtpl:28
		qw422016.N().S(`
    * 以上只显示最早 10 条信息
    `)
//line views/ybs/my_msg.qtpl:30
	} else {
//line views/ybs/my_msg.qtpl:30
		qw422016.N().S(`
    &nbsp;
    `)
//line views/ybs/my_msg.qtpl:32
	}
//line views/ybs/my_msg.qtpl:32
	qw422016.N().S(`
    </div>

</div>

`)
//line views/ybs/my_msg.qtpl:37
}

//line views/ybs/my_msg.qtpl:37
func (p *MyMsg) WriteMainBody(qq422016 qtio422016.Writer) {
//line views/ybs/my_msg.qtpl:37
	qw422016 := qt422016.AcquireWriter(qq422016)
//line views/ybs/my_msg.qtpl:37
	p.StreamMainBody(qw422016)
//line views/ybs/my_msg.qtpl:37
	qt422016.ReleaseWriter(qw422016)
//line views/ybs/my_msg.qtpl:37
}

//line views/ybs/my_msg.qtpl:37
func (p *MyMsg) MainBody() string {
//line views/ybs/my_msg.qtpl:37
	qb422016 := qt422016.AcquireByteBuffer()
//line views/ybs/my_msg.qtpl:37
	p.WriteMainBody(qb422016)
//line views/ybs/my_msg.qtpl:37
	qs422016 := string(qb422016.B)
//line views/ybs/my_msg.qtpl:37
	qt422016.ReleaseByteBuffer(qb422016)
//line views/ybs/my_msg.qtpl:37
	return qs422016
//line views/ybs/my_msg.qtpl:37
}