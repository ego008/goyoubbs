package cronjob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ego008/sdb"
	"github.com/valyala/fasthttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/indexing/v3"
	"google.golang.org/api/option"
	"goyoubbs/model"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	subBaidu   = "baidu"
	subBing    = "bing"
	subGoogle  = "google"
	leftSecond = 60 // 离过期时间 秒
)

var (
	conf        *jwt.Config
	tokenSource oauth2.TokenSource
	token       *oauth2.Token
)

// submit to baidu bing google
func submitUrl(db *sdb.DB, scf *model.SiteConf) {
	var startKey []byte
	var curKey []byte
	var curValue []byte
	rs := db.Hget("status_kv", []byte("cur_aid_for_url_submit"))
	if rs.OK() {
		startKey = append(startKey, rs.Bytes()...)
		db.Hscan(model.TopicTbName, startKey, 1).KvEach(func(key, value sdb.BS) {
			curKey = append(curKey, key...)
			curValue = append(curValue, value...)
		})
	} else {
		// 只推最后一条
		db.Hrscan(model.TopicTbName, nil, 1).KvEach(func(key, value sdb.BS) {
			curKey = append(curKey, key...)
			curValue = append(curValue, value...)
		})
	}

	if len(curKey) == 0 {
		return
	}

	obj := model.Topic{}
	err := json.Unmarshal(curValue, &obj)
	if err != nil {
		log.Println(err)
		return
	}

	aUrl := scf.MainDomain + "/t/" + strconv.FormatUint(sdb.B2i(curKey), 10)
	log.Println("submit aUrl", aUrl)

	hasSubmit := map[string]bool{}

	// baidu
	if scf.BaiduSubUrl != "" {
		hasSubmit[subBaidu] = false
		rs := db.Hget("url_submit_status", []byte(subBaidu))
		if bytes.Equal(rs.Bytes(), curKey) {
			hasSubmit[subBaidu] = true
		} else {
			err = baiduSubmit(scf, aUrl)
			if err != nil {
				log.Println("baidu Submit err", err)
			} else {
				hasSubmit[subBaidu] = true
				_ = db.Hset("url_submit_status", []byte(subBaidu), curKey)
				log.Println("baidu Submit ok :)")
			}
		}
	}

	// bing
	if scf.BingSubUrl != "" {
		hasSubmit[subBing] = false
		rs := db.Hget("url_submit_status", []byte(subBing))
		if bytes.Equal(rs.Bytes(), curKey) {
			hasSubmit[subBing] = true
		} else {
			err = bingSubmit(scf, `{"siteUrl":"`+scf.MainDomain+`","url":"`+aUrl+`"}`)
			if err != nil {
				log.Println("bing Submit err", err)
			} else {
				hasSubmit[subBing] = true
				_ = db.Hset("url_submit_status", []byte(subBing), curKey)
				log.Println("bing Submit ok :)")
			}
		}
	}

	// google
	if scf.GoogleJWTConf != "" {
		hasSubmit[subGoogle] = false
		rs := db.Hget("url_submit_status", []byte(subGoogle))
		if bytes.Equal(rs.Bytes(), curKey) {
			hasSubmit[subGoogle] = true
		} else {
			// // {"type":"URL_UPDATED","url":"https://www.youbbs.org/t/110"}
			o := indexing.UrlNotification{
				Type: "URL_UPDATED",
				Url:  scf.MainDomain + "/t/" + strconv.FormatUint(sdb.B2i(curKey), 10),
			}
			err = googleSubmit(scf, o)
			if err != nil {
				log.Println("google submit err", err)
			} else {
				hasSubmit[subGoogle] = true
				_ = db.Hset("url_submit_status", []byte(subGoogle), curKey)
				log.Println("google Submit ok :)")
			}
		}
	}

	for _, v := range hasSubmit {
		if !v {
			return
		}
	}

	_ = db.Hset("status_kv", []byte("cur_aid_for_url_submit"), curKey)
	log.Println("all submit done", aUrl)
}

func baiduSubmit(scf *model.SiteConf, reqBody string) (err error) {
	log.Println("baidu Submit")
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()

	req.SetRequestURI(scf.BaiduSubUrl)
	req.Header.SetMethod("POST")
	req.Header.Set("Content-Type", "text/plain")
	req.SetBodyString(reqBody)

	err = fastHttpClient.Do(req, res)
	if err != nil {
		return
	}

	if res.StatusCode() != fasthttp.StatusOK {
		fmt.Println("res.StatusCode", res.StatusCode())
		err = errors.New("res.StatusCode not ok: " + strconv.Itoa(res.StatusCode()))
		return
	}

	bodyStr := string(res.Body())

	fmt.Println(bodyStr)

	if !strings.Contains(bodyStr, "success") {
		err = errors.New("baidu resp err " + bodyStr)
		return
	}

	err = nil
	return
}

func bingSubmit(scf *model.SiteConf, reqBody string) (err error) {
	log.Println("bing Submit")
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()

	req.SetRequestURI(scf.BingSubUrl)
	req.Header.SetMethod("POST")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.SetBodyString(reqBody)

	err = fastHttpClient.Do(req, res)
	if err != nil {
		return
	}

	if res.StatusCode() != fasthttp.StatusOK {
		fmt.Println("res.StatusCode", res.StatusCode())
		err = errors.New("res.StatusCode not ok: " + strconv.Itoa(res.StatusCode()))
		return
	}

	bodyStr := string(res.Body())

	fmt.Println(bodyStr)

	if !strings.Contains(bodyStr, `"d"`) {
		err = errors.New("bing resp err " + bodyStr)
		return
	}

	err = nil
	return
}

func googleSubmit(scf *model.SiteConf, o indexing.UrlNotification) (err error) {
	log.Println("google Submit")
	//var jsData []byte
	//jsData, err = ioutil.ReadFile(scf.GoogleSubKey)
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	conf, err = google.JWTConfigFromJSON([]byte(scf.GoogleJWTConf), indexing.IndexingScope)
	if err != nil {
		fmt.Println("#JWTConfigFromJSON err", err)
		return
	}
	tokenSource = conf.TokenSource(context.Background())

	// check token is expiry
	tn := time.Now().UTC()
	if token == nil || token.Expiry.Sub(tn).Seconds() < leftSecond {
		// get Token from remote
		token, err = tokenSource.Token()
		if err != nil {
			log.Println("#ts.Token err", err)
			return
		}
	}

	var indexingService *indexing.Service
	ctx := context.Background()
	indexingService, err = indexing.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		log.Println(err)
		return
	}

	req := indexingService.UrlNotifications.Publish(&o)
	var rsp *indexing.PublishUrlNotificationResponse
	rsp, err = req.Do()
	if err != nil {
		log.Println(err)
		return
	}
	// fmt.Println(rsp)
	var buf []byte
	buf, err = rsp.MarshalJSON()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(buf))

	return nil
}
