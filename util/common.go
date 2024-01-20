package util

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/cespare/xxhash/v2"
	"sort"
	"strconv"
	"strings"
)

// Md5 生成32位MD5
func Md5(text string) string {
	ctx := md5.New()
	ctx.Write([]byte(text))
	return hex.EncodeToString(ctx.Sum(nil))
}

// Xxhash hash string 2 uint64
func Xxhash(s []byte) uint64 {
	return xxhash.Sum64(s)
}

// GetDomainFromURL 从url 解析出域名
// https://stackoverflow.com/questions/tagged/go?tab=newest&page=2922&pagesize=15
// -> https://stackoverflow.com stackoverflow.com
func GetDomainFromURL(fullURL string) (bsURL, host string) {
	urls := strings.Split(fullURL, "/")
	if len(urls) > 2 {
		host = urls[2]
	} else {
		return
	}
	bsURL = strings.Join(urls[:3], "/")

	return
}

func SliceUniqStr(s, sep string) string {
	if len(sep) == 0 {
		sep = ","
	}
	ss := strings.Split(s, sep)
	seen := make(map[string]struct{}, len(ss))
	j := 0
	for _, v := range ss {
		v = strings.TrimSpace(v)
		if len(v) == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		ss[j] = v
		j++
	}
	sort.Strings(ss[:j])
	return strings.Join(ss[:j], sep)
}

// IpTrimRightDot trim right dot
// ex: 127.66. -> 127.66
// 127.17. -> 127.17.
// 127.17 -> 127.17
func IpTrimRightDot(s string) string {
	if len(s) == 0 {
		return ""
	}
	hasDotSuffix := s[len(s)-1] == 46 // '.' == 46
	if !hasDotSuffix {
		return s
	}
	s = s[:len(s)-1]

	ss := strings.Split(s, ".")
	lastPn, _ := strconv.Atoi(ss[len(ss)-1])
	if lastPn > 25 {
		return s
	}
	return s + "."
}

// TenTo62 将十进制转换为62进制   0-9a-zA-Z 六十二进制
func TenTo62(id uint64) string {
	// 1 -- > 1
	// 10-- > a
	// 61-- > Z
	charset := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var shortUrl []byte
	for {
		var result byte
		number := id % 62
		result = charset[number]
		var tmp []byte
		tmp = append(tmp, result)
		shortUrl = append(tmp, shortUrl...)
		id = id / 62
		if id == 0 {
			break
		}
	}
	return string(shortUrl)
}
