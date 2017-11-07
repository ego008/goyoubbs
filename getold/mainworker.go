package getold

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ego008/goyoubbs/model"
	"github.com/ego008/goyoubbs/system"
	"github.com/ego008/goyoubbs/util"
	"github.com/ego008/youdb"
	"github.com/weint/httpclient"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type BaseHandler struct {
	App *system.Application
}

var tbs = []string{"users", "articles", "categories", "comments", "tags", "qqweibo", "weibo"}

func (h *BaseHandler) GetRemote() error {
	db := h.App.Db
	oldDomain := h.App.Cf.Main.OldSiteDomain

	for _, tb := range tbs {
		fmt.Println("------tb", tb)
		for {
			rs := db.Hget("getold_last_tb_id", []byte(tb))
			curId := uint64(1)
			if rs.State == "ok" {
				curId = rs.Uint64()
			}

			timer := time.NewTimer(100 * time.Millisecond)
			<-timer.C
			url := oldDomain + "/get/" + tb + "/" + strconv.FormatUint(curId, 10)
			fmt.Println(url)
			hc := httpclient.Get(url)
			resp, err := hc.Response()
			if err != nil {
				fmt.Println(err)
				continue
			}

			if resp.StatusCode == 200 {
				if resp.Body == nil {
					continue
				}
				data, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					continue
				}
				resp.Body.Close()

				// save to db for local - for tmp data
				db.Hset("old_data:"+tb, youdb.I2b(curId), data)
				db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
			} else {
				fmt.Println("reply state", resp.StatusCode)
				break
			}
		}
	}
	fmt.Println("remote done")
	return nil
}

