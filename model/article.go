package model

import (
	"encoding/json"
	"errors"
	"goyoubbs/util"
	"github.com/ego008/youdb"
	"html"
	"sort"
	"strings"
	"time"
)

type Article struct {
	Id           uint64 `json:"id"`
	Uid          uint64 `json:"uid"`
	Cid          uint64 `json:"cid"`
	RUid         uint64 `json:"ruid"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	ClientIp     string `json:"clientip"`
	Tags         string `json:"tags"`
	AddTime      uint64 `json:"addtime"`
	EditTime     uint64 `json:"edittime"`
	Comments     uint64 `json:"comments"`
	CloseComment bool   `json:"closecomment"`
	Hidden       bool   `json:"hidden"`
}

type ArticleMini struct {
	Id       uint64 `json:"id"`
	Uid      uint64 `json:"uid"`
	Cid      uint64 `json:"cid"`
	Ruid     uint64 `json:"ruid"`
	Title    string `json:"title"`
	EditTime uint64 `json:"edittime"`
	Comments uint64 `json:"comments"`
	Hidden   bool   `json:"hidden"`
}

type ArticleListItem struct {
	Id          uint64 `json:"id"`
	Uid         uint64 `json:"uid"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	Cid         uint64 `json:"cid"`
	Cname       string `json:"cname"`
	Ruid        uint64 `json:"ruid"`
	Rname       string `json:"rname"`
	Title       string `json:"title"`
	EditTime    uint64 `json:"edittime"`
	EditTimeFmt string `json:"edittimefmt"`
	Comments    uint64 `json:"comments"`
}

type ArticlePageInfo struct {
	Items      []ArticleListItem `json:"items"`
	HasPrev    bool              `json:"hasprev"`
	HasNext    bool              `json:"hasnext"`
	FirstKey   uint64            `json:"firstkey"`
	FirstScore uint64            `json:"firstscore"`
	LastKey    uint64            `json:"lastkey"`
	LastScore  uint64            `json:"lastscore"`
}

type ArticleLi struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
	Tags  string `json:"tags"`
}

type ArticleRelative struct {
	Articles []ArticleLi
	Tags     []string
}

type ArticleFeedListItem struct {
	Id          uint64
	Uid         uint64
	Name        string
	Cname       string
	Title       string
	AddTimeFmt  string
	EditTimeFmt string
	Des         string
}

// 文章添加、编辑后传给后台任务的信息
type ArticleTag struct {
	Id      uint64
	OldTags string
	NewTags string
}

func ArticleGetById(db *youdb.DB, aid string) (Article, error) {
	obj := Article{}
	rs := db.Hget("article", youdb.DS2b(aid))
	if rs.State == "ok" {
		json.Unmarshal(rs.Data[0], &obj)
		return obj, nil
	}
	return obj, errors.New(rs.State)
}

func ArticleList(db *youdb.DB, cmd, tb, key, score string, limit, tz int) ArticlePageInfo {
	var items []ArticleListItem
	var keys [][]byte
	var hasPrev, hasNext bool
	var firstKey, firstScore, lastKey, lastScore uint64

	keyStart := youdb.DS2b(key)
	scoreStart := youdb.DS2b(score)
	if cmd == "zrscan" {
		rs := db.Zrscan(tb, keyStart, scoreStart, limit)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				keys = append(keys, rs.Data[i])
			}
		}
	} else if cmd == "zscan" {
		rs := db.Zscan(tb, keyStart, scoreStart, limit)
		if rs.State == "ok" {
			for i := len(rs.Data) - 2; i >= 0; i -= 2 {
				keys = append(keys, rs.Data[i])
			}
		}
	}

	if len(keys) > 0 {
		var aitems []ArticleMini
		userMap := map[uint64]UserMini{}
		categoryMap := map[uint64]CategoryMini{}

		rs := db.Hmget("article", keys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := ArticleMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				if !item.Hidden {
					aitems = append(aitems, item)
					userMap[item.Uid] = UserMini{}
					if item.Ruid > 0 {
						userMap[item.Ruid] = UserMini{}
					}
					categoryMap[item.Cid] = CategoryMini{}
				}
			}
		}

		userKeys := make([][]byte, 0, len(userMap))
		for k := range userMap {
			userKeys = append(userKeys, youdb.I2b(k))
		}
		rs = db.Hmget("user", userKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := UserMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				userMap[item.Id] = item
			}
		}

		categoryKeys := make([][]byte, 0, len(categoryMap))
		for k := range categoryMap {
			categoryKeys = append(categoryKeys, youdb.I2b(k))
		}
		rs = db.Hmget("category", categoryKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := CategoryMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				categoryMap[item.Id] = item
			}
		}

		for _, article := range aitems {
			user := userMap[article.Uid]
			category := categoryMap[article.Cid]
			item := ArticleListItem{
				Id:          article.Id,
				Uid:         article.Uid,
				Name:        user.Name,
				Avatar:      user.Avatar,
				Cid:         article.Cid,
				Cname:       category.Name,
				Ruid:        article.Ruid,
				Title:       article.Title,
				EditTime:    article.EditTime,
				EditTimeFmt: util.TimeFmt(article.EditTime, "2006-01-02 15:04", tz),
				Comments:    article.Comments,
			}
			if article.Ruid > 0 {
				item.Rname = userMap[article.Ruid].Name
			}
			items = append(items, item)
			if firstKey == 0 {
				firstKey = item.Id
				firstScore = item.EditTime
			}
			lastKey = item.Id
			lastScore = item.EditTime
		}

		// not fix hidden article
		rs = db.Zscan(tb, youdb.I2b(firstKey), youdb.I2b(firstScore), 1)
		if rs.State == "ok" {
			hasPrev = true
		}
		rs = db.Zrscan(tb, youdb.I2b(lastKey), youdb.I2b(lastScore), 1)
		if rs.State == "ok" {
			hasNext = true
		}

	}

	return ArticlePageInfo{
		Items:      items,
		HasPrev:    hasPrev,
		HasNext:    hasNext,
		FirstKey:   firstKey,
		FirstScore: firstScore,
		LastKey:    lastKey,
		LastScore:  lastScore,
	}
}

