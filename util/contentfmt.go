package util

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/ego008/youdb"
	"regexp"
	"strconv"
	"strings"
)

var (
	codeRegexp = regexp.MustCompile("(?s:```(.+?)```)")
	// #imgRegexp     = regexp.MustCompile(`(https?://[\w./:]+/[\w./]+\.(jpg|jpe|jpeg|gif|png))`)
	imgRegexp     = regexp.MustCompile(`(https?://.+\.(jpg|jpe|jpeg|gif|png))`)
	gistRegexp    = regexp.MustCompile(`(https?://gist\.github\.com/([a-zA-Z0-9-]+/)?[\d]+)`)
	mentionRegexp = regexp.MustCompile(`\B@([a-zA-Z0-9\p{Han}]{1,32})\s?`)
	urlRegexp     = regexp.MustCompile(`([^;"='>])(https?://[^\s<]+[^\s<.)])`)
	nlineRegexp   = regexp.MustCompile(`\s{2,}`)
	youku1Regexp  = regexp.MustCompile(`https?://player\.youku\.com/player\.php/sid/([a-zA-Z0-9=]+)/v\.swf`)
	youku2Regexp  = regexp.MustCompile(`https?://v\.youku\.com/v_show/id_([a-zA-Z0-9=]+)(/|\.html?)?`)
)

// 文本格式化
func ContentFmt(db *youdb.DB, input string) string {
	if strings.Index(input, "```") >= 0 {
		sepNum := strings.Count(input, "```")
		if sepNum < 2 {
			return input
		}
		codeMap := map[string]string{}
		input = codeRegexp.ReplaceAllStringFunc(input, func(m string) string {
			m = strings.Trim(m, "```")
			m = strings.Trim(m, "\n")
			//m = strings.TrimSpace(m)
			m = strings.Replace(m, "&", "&amp;", -1)
			m = strings.Replace(m, "<", "&lt;", -1)
			m = strings.Replace(m, ">", "&gt;", -1)

			codeTag := "[mspctag_" + strconv.FormatInt(int64(len(codeMap)+1), 10) + "]"
			codeMap[codeTag] = "<pre><code>" + m + "</code></pre>"
			return codeTag
		})

		input = ContentRich(db, input)
		// replace tmp code tag
		if len(codeMap) > 0 {
			for k, v := range codeMap {
				input = strings.Replace(input, k, v, -1)
			}
		}
		//
		input = strings.Replace(input, "<p><pre>", "<pre>", -1)
		input = strings.Replace(input, "</pre></p>", "</pre>", -1)
		return input
	}
	return ContentRich(db, input)
}

type urlInfo struct {
	Href  string
	Click string
}

func ContentRich(db *youdb.DB, input string) string {
	input = strings.TrimSpace(input)
	input = " " + input // fix Has url Prefix
	input = strings.Replace(input, "<", "&lt;", -1)
	input = strings.Replace(input, ">", "&gt;", -1)
	input = imgRegexp.ReplaceAllString(input, `<img src="$1" />`)

	// video
	// youku
	if strings.Index(input, "player.youku.com") >= 0 {
		input = youku1Regexp.ReplaceAllString(input, `<embed src="https://player.youku.com/player.php/sid/$1/v.swf" quality="high" width="590" height="492" align="middle" allowScriptAccess="sameDomain" type="application/x-shockwave-flash"></embed>`)
	}
	if strings.Index(input, "v.youku.com") >= 0 {
		input = youku2Regexp.ReplaceAllString(input, `<embed src="https://player.youku.com/player.php/sid/$1/v.swf" quality="high" width="590" height="492" align="middle" allowScriptAccess="sameDomain" type="application/x-shockwave-flash"></embed>`)
	}

	if strings.Index(input, "://gist") >= 0 {
		input = gistRegexp.ReplaceAllString(input, `<script src="$1.js"></script>`)
	}
	if strings.Index(input, "@") >= 0 {
		input = mentionRegexp.ReplaceAllString(input, ` @<a href="/member/$1">$1</a> `)
	}
	if strings.Index(input, "http") >= 0 {
		//input = urlRegexp.ReplaceAllString(input, `$1<a href="$2">$2</a>`)
		urlMd5Map := map[string]urlInfo{}
		var keys [][]byte
		input = urlRegexp.ReplaceAllStringFunc(input, func(m string) string {
			n := strings.Index(m, "http")
			url := strings.Replace(strings.TrimSpace(m[n:]), "&amp;", "&", -1)
			hash := md5.Sum([]byte(url))
			urlMd5 := hex.EncodeToString(hash[:])
			urlMd5Map[urlMd5] = urlInfo{Href: url}
			keys = append(keys, []byte(urlMd5))
			return m[:n] + "[" + urlMd5 + "]"
		})

		if len(urlMd5Map) > 0 {
			rs := db.Hmget("url_md5_click", keys)
			for i := 0; i < (len(rs.Data) - 1); i += 2 {
				key := rs.Data[i].String()
				tmp := urlMd5Map[key]
				tmp.Click = youdb.B2ds(rs.Data[i+1])
				urlMd5Map[key] = tmp
			}
			for k, v := range urlMd5Map {
				var aTag string
				if len(v.Click) > 0 {
					aTag = `<a href="` + v.Href + `" target="_blank">` + v.Href + `</a> <span class="badge-notification clicks" title="` + v.Click + ` 次点击">` + v.Click + `</span>`
				} else {
					aTag = `<a href="` + v.Href + `" target="_blank">` + v.Href + `</a>`
				}
				input = strings.Replace(input, "["+k+"]", aTag, -1)
			}
		}
	}

	input = strings.Replace(input, "\r\n", "\n", -1)
	input = strings.Replace(input, "\r", "\n", -1)

	input = nlineRegexp.ReplaceAllString(input, "</p><p>")
	input = strings.Replace(input, "\n", "<br>", -1)

	input = "<p>" + input + "</p>"
	input = strings.Replace(input, "<p></p>", "", -1)

	return input
}

func GetMention(input string, notInclude []string) []string {
	notIncludeMap := make(map[string]struct{}, len(notInclude))
	for _, v := range notInclude {
		notIncludeMap[v] = struct{}{}
	}
	sbMap := map[string]struct{}{}
	for _, at := range mentionRegexp.FindAllString(input, -1) {
		sb := strings.TrimSpace(at)[1:]
		if _, ok := notIncludeMap[sb]; ok {
			continue
		}
		sbMap[sb] = struct{}{}
	}
	if len(sbMap) > 0 {
		sb := make([]string, len(sbMap))
		i := 0
		for k := range sbMap {
			sb[i] = k
			i++
		}
		return sb
	}
	return []string{}
}
