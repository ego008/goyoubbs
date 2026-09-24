package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"goyoubbs/views/admin"
	"net/http"
	"strconv"
	"strings"

	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) AdminUserPage(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/login")
		return
	}

	scf := h.App.Cf.Site

	evn := &admin.User{}
	evn.CurrentUser = *curUser
	evn.SiteCf = scf
	evn.Title = "用户管理"

	evn.FlagLst = []model.Flag{
		{model.FlagForbidden, "0 禁用"},
		{model.FlagReview, "1 待审核"},
		{model.FlagAuthor, "5 普通"},
		{model.FlagTrust, "10 可信任"},
		{model.FlagAdmin, "99 管理员"},
	}

	//
	evn.Act = "添加"
	evn.User = model.User{}
	_id := c.Query("id")

	_ = h.App.Db.View(func(tx *bbolt.Tx) error {

		if len(_id) > 0 {
			idi, _ := strconv.ParseUint(_id, 10, 64)
			evn.User = model.User{}
			user, code := model.UserGetById(h.App.Db, tx, idi)
			if code == 1 {
				evn.User = user
				evn.Act = "编辑"
			}
		}

		if evn.User.ID == 0 {
			var tbn string
			flag := c.Query("flag")
			if len(flag) > 0 {
				tbn = "user_flag:" + flag
			} else {
				tbn = model.UserTbName
			}

			var userLst []model.User
			q := strings.TrimSpace(c.Query("q"))
			if len(q) > 0 {
				// 搜索用户
				userLst = model.UserGetRecentByKw(h.App.Db, tx, q, 100)
			} else {
				userLst = model.UserGetRecentByFlag(h.App.Db, tx, tbn, 100)
			}
			evn.UserLst = userLst
		}

		evn.NodeLst = model.NodeGetAll(h.App.Mc, h.App.Db, tx)

		evn.HasMsg = model.MsgCheckHasOne(h.App.Db, tx, curUser.ID)
		evn.HasTopicReview = model.CheckHasTopic2Review(h.App.Db, tx)
		evn.HasReplyReview = model.CheckHasComment2Review(h.App.Db, tx)

		return nil
	})

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	admin.WritePageTemplate(c.Writer, evn)
}

func (h *BaseHandler) AdminUserPost(c *gin.Context) {
	curUser, _ := h.CurrentUser(c)
	if curUser.Flag < model.FlagAdmin {
		c.Redirect(302, "/admin")
		return
	}

	var id uint64
	_id := c.Query("id")
	if len(_id) > 0 {
		idI, err := strconv.ParseUint(_id, 10, 64)
		if err == nil {
			id = idI
		}
	}

	var obj model.User
	var isAdd bool
	var oldFlag int

	db := h.App.Db

	_ = db.Update(func(tx *bbolt.Tx) error {
		if id > 0 {
			// 编辑
			obj, _ = model.UserGetById(db, tx, id)
			if obj.ID == 0 {
				c.String(200, `{"Code":400,"Msg":"not has this id"}`)
				return nil
			}
			oldFlag = obj.Flag
		} else {
			// 添加
			fName := strings.TrimSpace(c.PostForm("Name"))
			// 检测重名
			nameLow := strings.ToLower(fName)
			if !util.IsNickname(nameLow) {
				c.String(200, `{"Code":400,"Msg":"name fmt err"}`)
				return nil
			}
			tmpObj, _ := model.UserGetByName(db, tx, nameLow)
			if tmpObj.ID > 0 {
				c.String(200, `{"Code":400,"Msg":"name is exist"}`)
				return nil
			}

			isAdd = true
			userId, _ := db.HIncr(tx, model.CountTb, mdb.S2b(model.UserTbName), 1)
			obj = model.User{
				ID:      userId,
				Name:    fName,
				RegTime: uint64(util.GetCNTM(model.TimeOffSet)),
			}
		}

		pw := strings.TrimSpace(c.PostForm("Password"))
		if len(pw) > 0 {
			obj.Password = util.Md5(pw)
		}

		obj.Flag, _ = strconv.Atoi(c.PostForm("Flag"))
		obj.Url = c.PostForm("Url")
		obj.About = c.PostForm("About")

		obj = model.UserSet(db, tx, obj)

		if isAdd {
			nameLow := strings.ToLower(obj.Name)
			_ = db.HSet(tx, "user_name2uid", []byte(nameLow), mdb.I2b(obj.ID))
			_ = db.HSet(tx, "user_flag:"+strconv.Itoa(obj.Flag), mdb.I2b(obj.ID), nil)
			//生成头像
			_ = util.GenAvatar(db, tx, obj.ID, obj.Name)
		} else {
			if oldFlag != obj.Flag {
				_ = db.HSet(tx, "user_flag:"+strconv.Itoa(obj.Flag), mdb.I2b(obj.ID), nil)
				_ = db.HDel(tx, "user_flag:"+strconv.Itoa(oldFlag), mdb.I2b(obj.ID))
			}
		}

		c.Redirect(302, "/admin/user")
		return nil
	})

}
