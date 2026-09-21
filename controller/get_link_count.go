package controller

import (
	"goyoubbs/model"
	"goyoubbs/util"
	"strconv"
	"time"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

const linkCountTbName = "link_click_count"

func (h *BaseHandler) GetLinkCount(c *gin.Context) {
	token := h.GetCookie(c, "token")
	if len(token) == 0 {
		c.String(200, `{"Code":403, "Msg": "token err"}`)
		return
	}

	db := h.App.Db

	type response struct {
		model.NormalRsp
		Num  string            // 点击后返回点击后次数
		Info map[string]string // 批量获取
	}

	var rec = struct {
		Act   string // get/set
		Items []string
	}{}

	err := util.Bind(c, util.JSON, &rec)
	if err != nil {
		c.String(200, `{"Code":400, "Msg": "解析出错"}`)
		return
	}

	if len(rec.Items) == 0 {
		c.String(200, `{"Code":400, "Msg": "没有 href"}`)
		return
	}

	rsp := response{}
	rsp.Code = 201

	info := map[string]string{}

	_ = db.Update(func(tx *bbolt.Tx) error {

		if rec.Act == "set" {
			// 点击
			enLinkI64 := util.Xxhash(mdb.S2b(rec.Items[0]))
			clickTbName := "article_detail_token" // 点击限制，24小时只能计数一次，后台任务需要清除过期数据
			clickKey := mdb.S2b(token + ":click:" + strconv.FormatUint(enLinkI64, 10))
			if db.ZGetInt(tx, clickTbName, clickKey) > 0 {
				c.String(200, `{"Code":403, "Msg": "click count 1/day"}`)
				return nil
			}
			num, err := db.HIncr(tx, linkCountTbName, mdb.I2b(enLinkI64), 1)
			if err != nil {
				c.String(200, `{"Code":500, "Msg": "计数失败"}`)
				return nil
			}
			_ = db.ZSet(tx, clickTbName, clickKey, uint64(time.Now().UTC().Unix())) // 计时限制

			rsp.Num = strconv.FormatUint(num, 10)
			rsp.Code = 200

			_ = json.NewEncoder(c.Writer).Encode(rsp)
			return nil
		}

		// 获取
		u64ToLink := map[uint64]string{}

		var keys [][]byte

		// fix for old md5 data
		var hasMd5Data bool
		_ = db.HScanFunc(tx, "url_md5_click", nil, 1, func(key, val []byte) bool {
			hasMd5Data = true
			return true
		})

		for _, v := range rec.Items {
			enLinkI64 := util.Xxhash(mdb.S2b(v))
			u64ToLink[enLinkI64] = v
			keys = append(keys, mdb.I2b(enLinkI64))

			// fix md5,遍历后删除
			if hasMd5Data {
				urlMd5 := util.Md5(v)
				if rs := db.HGetInt(tx, "url_md5_click", []byte(urlMd5)); rs > 0 {
					_ = db.HSet(tx, linkCountTbName, mdb.I2b(enLinkI64), mdb.I2b(rs))
					_ = db.HDel(tx, "url_md5_click", []byte(urlMd5))
				}
			}
		}

		_ = db.HMGetFunc(tx, linkCountTbName, keys, func(key, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			enLinkI64 := mdb.B2i(key)
			info[u64ToLink[enLinkI64]] = strconv.FormatUint(mdb.B2i(val), 10)
			return nil
		})
		return nil
	})

	rsp.Code = 200
	rsp.Info = info
	_ = json.NewEncoder(c.Writer).Encode(rsp)
}
