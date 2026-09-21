package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/fasthash/fnv1a"
	"go.etcd.io/bbolt"
)

func s2uint64(s string) uint64 {
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func (h *BaseHandler) ApiAdminRemotePost(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")                                                            // 允许访问所有域，可以换成具体url，注意仅具体url才能带cookie信息
	c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型
	c.Header("Access-Control-Allow-Credentials", "true")                                                    //设置为true，允许ajax异步请求带cookie信息
	c.Header("Access-Control-Allow-Methods", "POST")                                                        //允许请求方法
	c.Header("Content-Type", "application/json; charset=UTF-8")

	if c.Request.Method != http.MethodPost {
		c.Status(http.StatusMethodNotAllowed)
		return
	}

	if h.App.Cf.Site.RemotePostPw == "" {
		c.String(200, `{"Code":403,"Msg":"Site.RemotePostPw is empty"}`)
		return
	}

	topicId := s2uint64(strings.TrimSpace(c.Query("TopicId")))
	nodeId := s2uint64(strings.TrimSpace(c.Query("NodeId")))
	userName := strings.TrimSpace(c.Query("UserName"))
	title := strings.TrimSpace(c.Query("Title"))
	content := strings.TrimSpace(c.Query("Content"))
	remotePostPw := strings.TrimSpace(c.Query("RemotePostPw"))

	// limit
	clientIp := []byte(ReadUserIP(c))
	if len(clientIp) == 0 {
		c.String(200, `{"Code":400,"Msg":"clientIp is empty"}`)
		return
	}

	db := h.App.Db
	var hasDelKey bool
	offSetSeconds := int64(120)
	nowTm := time.Now().Unix()

	_ = db.Update(func(tx *bbolt.Tx) error {

		tbn := "remote_post_client_pw_err"
		if tm := db.HGetInt(tx, tbn, clientIp); tm > 0 {
			if nowTm-int64(tm) < offSetSeconds {
				c.String(200, `{"Code":403,"Msg":"sleep 2 min `+string(clientIp)+`"}`)
				return nil
			}
			hasDelKey = true
		}

		// check
		if remotePostPw != h.App.Cf.Site.RemotePostPw {
			_ = db.HSet(tx, tbn, clientIp, mdb.I2b(uint64(nowTm)))
			c.String(200, `{"Code":403,"Msg":"remotePostPw not match"}`)
			return nil
		}
		if hasDelKey {
			_ = db.HDel(tx, tbn, clientIp)
		}

		if topicId == 0 && nodeId == 0 {
			c.String(200, `{"Code":400,"Msg":"topicId or nodeId must be an int type"}`)
			return nil
		}

		if content == "" {
			c.String(200, `{"Code":400,"Msg":"content is empty"}`)
			return nil
		}

		userName = strings.TrimSpace(util.RemoveCharacter(userName))
		if len(userName) == 0 {
			c.String(200, `{"Code":400,"Msg":"userName is empty"}`)
			return nil
		}

		stamp := util.GetCNTM(model.TimeOffSet)

		nameLow := strings.ToLower(userName)
		user, _ := model.UserGetByName(db, tx, nameLow)
		if user.ID > 0 {
			if user.Flag == model.FlagForbidden {
				c.String(200, `{"Code":403,"Msg":"user Flag is 0"}`)
				return nil
			}
		} else {
			// add user
			userId, _ := db.HIncr(tx, model.CountTb, mdb.S2b(model.UserTbName), 1)
			user = model.User{
				ID:       userId,
				Name:     userName,
				Flag:     model.FlagAuthor,
				Password: util.Md5(util.RandStringBytesMaskImprSrcSB(8)),
				RegTime:  uint64(stamp),
			}

			user = model.UserSet(db, tx, user)
			if user.ID == 0 {
				c.String(200, `{"Code":500,"Msg":"user set err"}`)
				return nil
			}
			_ = db.HSet(tx, "user_name2uid", []byte(nameLow), mdb.I2b(user.ID))
			_ = db.HSet(tx, "user_flag:"+strconv.Itoa(user.Flag), mdb.I2b(user.ID), nil)
			//生成头像
			_ = util.GenAvatar(db, tx, user.ID, user.Name)
		}

		var topic model.Topic
		var comment model.Comment

		if nodeId > 0 {
			// 优先发帖
			// add topic
			if title == "" {
				c.String(200, `{"Code":400,"Msg":"title is empty"}`)
				return nil
			}
			if _, cn := model.NodeGetById(db, tx, nodeId); cn != 1 {
				c.String(200, `{"Code":400,"Msg":"nodeId not exist"}`)
				return nil
			}
			// check title
			titleMd5 := fnv1a.HashString64(title)
			if db.HKeyExist(tx, "title_fnv1a", mdb.I2b(titleMd5)) {
				c.String(200, `{"Code":400,"Msg":"相同的文章标题已存在，请修改"}`)
				return nil
			}

			topic = model.Topic{
				UserId:   user.ID,
				NodeId:   nodeId,
				Title:    title,
				Content:  content,
				ClientIp: mdb.B2s(clientIp),
				AddTime:  stamp,
				EditTime: stamp,
			}
			// 直接保存
			topic = model.TopicAdd(h.App.Mc, db, tx, topic)

			// 自动从标题里提取标签
			if len(h.App.Cf.Site.GetTagApi) > 0 {
				_ = db.HSet(tx, "task_to_get_tag", mdb.I2b(topic.ID), mdb.S2b(topic.Title))
			}

			// 记录标题md5
			_ = db.HSet(tx, "title_fnv1a", mdb.I2b(titleMd5), mdb.I2b(topic.ID))

			c.String(200, `{"Code":200,"Msg":"ok, new topicId `+strconv.FormatUint(topic.ID, 10)+`"}`)
			return nil
		}

		// add comment
		if topicId <= 0 {
			c.String(200, `{"Code":400,"Msg":"topicId must > 0"}`)
			return nil
		}
		comment = model.Comment{
			UserId:   user.ID,
			TopicId:  topicId,
			AddTime:  stamp,
			Content:  content,
			ClientIp: mdb.B2s(clientIp),
		}
		comment = model.CommentAdd(h.App.Mc, db, tx, comment)
		c.String(200, `{"Code":200,"Msg":"ok, new commentId `+strconv.FormatUint(comment.ID, 10)+`"}`)
		return nil

	})
}
