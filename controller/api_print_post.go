package controller

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/gin-gonic/gin"
	"go.etcd.io/bbolt"
)

func (h *BaseHandler) ApiAdminPrintPost(c *gin.Context) {
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

	body, _ := c.GetRawData()

	rec := struct {
		Pw string `json:"pw"`
	}{}
	err := json.Unmarshal(body, &rec)
	if err != nil {
		c.String(200, `{"Code":500,"Msg":"json.Unmarshal error"}`)
		return
	}

	remotePostPw := rec.Pw

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

		return nil
	})

	bodyStr := string(body)
	bodyStr = strings.Replace(bodyStr, `"pw":"`+rec.Pw+`",`, "", -1)
	log.Println(bodyStr)
	c.String(200, `{"Code":200,"Msg":"ok"}`)
	return
}
