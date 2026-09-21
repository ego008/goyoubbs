package controller

import (
	"fmt"
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/ybs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/fasthash/fnv1a"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) TopicAddPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAuthor {
		c.Redirect(302, "/login")
		return
	}

	db := h.App.Db
	scf := h.App.Cf.Site

	evn := &ybs.UserTopicAdd{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "发表文章"
	evn.PageName = "topic_input"

	nid := c.Query("nid")
	nidInt, err := strconv.ParseUint(nid, 10, 64)
	if err != nil {
		nidInt = 1
	}

	_ = db.View(func(tx *bbolt.Tx) error {

		node, _ := model.NodeGetById(db, tx, nidInt)

		evn.ReadMoreBreak = model.ReadMoreBreak
		evn.DefaultTopic = model.Topic{
			NodeId: 1,
			UserId: curUser.ID,
		}
		evn.DefaultUser = evn.CurrentUser
		evn.DefaultNode = node
		evn.NodeLst = model.NodeGetAll(h.App.Mc, db, tx)
		evn.HasMsg = model.MsgCheckHasOne(db, tx, curUser.ID)

		if curUser.Flag >= model.FlagAdmin {
			evn.UserLst = model.UserGetAllAdmin(db, tx)
			evn.HasTopicReview = model.CheckHasTopic2Review(db, tx)
			evn.HasReplyReview = model.CheckHasComment2Review(db, tx)
		}
		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	ybs.WritePageTemplate(c.Writer, evn)
}

// TopicAddPost 发表
func (h *BaseHandler) TopicAddPost(c *gin.Context) {
	c.Header("Content-Type", "application/json; charset=UTF-8")

	curUser, _ := h.CurrentUser(c)

	if curUser.Flag < model.FlagAuthor {
		c.String(200, `{"Code":401,"Msg":"请先登录"}`)
		return
	}

	var rec model.TopicRecForm
	err := util.Bind(c, util.JSON, &rec)
	if err != nil {
		c.String(200, `{"Code":400,"Msg":"unable to read body"}`)
		return
	}

	db := h.App.Db
	scf := h.App.Cf.Site

	rec.Title = strings.TrimSpace(rec.Title)
	rec.Content = strings.TrimSpace(rec.Content)

	titleLen := len(rec.Title)
	if titleLen == 0 {
		c.String(200, `{"Code":400,"Msg":"文章标题不能为空"}`)
		return
	} else if titleLen > scf.TitleMaxLen {
		msg := fmt.Sprintf(`{"Code":400,"Msg":"文章标题太长 %d > %d "}`, titleLen, scf.TitleMaxLen)
		c.String(200, msg)
		return
	}

	contentLen := len(rec.Content)
	if curUser.Flag < model.FlagAdmin && contentLen > scf.TopicConMaxLen {
		msg := fmt.Sprintf(`{"Code":400,"Msg":"文章内容太长 %d > %d "}`, contentLen, scf.TopicConMaxLen)
		c.String(200, msg)
		return
	}

	// edit
	var oldTopic model.Topic
	var isEdit bool
	if rec.ID > 0 {
		isEdit = true
		// fix
		if curUser.Flag < model.FlagAdmin {
			c.String(200, `{"Code":403,"Msg":"权限限制"}`)
			return
		}
	}

	_ = db.Update(func(tx *bbolt.Tx) error {
		// check title
		titleMd5 := fnv1a.HashString64(rec.Title)
		var showStr string
		_ = db.HGetFunc(tx, "title_fnv1a", mdb.I2b(titleMd5), func(val []byte) error {
			if rec.ID != mdb.B2i(val) {
				showStr = `{"Code":400,"Msg":"相同的文章标题已存在，请修改"}`
				return nil
			}
			return nil
		})
		if len(showStr) > 0 {
			c.String(200, showStr)
			return nil
		}

		var topic model.Topic
		if isEdit {
			topic = model.TopicGetById(db, tx, rec.ID)
			if topic.ID == 0 {
				c.String(200, `{"Code":400,"Msg":"该 id 帖子不存在"}`)
				return nil
			}
			oldTopic = topic
		}

		topic.NodeId = rec.NodeId
		topic.UserId = rec.UserId // curUser.ID
		topic.Title = rec.Title
		topic.Content = rec.Content
		topic.AddTime = rec.AddTime // util.GetCNTM()
		topic.ReadAuthed = rec.ReadAuthed
		topic.ReadReply = rec.ReadReply

		// may fix
		if topic.UserId == 0 {
			topic.UserId = curUser.ID
		}
		if topic.AddTime == 0 {
			topic.AddTime = util.GetCNTM(model.TimeOffSet)
		}
		topic.EditTime = topic.AddTime

		// 审核发帖删掉信息
		if !isEdit && curUser.Flag >= model.FlagAdmin {
			if rec.AddTime > 0 {
				// 删掉管理员列表
				_ = db.HDel(tx, model.TopicReviewTbName, mdb.I2b(uint64(rec.AddTime)))
				// 删掉个人待审核列表
				_ = db.HDel(tx, "review_topic:"+strconv.FormatUint(rec.UserId, 10), mdb.I2b(uint64(rec.AddTime)))
			}
		}

		type response struct {
			model.NormalRsp
			Tid uint64
		}

		rsp := response{}
		rsp.Code = 200

		stamp := util.GetCNTM(model.TimeOffSet)
		// 编辑
		if isEdit {
			if curUser.Flag < model.FlagAdmin {
				c.String(200, `{"Code":403,"Msg":"权限限制"}`)
				return nil
			}
			if oldTopic.Title != topic.Title || oldTopic.Content != topic.Content {
				topic.EditTime = stamp
			}
			// 直接更新
			model.TopicSet(db, tx, topic)
			// 分类、title 变化
			if oldTopic.NodeId != topic.NodeId {
				_ = db.ZSet(tx, "topic_update:"+strconv.FormatUint(topic.NodeId, 10), mdb.I2b(topic.ID), uint64(topic.AddTime))
				_ = db.ZDel(tx, "topic_update:"+strconv.FormatUint(oldTopic.NodeId, 10), mdb.I2b(topic.ID))
			}
			if oldTopic.Title != topic.Title {
				_ = db.HDel(tx, "title_fnv1a", mdb.I2b(fnv1a.HashString64(oldTopic.Title)))
				_ = db.HSet(tx, "title_fnv1a", mdb.I2b(titleMd5), mdb.I2b(topic.ID))
				// 自动从标题里提取标签
				if len(scf.GetTagApi) > 0 {
					_ = db.HSet(tx, "task_to_get_tag", mdb.I2b(topic.ID), mdb.S2b(topic.Title))
				}
			}
			rsp.Tid = topic.ID
			_ = json.NewEncoder(c.Writer).Encode(rsp)

			// 删除缓存
			h.App.Mc.Del([]byte("ContentFmt:" + strconv.FormatUint(topic.ID, 10)))

			return nil
		}

		// 以下是添加或审核
		// get ip
		if topic.ClientIp == "" {
			// 审核不取ip
			clip := c.GetHeader("X-Forwarded-For")
			if len(clip) == 0 {
				clip = c.GetHeader("X-FORWARDED-FOR")
			}
			topic.ClientIp = clip
		}
		if curUser.Flag < model.FlagTrust && scf.PostReview {
			// 非管理员+开启审核
			// 检测限制，防止机器恶意灌水
			showStr = ""
			_ = db.HGetFunc(tx, "userLastPostTime", mdb.I2b(curUser.ID), func(val []byte) error {
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

			if model.TopicGetV2ReviewNum(db, tx, curUser.ID) >= 10 {
				c.String(200, `{"Code":403,"Msg":"请勿灌水"}`)
				return nil
			}

			// 把jb 内容暂存到审核列表
			jb, _ := json.Marshal(topic)
			// 给管理员看
			_ = db.HSet(tx, model.TopicReviewTbName, mdb.I2b(uint64(topic.AddTime)), jb)
			// 把key 放到个人的列表
			_ = db.HSet(tx, "review_topic:"+strconv.FormatUint(topic.UserId, 10), mdb.I2b(uint64(topic.AddTime)), nil)
			// 记录最后请求发表时间
			_ = db.HSet(tx, "userLastPostTime", mdb.I2b(curUser.ID), mdb.I2b(uint64(stamp)))
			rsp.Code = 201
			rsp.Msg = "* 您的帖子已经提交，系统开启了发帖审核，请耐心等管理员审核"

			// 构建邮件信息，给管理员发邮件，尽快来验证
			if scf.SendEmail {
				mailInfo := model.EmailInfo{}
				mailInfo.Key = uint64(time.Now().UTC().UnixNano())
				mailInfo.Subject = scf.Name + " " + curUser.Name + "发帖《" + rec.Title + "》审核"
				mailInfo.Body = "这是一封系统通知邮件：" + curUser.Name + "于" + util.TimeFmt(topic.AddTime, "") + " 发帖，IP" + topic.ClientIp + " 内容摘要：<br><br>" + util.GetDesc(rec.Content) + "<br><br>请尽快前往处理 " + scf.MainDomain + "/admin/topic/review"
				model.EmailInfoUpdate(db, tx, mailInfo)
			}

			_ = json.NewEncoder(c.Writer).Encode(rsp)
			return nil
		}

		// 直接保存
		topic = model.TopicAdd(h.App.Mc, db, tx, topic)
		rsp.Tid = topic.ID

		// 自动从标题里提取标签
		if len(scf.GetTagApi) > 0 {
			_ = db.HSet(tx, "task_to_get_tag", mdb.I2b(topic.ID), mdb.S2b(topic.Title))
		}

		// 记录标题md5
		_ = db.HSet(tx, "title_fnv1a", mdb.I2b(titleMd5), mdb.I2b(topic.ID))

		_ = json.NewEncoder(c.Writer).Encode(rsp)

		return nil
	})
}
