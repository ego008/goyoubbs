package util

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/cespare/xxhash/v2"
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
