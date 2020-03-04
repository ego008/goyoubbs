package cronjob

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ego008/youdb"
	"goyoubbs/model"
	"goyoubbs/system"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	subBaidu = "baidu"
	subBing  = "bing"
)

var tr = &http.Transport{
	TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	TLSHandshakeTimeout:   5 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}
var httpClient = &http.Client{
	Timeout:   time.Second * 30,
	Transport: tr,
}

// submit URL to baidu bing
func submitUrl(db *youdb.DB, scf *system.SiteConf) {
	var startKey []byte
	var curKey []byte
	var curValue []byte
	rs := db.Hget("status_kv", []byte("cur_aid_for_url_submit"))
	if rs.OK() {
		startKey = append(startKey, rs.Bytes()...)
		db.Hscan("article", startKey, 1).KvEach(func(key, value youdb.BS) {
			curKey = append(curKey, key...)
			curValue = append(curValue, value...)
		})
	} else {
		db.Hrscan("article", nil, 1).KvEach(func(key, value youdb.BS) {
			curKey = append(curKey, key...)
			curValue = append(curValue, value...)
		})
	}

	if len(curKey) == 0 {
		return
	}

	obj := model.Article{}
	err := json.Unmarshal(curValue, &obj)
	if err != nil {
		log.Println(err)
		return
	}

	aUrl := scf.MainDomain + "/t/" + strconv.FormatUint(youdb.B2i(curKey), 10)

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
				fmt.Println(err)
			} else {
				hasSubmit[subBaidu] = true
				_ = db.Hset("url_submit_status", []byte(subBaidu), curKey)
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
				fmt.Println(err)
			} else {
				hasSubmit[subBing] = true
				_ = db.Hset("url_submit_status", []byte(subBing), curKey)
			}
		}
	}

	for _, v := range hasSubmit {
		if !v {
			return
		}
	}

	_ = db.Hset("status_kv", []byte("cur_aid_for_url_submit"), curKey)
}

func baiduSubmit(scf *system.SiteConf, reqBody string) (err error) {
	var req *http.Request
	req, err = http.NewRequest("POST", scf.BaiduSubUrl, bytes.NewBufferString(reqBody))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "text/plain")
	var rsp *http.Response
	rsp, err = httpClient.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	var body []byte
	body, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return
	}

	fmt.Println(string(body))

	if !strings.Contains(string(body), "success") {
		err = errors.New("baidu resp err " + string(body))
		return
	}

	return nil
}

func bingSubmit(scf *system.SiteConf, reqBody string) (err error) {
	var req *http.Request
	req, err = http.NewRequest("POST", scf.BingSubUrl, bytes.NewBufferString(reqBody))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	var rsp *http.Response
	rsp, err = httpClient.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	var body []byte
	body, err = ioutil.ReadAll(rsp.Body)
	if err != nil {
		return
	}

	fmt.Println(string(body))

	if !strings.Contains(string(body), `"d"`) {
		err = errors.New("bing resp err " + string(body))
		return
	}

	return nil
}
