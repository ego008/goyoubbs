package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

func SliceUniqInt(s []int) []int {
	if len(s) == 0 {
		return s
	}
	seen := make(map[int]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func SliceUniqStr(s []string) []string {
	if len(s) == 0 {
		return s
	}
	seen := make(map[string]struct{}, len(s))
	j := 0
	for _, v := range s {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		s[j] = v
		j++
	}
	return s[:j]
}

func HashFileMD5(filePath string) (string, error) {
	var returnMD5String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}
	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}

func CheckTags(tags string) string {
	limit := 5
	seen := map[string]struct{}{}
	tmpTags := strings.Replace(tags, " ", ",", -1)
	tmpTags = strings.Replace(tmpTags, "，", ",", -1)
	tagList := make([]string, limit)
	j := 0
	for _, tag := range strings.Split(tmpTags, ",") {
		if len(tag) > 1 && len(tag) < 25 {
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			tagList[j] = tag
			j++
			if j == limit {
				break
			}
		}
	}
	return strings.Join(tagList[:j], ",")
}
