package cronjob

import (
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"goyoubbs/model"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

var whiteDnsKws = []string{".petalsearch.com", ".search.msn.com", ".applebot.apple.com",
	".googleusercontent.com", ".googlebot.com", ".google.com", ".yandex.com"}

func spiderIpCheck(db *sdb.DB) {
	v := model.IpQue.Dequeue()
	if v == nil {
		return
	}
	uip := v.(string)
	uipByte := sdb.S2b(uip)

	obj := model.IpInfo{}
	if rs := db.Hget(model.TbnIpInfo, uipByte); rs.OK() {
		_ = json.Unmarshal(rs.Bytes(), &obj)
	}

	tm := time.Now().UTC().Unix()
	if obj.AddTime == 0 {
		// add
		obj.Ip = uip
		obj.AddTime = tm
	}

	if (tm - obj.UpTime) < 864000 {
		// 3600*24*10
		return
	}

	obj.UpTime = tm

	names, err := net.LookupAddr(obj.Ip)
	if err != nil {
		log.Printf("LookupAddr %s, %v \n ", uip, err)
		return
	}
	save2db := false
	for _, name := range names {
		var addrLst []string
		addrLst, err = net.LookupHost(name)
		if err != nil {
			log.Printf("LookupHost ip %s, %v \n", obj.Ip, err)
			return
		}
		for _, addr := range addrLst {
			//log.Println(addr)
			if strings.Compare(addr, obj.Ip) == 0 {
				save2db = true
			}
		}
	}

	if !save2db {
		return
	}

	obj.Names = strings.Join(names, ",")
	// auto save
	isGood := false
	for _, kw := range whiteDnsKws {
		if strings.Contains(obj.Names, kw) {
			isGood = true
			break
		}
	}

	var prefix string
	if isGood {
		ss := strings.Split(obj.Ip, ".")
		prefix = strings.Join(ss[:2], ".") // get two part
		lastPn, _ := strconv.Atoi(ss[1])
		if lastPn < 26 {
			prefix += "."
		}
	}

	wkeyB := sdb.S2b(model.SettingKeyAllowIp)
	addPrefixToWhite := true
	jb, _ := json.Marshal(obj)

	if len(prefix) > 0 {
		// auto add prefix to AllowIpPrefixLst
		ips := strings.Split(db.Hget(model.TbnSetting, wkeyB).String(), ",")
		for _, ip := range ips {
			if strings.HasPrefix(ip, prefix) {
				addPrefixToWhite = false
				break
			}
		}
		if addPrefixToWhite {
			if len(ips) == 1 && ips[0] == "" {
				ips[0] = prefix
			} else {
				ips = append(ips, prefix)
			}
			sort.Strings(ips)
			_ = db.Hset(model.TbnSetting, wkeyB, sdb.S2b(strings.Join(ips, ",")))
			// update
			// model.AllowIpPrefixLst.Copy(ips[:])
			log.Println("auto add prefix to white list", prefix)
		}
	}
	_ = db.Hset(model.TbnIpInfo, uipByte, jb)

	// update
	if addPrefixToWhite && len(prefix) > 0 {
		model.UpdateAllowIpPrefix(db)
		log.Println("fresh ip white list", prefix)
	}

	log.Println("save ok", uip, obj.Names)
}