func ArticleGetRelative(db *youdb.DB, aid uint64, tags string) ArticleRelative {
	if len(tags) == 0 {
		return ArticleRelative{}
	}
	getMax := 10
	scanMax := 100

	var aitems []ArticleLi
	var titems []string

	tagsLow := strings.ToLower(tags)

	ctagMap := map[string]struct{}{}

	aidCount := map[uint64]int{}

	for _, tag := range strings.Split(tagsLow, ",") {
		ctagMap[tag] = struct{}{}
		rs := db.Hrscan("tag:"+tag, []byte(""), scanMax)
		if rs.State == "ok" {
			for i := 0; i < len(rs.Data)-1; i += 2 {
				aid2 := youdb.B2i(rs.Data[i])
				if aid2 > 0 && aid2 != aid {
					if _, ok := aidCount[aid2]; ok {
						aidCount[aid2] += 1
					} else {
						aidCount[aid2] = 1
					}
				}
			}
		}
	}

	if len(aidCount) > 0 {

		type Kv struct {
			Key   uint64
			Value int
		}

		var ss []Kv
		for k, v := range aidCount {
			ss = append(ss, Kv{k, v})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		var akeys [][]byte
		j := 0
		for _, kv := range ss {
			akeys = append(akeys, youdb.I2b(kv.Key))
			j++
			if j == getMax {
				break
			}
		}

		rs := db.Hmget("article", akeys)
		if rs.State == "ok" {
			tmpMap := map[string]struct{}{}
			for i := 0; i < len(rs.Data)-1; i += 2 {
				item := ArticleLi{}
				json.Unmarshal(rs.Data[i+1], &item)
				if item.Id == 0 {
					continue
				}
				aitems = append(aitems, item)
				for _, tag := range strings.Split(strings.ToLower(item.Tags), ",") {
					if _, ok := ctagMap[tag]; !ok && len(tag) > 0 {
						tmpMap[tag] = struct{}{}
					}
				}
			}

			for k := range tmpMap {
				titems = append(titems, k)
			}
		}
	}

	return ArticleRelative{
		Articles: aitems,
		Tags:     titems,
	}
}

func UserArticleList(db *youdb.DB, cmd, tb, key string, limit, tz int) ArticlePageInfo {
	var items []ArticleListItem
	var keys [][]byte

	var hasPrev, hasNext bool
	var firstKey, lastKey uint64

	keyStart := youdb.DS2b(key)
	if cmd == "hrscan" {
		rs := db.Hrscan(tb, keyStart, limit)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				keys = append(keys, rs.Data[i])
			}
		}
	} else if cmd == "hscan" {
		rs := db.Hscan(tb, keyStart, limit)
		if rs.State == "ok" {
			for i := len(rs.Data) - 2; i >= 0; i -= 2 {
				keys = append(keys, rs.Data[i])
			}
		}
	}

	if len(keys) > 0 {
		var aitems []ArticleMini
		userMap := map[uint64]UserMini{}
		categoryMap := map[uint64]CategoryMini{}

		rs := db.Hmget("article", keys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := ArticleMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				aitems = append(aitems, item)
				userMap[item.Uid] = UserMini{}
				if item.Ruid > 0 {
					userMap[item.Ruid] = UserMini{}
				}
				categoryMap[item.Cid] = CategoryMini{}
			}
		}

		userKeys := make([][]byte, 0, len(userMap))
		for k := range userMap {
			userKeys = append(userKeys, youdb.I2b(k))
		}
		rs = db.Hmget("user", userKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := UserMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				userMap[item.Id] = item
			}
		}

		categoryKeys := make([][]byte, 0, len(categoryMap))
		for k := range categoryMap {
			categoryKeys = append(categoryKeys, youdb.I2b(k))
		}
		rs = db.Hmget("category", categoryKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := CategoryMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				categoryMap[item.Id] = item
			}
		}

		for _, article := range aitems {
			user := userMap[article.Uid]
			category := categoryMap[article.Cid]
			item := ArticleListItem{
				Id:          article.Id,
				Uid:         article.Uid,
				Name:        user.Name,
				Avatar:      user.Avatar,
				Cid:         article.Cid,
				Cname:       category.Name,
				Ruid:        article.Ruid,
				Title:       article.Title,
				EditTime:    article.EditTime,
				EditTimeFmt: util.TimeFmt(article.EditTime, "2006-01-02 15:04", tz),
				Comments:    article.Comments,
			}
			if article.Ruid > 0 {
				item.Rname = userMap[article.Ruid].Name
			}
			items = append(items, item)
			if firstKey == 0 {
				firstKey = item.Id
			}
			lastKey = item.Id
		}

		rs = db.Hscan(tb, youdb.I2b(firstKey), 1)
		if rs.State == "ok" {
			hasPrev = true
		}
		rs = db.Hrscan(tb, youdb.I2b(lastKey), 1)
		if rs.State == "ok" {
			hasNext = true
		}
	}

	return ArticlePageInfo{
		Items:    items,
		HasPrev:  hasPrev,
		HasNext:  hasNext,
		FirstKey: firstKey,
		LastKey:  lastKey,
	}
}

