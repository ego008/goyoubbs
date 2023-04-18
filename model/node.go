package model

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"github.com/tidwall/gjson"
	"goyoubbs/util"
	"sync"
)

const (
	NodeTbName         = "node"
	NodeTopicNumTbName = "node_topic_num"
)

var nodeNameMap = sync.Map{} // 缓存

// Node 节点
type Node struct {
	ID       uint64
	Name     string
	About    string // 描述
	Score    int    // 显示排序
	TopicNum uint64 //包含文章数
}

func NodeSet(db *sdb.DB, obj Node) (Node, error) {
	if obj.ID == 0 {
		// 添加
		maxId, _ := db.Hincr(CountTb, sdb.S2b(NodeTbName), 1)
		obj.ID = maxId
	}
	jb, err := json.Marshal(obj)
	if err != nil {
		return Node{}, err
	}
	_ = db.Hset(NodeTbName, sdb.I2b(obj.ID), jb)

	// update nodeNameMap
	if v, ok := nodeNameMap.Load(obj.ID); ok {
		if obj.Name != v.(string) {
			nodeNameMap.Store(obj.ID, obj.Name)
		}
	} else {
		nodeNameMap.Store(obj.ID, obj.Name)
	}

	return obj, err
}

func NodeGetById(db *sdb.DB, nodeId uint64) (obj Node, code int) {
	if rs := db.Hget(NodeTbName, sdb.I2b(nodeId)); rs.OK() {
		err := json.Unmarshal(rs.Bytes(), &obj)
		if err == nil {
			code = 1 // 存在时 code 返回1
		}
	}
	if code == 1 {
		obj.TopicNum = db.HgetInt(NodeTopicNumTbName, sdb.I2b(obj.ID))
	}
	return // code 返回 0
}

func NodeGetAll(mc *fastcache.Cache, db *sdb.DB) (objLst []Node) {
	mcKey := []byte("NodeGetAll")
	if _, exist := util.ObjCachedGet(mc, mcKey, &objLst, false); exist {
		return
	}

	var keys [][]byte
	db.Hscan(NodeTbName, nil, 100).KvEach(func(key, value sdb.BS) {
		obj := Node{}
		err := json.Unmarshal(value, &obj)
		if err != nil {
			return
		}
		keys = append(keys, key)
		objLst = append(objLst, obj)
	})

	// 文章数
	numMap := map[uint64]uint64{}
	db.Hmget(NodeTopicNumTbName, keys).KvEach(func(key, value sdb.BS) {
		numMap[sdb.B2i(key)] = sdb.B2i(value)
	})

	for i, v := range objLst {
		v.TopicNum, _ = numMap[v.ID]
		objLst[i] = v
	}

	// set to mc
	util.ObjCachedSet(mc, mcKey, objLst)

	return
}

// NodeGetNamesByIds 根据 ids 取 name ，返回id:name 的map
// 只解析 Name 字段，性能提高一丁点
func NodeGetNamesByIds(db *sdb.DB, ids []uint64) map[uint64]string {
	id2name := map[uint64]string{}
	if len(ids) == 0 {
		return id2name
	}
	var idsb [][]byte
	for _, k := range ids {
		if v, ok := nodeNameMap.Load(k); ok {
			id2name[k] = v.(string)
		} else {
			idsb = append(idsb, sdb.I2b(k))
		}
	}
	if len(idsb) > 0 {
		db.Hmget(NodeTbName, idsb).KvEach(func(key, value sdb.BS) {
			name := gjson.Get(value.String(), "Name").String()
			nId := sdb.B2i(key.Bytes())
			id2name[nId] = name
			nodeNameMap.Store(nId, name)
		})
	}
	return id2name
}