func (h *BaseHandler) GetLocal() error {
	db := h.App.Db

	for _, tb := range tbs {
		fmt.Println("------tb", tb)
		db.Hset("getold_last_tb_id", []byte(tb), youdb.I2b(1))
		for {
			timer := time.NewTimer(5 * time.Millisecond)
			<-timer.C
			rs := db.Hget("getold_last_tb_id", []byte(tb))
			curId := uint64(1)
			if rs.State == "ok" {
				curId = rs.Uint64()
			}
			fmt.Println(tb, curId)

			rs3 := db.Hget("old_data:"+tb, youdb.I2b(curId))

			if rs3.State == "ok" {
				data := rs3.Data[0]

				switch {
				case tb == "users":
					t := OldUser{}
					err := json.Unmarshal(data, &t)
					if err == nil {
						if t.Code == 404 {
							db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
							continue
						}
						obj := model.User{
							Id:            youdb.DS2i(t.Id),
							Name:          t.Name,
							Flag:          int(youdb.DS2i(t.Flag)),
							Avatar:        t.Avatar,
							Password:      t.Password,
							Email:         t.Email,
							Url:           t.Url,
							Articles:      youdb.DS2i(t.Articles),
							Replies:       youdb.DS2i(t.Replies),
							RegTime:       youdb.DS2i(t.RegTime),
							LastPostTime:  youdb.DS2i(t.LastPostTime),
							LastReplyTime: youdb.DS2i(t.LastReplyTime),
							LastLoginTime: youdb.DS2i(t.LastLoginTime),
							About:         t.About,
							Notice:        strings.Trim(t.Notice, ","),
							Hidden:        false,
						}
						if len(obj.Notice) > 0 {
							obj.NoticeNum = len(util.SliceUniqStr(strings.Split(obj.Notice, ",")))
						}
						jb, _ := json.Marshal(obj)
						db.Hset("user", youdb.I2b(obj.Id), jb)
						db.HsetSequence("user", obj.Id)
						db.Hset("user_name2uid", []byte(strings.ToLower(t.Name)), youdb.I2b(obj.Id))
						db.Hset("user_flag:"+t.Flag, youdb.I2b(obj.Id), []byte(""))
						db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
					}
				case tb == "articles":
					t := OldArticle{}
					err := json.Unmarshal(data, &t)
					if err == nil {
						if t.Code == 404 {
							db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
							continue
						}

						obj := model.Article{
							Id:           youdb.DS2i(t.Id),
							Uid:          youdb.DS2i(t.Uid),
							Cid:          youdb.DS2i(t.Cid),
							RUid:         youdb.DS2i(t.RUid),
							Title:        t.Title,
							Content:      t.Content,
							Tags:         t.Tags,
							AddTime:      youdb.DS2i(t.AddTime),
							EditTime:     youdb.DS2i(t.EditTime),
							Comments:     youdb.DS2i(t.Comments),
							CloseComment: false,
							Hidden:       false,
						}
						jb, _ := json.Marshal(obj)

						db.Hset("article", youdb.I2b(obj.Id), jb)
						db.HsetSequence("article", obj.Id)
						// 浏览数
						db.Hset("article_views", youdb.I2b(obj.Id), youdb.I2b(youdb.DS2i(t.Views)))
						// 总文章列表
						db.Zset("article_timeline", youdb.I2b(obj.Id), obj.EditTime)
						// 分类文章列表
						db.Zset("category_article_timeline:"+strconv.FormatUint(obj.Cid, 10), youdb.I2b(obj.Id), obj.EditTime)
						// 用户文章列表
						db.Hset("user_article_timeline:"+strconv.FormatUint(obj.Uid, 10), youdb.I2b(obj.Id), []byte(""))
						// 分类下文章数
						db.Zincr("category_article_num", youdb.I2b(obj.Cid), 1)
						// title md5
						hash := md5.Sum([]byte(obj.Title))
						titleMd5 := hex.EncodeToString(hash[:])
						db.Hset("title_md5", []byte(titleMd5), youdb.I2b(obj.Id))

						db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
					}
				case tb == "categories":
					t := OldCategory{}
					err := json.Unmarshal(data, &t)
					if err == nil {
						if t.Code == 404 {
							db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
							continue
						}
						obj := model.Category{
							Id:       youdb.DS2i(t.Id),
							Name:     t.Name,
							Articles: youdb.DS2i(t.Articles),
							About:    t.About,
							Hidden:   false,
						}
						jb, _ := json.Marshal(obj)

						db.Hset("category", youdb.I2b(obj.Id), jb)
						db.HsetSequence("category", obj.Id)          // 分类数
						db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
					}
				case tb == "comments":
					t := OldComment{}
					err := json.Unmarshal(data, &t)
					if err == nil {
						if t.Code == 404 {
							db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
							continue
						}

						rs := db.Hget("article", youdb.DS2b(t.Aid))
						if rs.State == "ok" {
							commentId, _ := db.HnextSequence("article_comment:" + t.Aid)

							newContent := strings.Replace(t.Content, "http://youbbs.sinaapp.com", "https://www.youbbs.org", -1)
							newContent = strings.Replace(newContent, "https://www.youbbs.org/t-", "https://www.youbbs.org/t/", -1)
							newContent = strings.Replace(newContent, "https://www.youbbs.org/n-", "https://www.youbbs.org/n/", -1)

							obj := model.Comment{
								Id:      commentId,
								Aid:     youdb.DS2i(t.Aid),
								Uid:     youdb.DS2i(t.Uid),
								Content: newContent,
								AddTime: youdb.DS2i(t.AddTime),
							}
							jb, _ := json.Marshal(obj)

							db.Hset("article_comment:"+t.Aid, youdb.I2b(obj.Id), jb) // 文章评论bucket
							db.Hset("count", []byte("comment_num"), youdb.DS2b(t.Id))
							// 用户回复文章列表
							db.Zset("user_article_reply:"+strconv.FormatUint(obj.Uid, 10), youdb.I2b(obj.Aid), obj.AddTime)
						}
						db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
					}
				case tb == "tags":
					t := OldTag{}
					err := json.Unmarshal(data, &t)
					if err == nil {
						if t.Code == 404 {
							db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
							continue
						}
						if len(t.Ids) > 0 {
							tagLow := strings.ToLower(t.Name)
							db.Hset("tag", []byte(tagLow), []byte(""))
							db.HsetSequence("tag", youdb.DS2i(t.Id)) // 标签个数
							aids := strings.Split(t.Ids, ",")
							for _, aid := range aids {
								db.Hset("tag:"+tagLow, youdb.I2b(youdb.DS2i(aid)), []byte(""))
							}
							db.Zset("tag_article_num", []byte(tagLow), uint64(len(aids))) // 标签文章个数
						}
						db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
					}
				case tb == "qqweibo":
					t := OldQQ{}
					err := json.Unmarshal(data, &t)
					if err == nil {
						if t.Code == 404 {
							db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
							continue
						}
						obj := model.QQ{
							Uid:    youdb.DS2i(t.Uid),
							Name:   t.Name,
							Openid: t.Openid,
						}
						jb, _ := json.Marshal(obj)

						db.Hset("oauth_qq", []byte(t.Openid), jb)
						db.HnextSequence("oauth_qq")
						db.Hset("qq_uid2openid", youdb.I2b(obj.Uid), []byte(t.Openid))
						db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
					}
				case tb == "weibo":
					t := OldWeibo{}
					err := json.Unmarshal(data, &t)
					if err == nil {
						if t.Code == 404 {
							db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
							continue
						}
						obj := model.WeiBo{
							Uid:    youdb.DS2i(t.Uid),
							Name:   t.Name,
							Openid: t.Openid,
						}
						jb, _ := json.Marshal(obj)

						db.Hset("oauth_weibo", []byte(t.Openid), jb)
						db.HnextSequence("oauth_weibo")
						db.Hset("weibo_uid2openid", youdb.I2b(obj.Uid), []byte(t.Openid))
						db.Hincr("getold_last_tb_id", []byte(tb), 1) // count flag
					}
				}
			} else {
				fmt.Println("reply state", rs3.State)
				break
			}
		}
		db.HdelBucket("old_data:" + tb)
	}
	fmt.Println("local done")
	return nil
}

func (h *BaseHandler) DelOldData() {
	db := h.App.Db
	for _, tb := range tbs {
		db.HdelBucket("old_data:" + tb)
	}
}

func downloadFromUrl(url string) (string, int) {
	// fmt.Println("url:", url)
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	newFilePath := "static/avatar/" + fileName

	if _, err := os.Stat(newFilePath); err == nil {
		// path exists
		return newFilePath, 0
	}

	hc := httpclient.NewHttpClientRequest("GET", url)
	hc.SetTimeout(600 * time.Second)
	resp, err := hc.Response()
	defer resp.Body.Close()
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return "", 1
	}
	if resp.StatusCode != 200 {
		fmt.Println("qiniu resp.StatusCode != 200", resp.StatusCode)
		if resp.StatusCode == 404 {
			return "", 404
		}
		return "", 3
	}

	//fmt.Println("newFilePath", newFilePath)
	out, err := os.Create(newFilePath)
	defer out.Close()

	nb, err := io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return "", 2
	}
	fmt.Println(nb, "bytes downloaded.")

	return newFilePath, 0
}