func ArticleNotificationList(db *youdb.DB, ids string, tz int) ArticlePageInfo {
	var items []ArticleListItem
	var keys [][]byte

	for _, v := range strings.Split(ids, ",") {
		keys = append(keys, youdb.DS2b(v))
	}

	if len(keys) > 0 {
		var aitems []ArticleMini
		userMap := map[uint64]UserMini{}
		categoryMap := map[uint64]CategoryMini{}

		rs := db.Hmget("article", keys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := ArticleMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				aitems = append(aitems, item)
				userMap[item.Uid] = UserMini{}
				if item.Ruid > 0 {
					userMap[item.Ruid] = UserMini{}
				}
				categoryMap[item.Cid] = CategoryMini{}
			}
		}

		userKeys := make([][]byte, 0, len(userMap))
		for k := range userMap {
			userKeys = append(userKeys, youdb.I2b(k))
		}
		rs = db.Hmget("user", userKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := UserMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				userMap[item.Id] = item
			}
		}

		categoryKeys := make([][]byte, 0, len(categoryMap))
		for k := range categoryMap {
			categoryKeys = append(categoryKeys, youdb.I2b(k))
		}
		rs = db.Hmget("category", categoryKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := CategoryMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				categoryMap[item.Id] = item
			}
		}

		for _, article := range aitems {
			user := userMap[article.Uid]
			category := categoryMap[article.Cid]
			item := ArticleListItem{
				Id:          article.Id,
				Uid:         article.Uid,
				Name:        user.Name,
				Avatar:      user.Avatar,
				Cid:         article.Cid,
				Cname:       category.Name,
				Ruid:        article.Ruid,
				Title:       article.Title,
				EditTime:    article.EditTime,
				EditTimeFmt: util.TimeFmt(article.EditTime, "2006-01-02 15:04", tz),
				Comments:    article.Comments,
			}
			if article.Ruid > 0 {
				item.Rname = userMap[article.Ruid].Name
			}
			items = append(items, item)
		}
	}

	return ArticlePageInfo{Items: items}
}

