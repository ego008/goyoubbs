package controller

import (
	"github.com/ego008/sdb"
	"github.com/segmentio/fasthash/fnv1a"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"strconv"
	"strings"
	"time"
)

func s2uint64(s string) uint64 {
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func (h *BaseHandler) ApiAdminRemotePost(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")                                                            // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
	ctx.Response.Header.Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型
	ctx.Response.Header.Add("Access-Control-Allow-Credentials", "true")                                                    //设置为true，允许ajax异步请求带cookie信息
	ctx.Response.Header.Add("Access-Control-Allow-Methods", "POST")                                                        //允许请求方法
	ctx.SetContentType("application/json; charset=UTF-8")

	if string(ctx.Method()) != fasthttp.MethodPost {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		return
	}

	if h.App.Cf.Site.RemotePostPw == "" {
		_, _ = ctx.WriteString(`{"Code":403,"Msg":"Site.RemotePostPw is empty"}`)
		return
	}

	topicId := s2uint64(strings.TrimSpace(string(ctx.FormValue("TopicId"))))
	nodeId := s2uint64(strings.TrimSpace(string(ctx.FormValue("NodeId"))))
	userName := strings.TrimSpace(string(ctx.FormValue("UserName")))
	title := strings.TrimSpace(string(ctx.FormValue("Title")))
	content := strings.TrimSpace(string(ctx.FormValue("Content")))
	remotePostPw := strings.TrimSpace(string(ctx.FormValue("RemotePostPw")))

	// limit
	clientIp := []byte(ReadUserIP(ctx))
	if len(clientIp) == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"clientIp is empty"}`)
		return
	}

	db := h.App.Db
	var hasDelKey bool
	offSetSeconds := int64(120)
	nowTm := time.Now().Unix()

	tbn := "remote_post_client_pw_err"
	if tm := db.HgetInt(tbn, clientIp); tm > 0 {
		if nowTm-int64(tm) < offSetSeconds {
			_, _ = ctx.WriteString(`{"Code":403,"Msg":"sleep 2 min ` + string(clientIp) + `"}`)
			return
		}
		hasDelKey = true
	}

	// check
	if remotePostPw != h.App.Cf.Site.RemotePostPw {
		_ = db.Hset(tbn, clientIp, sdb.I2b(uint64(nowTm)))
		_, _ = ctx.WriteString(`{"Code":403,"Msg":"remotePostPw not match"}`)
		return
	}
	if hasDelKey {
		_ = db.Hdel(tbn, clientIp)
	}

	if topicId == 0 && nodeId == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"topicId or nodeId must be an int type"}`)
		return
	}

	if content == "" {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"content is empty"}`)
		return
	}

	userName = strings.TrimSpace(util.RemoveCharacter(userName))
	if len(userName) == 0 {
		_, _ = ctx.WriteString(`{"Code":400,"Msg":"userName is empty"}`)
		return
	}

	stamp := util.GetCNTM(model.TimeOffSet)

	nameLow := strings.ToLower(userName)
	user, _ := model.UserGetByName(db, nameLow)
	if user.ID > 0 {
		if user.Flag == model.FlagForbidden {
			_, _ = ctx.WriteString(`{"Code":403,"Msg":"user Flag is 0"}`)
			return
		}
	} else {
		// add user
		userId, _ := db.Hincr(model.CountTb, sdb.S2b(model.UserTbName), 1)
		user = model.User{
			ID:       userId,
			Name:     userName,
			Flag:     model.FlagAuthor,
			Password: util.Md5(util.RandStringBytesMaskImprSrcSB(8)),
			RegTime:  uint64(stamp),
		}

		user = model.UserSet(db, user)
		if user.ID == 0 {
			_, _ = ctx.WriteString(`{"Code":500,"Msg":"user set err"}`)
			return
		}
		_ = db.Hset("user_name2uid", []byte(nameLow), sdb.I2b(user.ID))
		_ = db.Hset("user_flag:"+strconv.Itoa(user.Flag), sdb.I2b(user.ID), nil)
		//生成头像
		_ = util.GenAvatar(db, user.ID, user.Name)
	}

	var topic model.Topic
	var comment model.Comment

	if nodeId > 0 {
		// 优先发帖
		// add topic
		if title == "" {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"title is empty"}`)
			return
		}
		if _, c := model.NodeGetById(db, nodeId); c != 1 {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"nodeId not exist"}`)
			return
		}
		// check title
		titleMd5 := fnv1a.HashString64(title)
		if rs := db.Hget("title_fnv1a", sdb.I2b(titleMd5)); rs.OK() {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"相同的文章标题已存在，请修改"}`)
			return
		}

		topic = model.Topic{
			UserId:   user.ID,
			NodeId:   nodeId,
			Title:    title,
			Content:  content,
			ClientIp: sdb.B2s(clientIp),
			AddTime:  stamp,
			EditTime: stamp,
		}
		// 直接保存
		topic = model.TopicAdd(h.App.Mc, db, topic)

		// 自动从标题里提取标签
		if len(h.App.Cf.Site.GetTagApi) > 0 {
			_ = db.Hset("task_to_get_tag", sdb.I2b(topic.ID), sdb.S2b(topic.Title))
		}

		// 记录标题md5
		_ = db.Hset("title_fnv1a", sdb.I2b(titleMd5), sdb.I2b(topic.ID))

		_, _ = ctx.WriteString(`{"Code":200,"Msg":"ok, new topicId ` + strconv.FormatUint(topic.ID, 10) + `"}`)
		return
	} else {
		// add comment
		if topicId <= 0 {
			_, _ = ctx.WriteString(`{"Code":400,"Msg":"topicId must > 0"}`)
			return
		}
		comment = model.Comment{
			UserId:   user.ID,
			TopicId:  topicId,
			AddTime:  stamp,
			Content:  content,
			ClientIp: sdb.B2s(clientIp),
		}
		comment = model.CommentAdd(h.App.Mc, db, comment)
		_, _ = ctx.WriteString(`{"Code":200,"Msg":"ok, new commentId ` + strconv.FormatUint(comment.ID, 10) + `"}`)
		return
	}
}
