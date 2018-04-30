package storage

import (
	"context"
	"fmt"
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/x/rpc.v7"
)

// OperationManager 提供了数据处理相关的方法
type OperationManager struct {
	client *rpc.Client
	mac    *qbox.Mac
	cfg    *Config
}

// NewOperationManager 用来构建一个新的数据处理对象
func NewOperationManager(mac *qbox.Mac, cfg *Config) *OperationManager {
	if cfg == nil {
		cfg = &Config{}
	}

	return &OperationManager{
		client: NewClient(mac, nil),
		mac:    mac,
		cfg:    cfg,
	}
}

// NewOperationManager 用来构建一个新的数据处理对象
func NewOperationManagerEx(mac *qbox.Mac, cfg *Config, client *rpc.Client) *OperationManager {
	if cfg == nil {
		cfg = &Config{}
	}

	if client == nil {
		client = NewClient(mac, nil)
	}

	return &OperationManager{
		client: client,
		mac:    mac,
		cfg:    cfg,
	}
}

// PfopRet 为数据处理请求的回复内容
type PfopRet struct {
	PersistentID string `json:"persistentId,omitempty"`
}

// PrefopRet 为数据处理请求的状态查询回复内容
type PrefopRet struct {
	ID          string `json:"id"`
	Code        int    `json:"code"`
	Desc        string `json:"desc"`
	InputBucket string `json:"inputBucket,omitempty"`
	InputKey    string `json:"inputKey,omitempty"`
	Pipeline    string `json:"pipeline,omitempty"`
	Reqid       string `json:"reqid,omitempty"`
	Items       []FopResult
}

func (r *PrefopRet) String() string {
	strData := fmt.Sprintf("Id: %s\r\nCode: %d\r\nDesc: %s\r\n", r.ID, r.Code, r.Desc)
	if r.InputBucket != "" {
		strData += fmt.Sprintln(fmt.Sprintf("InputBucket: %s", r.InputBucket))
	}
	if r.InputKey != "" {
		strData += fmt.Sprintln(fmt.Sprintf("InputKey: %s", r.InputKey))
	}
	if r.Pipeline != "" {
		strData += fmt.Sprintln(fmt.Sprintf("Pipeline: %s", r.Pipeline))
	}
	if r.Reqid != "" {
		strData += fmt.Sprintln(fmt.Sprintf("Reqid: %s", r.Reqid))
	}

	strData = fmt.Sprintln(strData)
	for _, item := range r.Items {
		strData += fmt.Sprintf("\tCmd:\t%s\r\n\tCode:\t%d\r\n\tDesc:\t%s\r\n", item.Cmd, item.Code, item.Desc)
		if item.Error != "" {
			strData += fmt.Sprintf("\tError:\t%s\r\n", item.Error)
		} else {
			if item.Hash != "" {
				strData += fmt.Sprintf("\tHash:\t%s\r\n", item.Hash)
			}
			if item.Key != "" {
				strData += fmt.Sprintf("\tKey:\t%s\r\n", item.Key)
			}
			if item.Keys != nil {
				if len(item.Keys) > 0 {
					strData += "\tKeys: {\r\n"
					for _, key := range item.Keys {
						strData += fmt.Sprintf("\t\t%s\r\n", key)
					}
					strData += "\t}\r\n"
				}
			}
		}
		strData += "\r\n"
	}
	return strData
}

// FopResult 云处理操作列表，包含每个云处理操作的状态信息
type FopResult struct {
	Cmd   string   `json:"cmd"`
	Code  int      `json:"code"`
	Desc  string   `json:"desc"`
	Error string   `json:"error,omitempty"`
	Hash  string   `json:"hash,omitempty"`
	Key   string   `json:"key,omitempty"`
	Keys  []string `json:"keys,omitempty"`
}

// Pfop 持久化数据处理
//
//	bucket		资源空间
//	key   		源资源名
//	fops		云处理操作列表，
//	notifyURL	处理结果通知接收URL
//	pipeline	多媒体处理队列名称
//	force		强制执行数据处理
//
func (m *OperationManager) Pfop(bucket, key, fops, pipeline, notifyURL string,
	force bool) (persistentID string, err error) {
	pfopParams := map[string][]string{
		"bucket": []string{bucket},
		"key":    []string{key},
		"fops":   []string{fops},
	}

	if pipeline != "" {
		pfopParams["pipeline"] = []string{pipeline}
	}

	if notifyURL != "" {
		pfopParams["notifyURL"] = []string{notifyURL}
	}

	if force {
		pfopParams["force"] = []string{"1"}
	}
	var ret PfopRet
	ctx := context.TODO()
	reqHost, reqErr := m.apiHost(bucket)
	if reqErr != nil {
		err = reqErr
		return
	}
	reqURL := fmt.Sprintf("%s/pfop/", reqHost)
	err = m.client.CallWithForm(ctx, &ret, "POST", reqURL, pfopParams)
	if err != nil {
		return
	}

	persistentID = ret.PersistentID
	return
}

// Prefop 持久化处理状态查询
func (m *OperationManager) Prefop(persistentID string) (ret PrefopRet, err error) {
	ctx := context.TODO()
	reqHost := m.prefopApiHost(persistentID)
	reqURL := fmt.Sprintf("%s/status/get/prefop?id=%s", reqHost, persistentID)
	err = m.client.Call(ctx, &ret, "GET", reqURL)
	return
}

func (m *OperationManager) apiHost(bucket string) (apiHost string, err error) {
	var zone *Zone
	if m.cfg.Zone != nil {
		zone = m.cfg.Zone
	} else {
		if v, zoneErr := GetZone(m.mac.AccessKey, bucket); zoneErr != nil {
			err = zoneErr
			return
		} else {
			zone = v
		}
	}

	scheme := "http://"
	if m.cfg.UseHTTPS {
		scheme = "https://"
	}
	apiHost = fmt.Sprintf("%s%s", scheme, zone.ApiHost)

	return
}

func (m *OperationManager) prefopApiHost(persistentID string) (apiHost string) {
	apiHost = "api.qiniu.com"
	if m.cfg.Zone != nil {
		apiHost = m.cfg.Zone.ApiHost
	}

	if m.cfg.UseHTTPS {
		apiHost = fmt.Sprintf("https://%s", apiHost)
	} else {
		apiHost = fmt.Sprintf("http://%s", apiHost)
	}

	return
}