func ArticleSearchList(db *youdb.DB, where, kw string, limit, tz int) ArticlePageInfo {
	var items []ArticleListItem

	var aitems []Article
	userMap := map[uint64]UserMini{}
	categoryMap := map[uint64]CategoryMini{}

	startKey := []byte("")
	for {
		rs := db.Hrscan("article", startKey, limit)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				startKey = rs.Data[i]
				aitem := Article{}
				json.Unmarshal(rs.Data[i+1], &aitem)
				if !aitem.Hidden {
					var getIt bool
					if where == "title" {
						if strings.Index(strings.ToLower(aitem.Title), kw) >= 0 {
							getIt = true
						}
					} else {
						if strings.Index(strings.ToLower(aitem.Content), kw) >= 0 {
							getIt = true
						}
					}
					if getIt {
						aitems = append(aitems, aitem)
						userMap[aitem.Uid] = UserMini{}
						if aitem.RUid > 0 {
							userMap[aitem.RUid] = UserMini{}
						}
						categoryMap[aitem.Cid] = CategoryMini{}
						if len(aitems) == limit {
							break
						}
					}
				}
			}
			if len(aitems) == limit {
				break
			}
		} else {
			break
		}
	}

	if len(aitems) > 0 {
		userKeys := make([][]byte, 0, len(userMap))
		for k := range userMap {
			userKeys = append(userKeys, youdb.I2b(k))
		}
		rs := db.Hmget("user", userKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := UserMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				userMap[item.Id] = item
			}
		}

		categoryKeys := make([][]byte, 0, len(categoryMap))
		for k := range categoryMap {
			categoryKeys = append(categoryKeys, youdb.I2b(k))
		}
		rs = db.Hmget("category", categoryKeys)
		if rs.State == "ok" {
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				item := CategoryMini{}
				json.Unmarshal(rs.Data[i+1], &item)
				categoryMap[item.Id] = item
			}
		}

		for _, article := range aitems {
			user := userMap[article.Uid]
			category := categoryMap[article.Cid]
			item := ArticleListItem{
				Id:          article.Id,
				Uid:         article.Uid,
				Name:        user.Name,
				Avatar:      user.Avatar,
				Cid:         article.Cid,
				Cname:       category.Name,
				Ruid:        article.RUid,
				Title:       article.Title,
				EditTime:    article.EditTime,
				EditTimeFmt: util.TimeFmt(article.EditTime, "2006-01-02 15:04", tz),
				Comments:    article.Comments,
			}
			if article.RUid > 0 {
				item.Rname = userMap[article.RUid].Name
			}
			items = append(items, item)
		}
	}

	return ArticlePageInfo{Items: items}
}

func ArticleFeedList(db *youdb.DB, limit, tz int) []ArticleFeedListItem {
	var items []ArticleFeedListItem
	var keys [][]byte
	keyStart := []byte("")

	for {
		rs := db.Hrscan("article", keyStart, limit)
		if rs.State != "ok" {
			break
		}
		for i := 0; i < (len(rs.Data) - 1); i += 2 {
			keyStart = rs.Data[i]
			keys = append(keys, rs.Data[i])
		}

		if len(keys) > 0 {
			var aitems []Article
			userMap := map[uint64]UserMini{}
			categoryMap := map[uint64]CategoryMini{}

			rs := db.Hmget("article", keys)
			if rs.State == "ok" {
				for i := 0; i < (len(rs.Data) - 1); i += 2 {
					item := Article{}
					json.Unmarshal(rs.Data[i+1], &item)
					if !item.Hidden {
						aitems = append(aitems, item)
						userMap[item.Uid] = UserMini{}
						categoryMap[item.Cid] = CategoryMini{}
					}
				}
			}

			userKeys := make([][]byte, 0, len(userMap))
			for k := range userMap {
				userKeys = append(userKeys, youdb.I2b(k))
			}
			rs = db.Hmget("user", userKeys)
			if rs.State == "ok" {
				for i := 0; i < (len(rs.Data) - 1); i += 2 {
					item := UserMini{}
					json.Unmarshal(rs.Data[i+1], &item)
					userMap[item.Id] = item
				}
			}

			categoryKeys := make([][]byte, 0, len(categoryMap))
			for k := range categoryMap {
				categoryKeys = append(categoryKeys, youdb.I2b(k))
			}
			rs = db.Hmget("category", categoryKeys)
			if rs.State == "ok" {
				for i := 0; i < (len(rs.Data) - 1); i += 2 {
					item := CategoryMini{}
					json.Unmarshal(rs.Data[i+1], &item)
					categoryMap[item.Id] = item
				}
			}

			for _, article := range aitems {
				user := userMap[article.Uid]
				category := categoryMap[article.Cid]
				item := ArticleFeedListItem{
					Id:          article.Id,
					Uid:         article.Uid,
					Name:        user.Name,
					Cname:       html.EscapeString(category.Name),
					Title:       html.EscapeString(article.Title),
					AddTimeFmt:  util.TimeFmt(article.AddTime, time.RFC3339, tz),
					EditTimeFmt: util.TimeFmt(article.EditTime, time.RFC3339, tz),
				}

				contentRune := []rune(article.Content)
				if len(contentRune) > 150 {
					contentRune := []rune(article.Content)
					item.Des = string(contentRune[:150])
				} else {
					item.Des = article.Content
				}
				item.Des = html.EscapeString(item.Des)

				items = append(items, item)
			}

			keys = keys[:0]
			if len(items) >= limit {
				break
			}
		}
	}

	return items
}
