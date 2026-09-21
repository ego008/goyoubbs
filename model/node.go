package model

import (
	"bytes"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ego008/goutils/json"
	"github.com/ego008/mdb"
	"github.com/tidwall/gjson"
	"go.etcd.io/bbolt"

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

func NodeSet(db *mdb.DB, tx *bbolt.Tx, obj Node) (Node, error) {
	if obj.ID == 0 {
		// 添加
		maxId, _ := db.HIncr(tx, CountTb, mdb.S2b(NodeTbName), 1)
		obj.ID = maxId
	}
	jb, err := json.Marshal(obj)
	if err != nil {
		return Node{}, err
	}
	_ = db.HSet(tx, NodeTbName, mdb.I2b(obj.ID), jb)

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

func NodeGetById(db *mdb.DB, tx *bbolt.Tx, nodeId uint64) (obj Node, code int) {
	_ = db.HGetFunc(tx, NodeTbName, mdb.I2b(nodeId), func(val []byte) error {
		err := json.Unmarshal(val, &obj)
		if err == nil {
			code = 1 // 存在时 code 返回1
		}
		return nil
	})

	if code == 1 {
		_ = db.HGetFunc(tx, NodeTopicNumTbName, mdb.I2b(obj.ID), func(val []byte) error {
			obj.TopicNum = mdb.B2i(val)
			return nil
		})
	}
	return // code 返回 0
}

func NodeGetAll(mc *fastcache.Cache, db *mdb.DB, tx *bbolt.Tx) (objLst []Node) {
	mcKey := []byte("NodeGetAll")
	if _, exist := util.ObjCachedGet(mc, mcKey, &objLst, false); exist {
		return
	}

	var keys [][]byte

	_ = db.HScanFunc(tx, NodeTbName, nil, 100, func(key, val []byte) bool {
		obj := Node{}
		err := json.Unmarshal(val, &obj)
		if err != nil {
			return true
		}
		keys = append(keys, bytes.Clone(key))
		objLst = append(objLst, obj)
		return true
	})

	// 文章数
	numMap := map[uint64]uint64{}
	_ = db.HMGetFunc(tx, NodeTopicNumTbName, keys, func(key, val []byte) error {
		numMap[mdb.B2i(key)] = mdb.B2i(val)
		return nil
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
func NodeGetNamesByIds(db *mdb.DB, tx *bbolt.Tx, ids []uint64) map[uint64]string {
	id2name := map[uint64]string{}
	if len(ids) == 0 {
		return id2name
	}
	var idsb [][]byte
	for _, k := range ids {
		if v, ok := nodeNameMap.Load(k); ok {
			id2name[k] = v.(string)
		} else {
			idsb = append(idsb, mdb.I2b(k))
		}
	}
	if len(idsb) > 0 {
		_ = db.HMGetFunc(tx, NodeTbName, idsb, func(key, val []byte) error {
			if len(val) == 0 {
				return nil
			}
			name := gjson.Get(mdb.B2s(val), "Name").String()
			nId := mdb.B2i(key)
			id2name[nId] = name
			nodeNameMap.Store(nId, name)
			return nil
		})
	}
	return id2name
}
