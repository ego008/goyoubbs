// Code generated by qtc from "page_admin_topic_add.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

// admin_login
//

//line model/page_admin_topic_add.qtpl:3
package model

//line model/page_admin_topic_add.qtpl:3
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line model/page_admin_topic_add.qtpl:3
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line model/page_admin_topic_add.qtpl:4
type AdminTopicAdd struct {
	BasePage
	DefaultTopic Topic  // 编辑/添加
	DefaultUser  User   // 默认作者
	UserLst      []User // 可选发表用户列表，管理员
	GoBack       bool   // 返回到编辑前页面
}

//line model/page_admin_topic_add.qtpl:13
func (p *AdminTopicAdd) StreamMainBody(qw422016 *qt422016.Writer) {
//line model/page_admin_topic_add.qtpl:13
	qw422016.N().S(`

<div class="index">
    `)
//line model/page_admin_topic_add.qtpl:16
	if p.PageName == "admin_topic_review" && len(p.DefaultTopic.Title) == 0 {
//line model/page_admin_topic_add.qtpl:16
		qw422016.N().S(`
    <h1>没有 `)
//line model/page_admin_topic_add.qtpl:17
		qw422016.E().S(p.Title)
//line model/page_admin_topic_add.qtpl:17
		qw422016.N().S(`</h1>
    `)
//line model/page_admin_topic_add.qtpl:18
	} else {
//line model/page_admin_topic_add.qtpl:18
		qw422016.N().S(`
    <h1>
    `)
//line model/page_admin_topic_add.qtpl:20
		qw422016.E().S(p.Title)
//line model/page_admin_topic_add.qtpl:20
		qw422016.N().S(`
    `)
//line model/page_admin_topic_add.qtpl:21
		if len(p.DefaultTopic.Title) > 0 && p.PageName == "admin_topic_edit" {
//line model/page_admin_topic_add.qtpl:21
			qw422016.N().S(`
    <a href="/admin/topic/edit?id=`)
//line model/page_admin_topic_add.qtpl:22
			qw422016.N().DUL(p.DefaultTopic.ID)
//line model/page_admin_topic_add.qtpl:22
			qw422016.N().S(`&del=1">删除</a>
    `)
//line model/page_admin_topic_add.qtpl:23
		}
//line model/page_admin_topic_add.qtpl:23
		qw422016.N().S(`
    </h1>
    <p>`)
//line model/page_admin_topic_add.qtpl:25
		qw422016.E().S(p.DefaultTopic.Tags)
//line model/page_admin_topic_add.qtpl:25
		qw422016.N().S(`</p>
    <form class="pure-form" action="" method="post" onsubmit="form_post();return false;">
        <fieldset class="pure-group">
            <select id="select-nid">
                `)
//line model/page_admin_topic_add.qtpl:29
		for _, item := range p.NodeLst {
//line model/page_admin_topic_add.qtpl:29
			qw422016.N().S(`
                <option value="`)
//line model/page_admin_topic_add.qtpl:30
			qw422016.N().DUL(item.ID)
//line model/page_admin_topic_add.qtpl:30
			qw422016.N().S(`" `)
//line model/page_admin_topic_add.qtpl:30
			if item.ID == p.DefaultTopic.NodeId {
//line model/page_admin_topic_add.qtpl:30
				qw422016.N().S(`selected="selected"`)
//line model/page_admin_topic_add.qtpl:30
			}
//line model/page_admin_topic_add.qtpl:30
			qw422016.N().S(`>`)
//line model/page_admin_topic_add.qtpl:30
			qw422016.E().S(item.Name)
//line model/page_admin_topic_add.qtpl:30
			qw422016.N().S(`</option>
                `)
//line model/page_admin_topic_add.qtpl:31
		}
//line model/page_admin_topic_add.qtpl:31
		qw422016.N().S(`
            </select>
            <input id="id-title" type="text" value="`)
//line model/page_admin_topic_add.qtpl:33
		qw422016.E().S(p.DefaultTopic.Title)
//line model/page_admin_topic_add.qtpl:33
		qw422016.N().S(`" class="pure-input-1" placeholder="* 标题 MaxLen `)
//line model/page_admin_topic_add.qtpl:33
		qw422016.N().D(p.SiteCf.TitleMaxLen)
//line model/page_admin_topic_add.qtpl:33
		qw422016.N().S(`" />
            <textarea id="id-content" class="pure-input-1 topic-con-input" placeholder="* 内容 MaxLen `)
//line model/page_admin_topic_add.qtpl:34
		qw422016.N().D(p.SiteCf.TopicConMaxLen)
//line model/page_admin_topic_add.qtpl:34
		qw422016.N().S(`">`)
//line model/page_admin_topic_add.qtpl:34
		qw422016.N().S(p.DefaultTopic.Content)
//line model/page_admin_topic_add.qtpl:34
		qw422016.N().S(`</textarea>
            `)
//line model/page_admin_topic_add.qtpl:35
		if p.CurrentUser.Flag >= 99 {
//line model/page_admin_topic_add.qtpl:35
			qw422016.N().S(`
            <select id="select-uid">
                `)
//line model/page_admin_topic_add.qtpl:37
			for _, item := range p.UserLst {
//line model/page_admin_topic_add.qtpl:37
				qw422016.N().S(`
                <option value="`)
//line model/page_admin_topic_add.qtpl:38
				qw422016.N().DUL(item.ID)
//line model/page_admin_topic_add.qtpl:38
				qw422016.N().S(`" `)
//line model/page_admin_topic_add.qtpl:38
				if item.ID == p.DefaultUser.ID {
//line model/page_admin_topic_add.qtpl:38
					qw422016.N().S(`selected="selected"`)
//line model/page_admin_topic_add.qtpl:38
				}
//line model/page_admin_topic_add.qtpl:38
				qw422016.N().S(`>`)
//line model/page_admin_topic_add.qtpl:38
				qw422016.E().S(item.Name)
//line model/page_admin_topic_add.qtpl:38
				qw422016.N().S(`</option>
                `)
//line model/page_admin_topic_add.qtpl:39
			}
//line model/page_admin_topic_add.qtpl:39
			qw422016.N().S(`
            </select>
            <input id="id-addtime" type="text" value="`)
//line model/page_admin_topic_add.qtpl:41
			qw422016.N().DL(p.DefaultTopic.AddTime)
//line model/page_admin_topic_add.qtpl:41
			qw422016.N().S(`" class="pure-input-1" placeholder="发表的时间戳" />
            `)
//line model/page_admin_topic_add.qtpl:42
		} else {
//line model/page_admin_topic_add.qtpl:42
			qw422016.N().S(`
            <input type="hidden" id="select-uid" value="`)
//line model/page_admin_topic_add.qtpl:43
			qw422016.N().DUL(p.DefaultTopic.UserId)
//line model/page_admin_topic_add.qtpl:43
			qw422016.N().S(`">
            <input type="hidden" id="id-addtime" value="`)
//line model/page_admin_topic_add.qtpl:44
			qw422016.N().DL(p.DefaultTopic.AddTime)
//line model/page_admin_topic_add.qtpl:44
			qw422016.N().S(`">
            `)
//line model/page_admin_topic_add.qtpl:45
		}
//line model/page_admin_topic_add.qtpl:45
		qw422016.N().S(`
        </fieldset>
        <div id="id-msg"></div>
        <div class="fleft pure-button-group">
            <input id="btn-preview" type="button" value="预览" name="submit" class="pure-button button-success" />
            <input id="btn-submit" type="submit" value="发表" name="submit" class="pure-button pure-button-primary" />
            `)
//line model/page_admin_topic_add.qtpl:51
		if p.PageName == "admin_topic_review" {
//line model/page_admin_topic_add.qtpl:51
			qw422016.N().S(`
            <a href="?act=del" class="pure-button button-warning fr">直接删除</a>
            `)
//line model/page_admin_topic_add.qtpl:53
		}
//line model/page_admin_topic_add.qtpl:53
		qw422016.N().S(`
        </div>
        <div class="c"></div>

        <div id="id-preview" class="topic-content markdown-body"></div>
    </form>

    <script>
        let nodeEle = document.getElementById("select-nid");
        let titleEle = document.getElementById("id-title");
        let conEle = document.getElementById("id-content");
        let btnReviewEle = document.getElementById("btn-preview");
        let submitEle = document.getElementById("btn-submit");
        let msgEle = document.getElementById("id-msg");
        let addTimeEle = document.getElementById("id-addtime");
        let userIdEle = document.getElementById("select-uid");
        let reviewEle = document.getElementById("id-preview");

        btnReviewEle.addEventListener("click", function(){
            let con = conEle.value.trim();
            let title = titleEle.value.trim();
            if (con === "") {
                conEle.focus();
                return
            }

            btnReviewEle.setAttribute("disabled", "disabled");

            postAjax("/content/preview", JSON.stringify({Act: "topicPreview", Title: title, Content: con}), function(data){
                var obj = JSON.parse(data)
                //console.log(obj);
                if(obj.Code === 200) {
                    msgEle.style.display = "none";
                    reviewEle.innerHTML = obj.Html;
                    reviewEle.style.display = "block";
                }else{
                    reviewEle.innerHTML = "";
                    reviewEle.style.display = "none";
                    msgEle.innerText = obj.Msg;
                }
                btnReviewEle.removeAttribute('disabled');
            });
        }, false);

        function form_post(){
            let title = titleEle.value.trim();
            let con = conEle.value.trim();

            if (title === "") {
                titleEle.focus();
                return false;
            }

            if (con === "") {
                conEle.focus();
                return false;
            }

            reviewEle.innerHTML = "";
            reviewEle.style.display = "none";



            submitEle.setAttribute("disabled", "disabled");
            postAjax("/admin/topic/add", JSON.stringify({"Act": "submit", "ID": `)
//line model/page_admin_topic_add.qtpl:117
		qw422016.N().DUL(p.DefaultTopic.ID)
//line model/page_admin_topic_add.qtpl:117
		qw422016.N().S(`, "NodeId": parseInt(nodeEle.value, 10), "Title": title, "Content": con, "UserId": parseInt(userIdEle.value, 10), "AddTime": parseInt(addTimeEle.value.trim(), 10)}), function(data){
                var obj = JSON.parse(data)
                //console.log(obj);
                if(obj.Code === 200) {
                    msgEle.style.display = "none";
                    `)
//line model/page_admin_topic_add.qtpl:122
		if p.GoBack {
//line model/page_admin_topic_add.qtpl:122
			qw422016.N().S(`
                    window.location.href = "/t/`)
//line model/page_admin_topic_add.qtpl:123
			qw422016.N().DUL(p.DefaultTopic.ID)
//line model/page_admin_topic_add.qtpl:123
			qw422016.N().S(`";
                    return
                    `)
//line model/page_admin_topic_add.qtpl:125
		}
//line model/page_admin_topic_add.qtpl:125
		qw422016.N().S(`
                    `)
//line model/page_admin_topic_add.qtpl:126
		if p.PageName == "admin_topic_review" {
//line model/page_admin_topic_add.qtpl:126
			qw422016.N().S(`
                    window.location.href = "/admin/topic/review";
                    `)
//line model/page_admin_topic_add.qtpl:128
		} else {
//line model/page_admin_topic_add.qtpl:128
			qw422016.N().S(`
                    if(data.Tid > 0){
                        window.location.href = "/t/"+data.Tid;
                    }else{
                        window.location.href = "/admin/my/topic";
                    }
                    `)
//line model/page_admin_topic_add.qtpl:134
		}
//line model/page_admin_topic_add.qtpl:134
		qw422016.N().S(`
                    return false;
                } else if(obj.Code === 201){
                    msgEle.style.display = "block";
                    msgEle.innerText = obj.Msg;
                    titleEle.value = "";
                    conEle.value = "";

                    window.location.href = "/member/`)
//line model/page_admin_topic_add.qtpl:142
		qw422016.N().DUL(p.CurrentUser.ID)
//line model/page_admin_topic_add.qtpl:142
		qw422016.N().S(`";
                    return false;
                }else{
                    msgEle.style.display = "block";
                    msgEle.innerText = obj.Msg;
                }
                submitEle.removeAttribute('disabled');
            });

            return false;
        }

        document.addEventListener('paste', function (evt) {
            var url = "/file/upload";
            var items = evt.clipboardData && evt.clipboardData.items;
            var file = null;
            if(items && items.length) {
                for(var i=0; i!==items.length; i++) {
                    if(items[i].type.indexOf('image') !== -1) {
                        file = items[i].getAsFile();
                        if(!!!file) {
                            continue;
                        }

                        // upload file object.
                        var form = new FormData();
                        form.append('file', file);

                        postAjax("/file/upload", form, function(data){
                            let obj = JSON.parse(data)
                            //console.log(obj);
                            if(obj.Code === 200) {
                                let img_url = "\n" + obj.Url + "\n";
                                let pos = conEle.selectionStart;
                                let con = conEle.value;
                                conEle.value = con.slice(0, pos) + img_url + con.slice(pos);
                            }else{
                                console.warn(obj.Msg);
                            }
                        });
                    }
                }
            }

        });
    </script>

    `)
//line model/page_admin_topic_add.qtpl:189
	}
//line model/page_admin_topic_add.qtpl:189
	qw422016.N().S(`

</div>

`)
//line model/page_admin_topic_add.qtpl:193
}

//line model/page_admin_topic_add.qtpl:193
func (p *AdminTopicAdd) WriteMainBody(qq422016 qtio422016.Writer) {
//line model/page_admin_topic_add.qtpl:193
	qw422016 := qt422016.AcquireWriter(qq422016)
//line model/page_admin_topic_add.qtpl:193
	p.StreamMainBody(qw422016)
//line model/page_admin_topic_add.qtpl:193
	qt422016.ReleaseWriter(qw422016)
//line model/page_admin_topic_add.qtpl:193
}

//line model/page_admin_topic_add.qtpl:193
func (p *AdminTopicAdd) MainBody() string {
//line model/page_admin_topic_add.qtpl:193
	qb422016 := qt422016.AcquireByteBuffer()
//line model/page_admin_topic_add.qtpl:193
	p.WriteMainBody(qb422016)
//line model/page_admin_topic_add.qtpl:193
	qs422016 := string(qb422016.B)
//line model/page_admin_topic_add.qtpl:193
	qt422016.ReleaseByteBuffer(qb422016)
//line model/page_admin_topic_add.qtpl:193
	return qs422016
//line model/page_admin_topic_add.qtpl:193
}