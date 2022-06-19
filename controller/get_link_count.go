package controller

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"goyoubbs/model"
	"goyoubbs/util"
	"strconv"
	"time"
)

const linkCountTbName = "link_click_count"

func (h *BaseHandler) GetLinkCount(ctx *fasthttp.RequestCtx) {
	token := h.GetCookie(ctx, "token")
	if len(token) == 0 {
		_, _ = ctx.WriteString(`{"Code":403, "Msg": "token err"}`)
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

	err := util.Bind(ctx, util.JSON, &rec)
	if err != nil {
		_, _ = ctx.WriteString(`{"Code":400, "Msg": "解析出错"}`)
		return
	}

	if len(rec.Items) == 0 {
		_, _ = ctx.WriteString(`{"Code":400, "Msg": "没有 href"}`)
		return
	}

	rsp := response{}
	rsp.Code = 201
	if rec.Act == "set" {
		// 点击
		enLinkI64 := util.Xxhash(sdb.S2b(rec.Items[0]))
		clickTbName := "article_detail_token" // 点击限制，24小时只能计数一次，后台任务需要清除过期数据
		clickKey := sdb.S2b(token + ":click:" + strconv.FormatUint(enLinkI64, 10))
		if db.Zget(clickTbName, clickKey) > 0 {
			_, _ = ctx.WriteString(`{"Code":403, "Msg": "click count 1/day"}`)
			return
		}
		num, err := db.Hincr(linkCountTbName, sdb.I2b(enLinkI64), 1)
		if err != nil {
			_, _ = ctx.WriteString(`{"Code":500, "Msg": "计数失败"}`)
			return
		}
		_ = db.Zset(clickTbName, clickKey, uint64(time.Now().UTC().Unix())) // 计时限制

		rsp.Num = strconv.FormatUint(num, 10)
		rsp.Code = 200

		_ = json.NewEncoder(ctx).Encode(rsp)
		return
	}

	// 获取
	u64ToLink := map[uint64]string{}
	info := map[string]string{}
	var keys [][]byte

	// fix for old md5 data
	hasMd5Data := db.Hscan("url_md5_click", nil, 1).OK()

	for _, v := range rec.Items {
		enLinkI64 := util.Xxhash(sdb.S2b(v))
		u64ToLink[enLinkI64] = v
		keys = append(keys, sdb.I2b(enLinkI64))

		// fix md5,遍历后删除
		if hasMd5Data {
			urlMd5 := util.Md5(v)
			if rs := db.HgetInt("url_md5_click", []byte(urlMd5)); rs > 0 {
				_ = db.Hset(linkCountTbName, sdb.I2b(enLinkI64), sdb.I2b(rs))
				_ = db.Hdel("url_md5_click", []byte(urlMd5))
			}
		}
	}

	db.Hmget(linkCountTbName, keys).KvEach(func(key, value sdb.BS) {
		enLinkI64 := sdb.B2i(key)
		info[u64ToLink[enLinkI64]] = strconv.FormatUint(sdb.B2i(value), 10)
	})

	rsp.Code = 200
	rsp.Info = info
	_ = json.NewEncoder(ctx).Encode(rsp)
}
