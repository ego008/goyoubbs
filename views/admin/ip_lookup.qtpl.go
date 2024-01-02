// Code generated by qtc from "ip_lookup.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line views/admin/ip_lookup.qtpl:1
package admin

//line views/admin/ip_lookup.qtpl:1
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line views/admin/ip_lookup.qtpl:1
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line views/admin/ip_lookup.qtpl:1
func (p *IpLookup) StreamMainBody(qw422016 *qt422016.Writer) {
//line views/admin/ip_lookup.qtpl:1
	qw422016.N().S(`
<div class="index">
    <div class="markdown-body entry-content">
        <h1>`)
//line views/admin/ip_lookup.qtpl:4
	qw422016.E().S(p.Title)
//line views/admin/ip_lookup.qtpl:4
	qw422016.N().S(`</h1>

        <table>
          <tr>
            <th style="width: 40px">No.</th>
            <th>Ip</th>
            <th>DNS Host</th>
          </tr>
          `)
//line views/admin/ip_lookup.qtpl:12
	for i, item := range p.Items {
//line views/admin/ip_lookup.qtpl:12
		qw422016.N().S(`
          <tr>
            <td>`)
//line views/admin/ip_lookup.qtpl:14
		qw422016.N().D(i + 1)
//line views/admin/ip_lookup.qtpl:14
		qw422016.N().S(`</td>
            <td>`)
//line views/admin/ip_lookup.qtpl:15
		qw422016.N().S(item.Key)
//line views/admin/ip_lookup.qtpl:15
		qw422016.N().S(`</td>
            <td>`)
//line views/admin/ip_lookup.qtpl:16
		qw422016.E().S(item.Value)
//line views/admin/ip_lookup.qtpl:16
		qw422016.N().S(`</td>
          </tr>
          `)
//line views/admin/ip_lookup.qtpl:18
	}
//line views/admin/ip_lookup.qtpl:18
	qw422016.N().S(`
        </table>
        <ul class="paginate">
            `)
//line views/admin/ip_lookup.qtpl:21
	if p.ShowNext {
//line views/admin/ip_lookup.qtpl:21
		qw422016.N().S(`
            <li><a href="?key=`)
//line views/admin/ip_lookup.qtpl:22
		qw422016.E().S(p.KeyStart)
//line views/admin/ip_lookup.qtpl:22
		qw422016.N().S(`" class="next">Next Page </a></li>
            `)
//line views/admin/ip_lookup.qtpl:23
	}
//line views/admin/ip_lookup.qtpl:23
	qw422016.N().S(`
        </ul>

    </div>
</div>

`)
//line views/admin/ip_lookup.qtpl:29
}

//line views/admin/ip_lookup.qtpl:29
func (p *IpLookup) WriteMainBody(qq422016 qtio422016.Writer) {
//line views/admin/ip_lookup.qtpl:29
	qw422016 := qt422016.AcquireWriter(qq422016)
//line views/admin/ip_lookup.qtpl:29
	p.StreamMainBody(qw422016)
//line views/admin/ip_lookup.qtpl:29
	qt422016.ReleaseWriter(qw422016)
//line views/admin/ip_lookup.qtpl:29
}

//line views/admin/ip_lookup.qtpl:29
func (p *IpLookup) MainBody() string {
//line views/admin/ip_lookup.qtpl:29
	qb422016 := qt422016.AcquireByteBuffer()
//line views/admin/ip_lookup.qtpl:29
	p.WriteMainBody(qb422016)
//line views/admin/ip_lookup.qtpl:29
	qs422016 := string(qb422016.B)
//line views/admin/ip_lookup.qtpl:29
	qt422016.ReleaseByteBuffer(qb422016)
//line views/admin/ip_lookup.qtpl:29
	return qs422016
//line views/admin/ip_lookup.qtpl:29
}