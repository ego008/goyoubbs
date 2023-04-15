package util

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/ego008/sdb"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"log"
	"regexp"
	"strconv"
	"strings"
)

const (
	codeBlockFlag = "```"       // 代码块
	codeBlockTag  = "[qLvDwXa:" // 代码替换随意标签
)

var (
	img2Regexp        = regexp.MustCompile(`(?:\s|^)(https?://[\w./:]+/[\w./]+\.(jpg|jpe|jpeg|gif|png))`)
	mentionRegexp     = regexp.MustCompile(`(?:\s|^)@([^\s]{2,20})\s?`)
	aTagRegexp        = regexp.MustCompile(`(?m)(<a[^<]+?>.*?</a>)`)
	hrefRegexp        = regexp.MustCompile(`href="[^"]+?"`)                                                   // 图片地址被 auto link
	LocalImgRegexp    = regexp.MustCompile(`(?:\s|^)(/static/upload/([a-z0-9]+)\.(jpg|jpe|jpeg|gif|png))\s?`) // 本地上传的图片
	codeBlockRegexp   = regexp.MustCompile("(?s:(`{3} *([^\n]+)?\n(.+?)\n`{3}))")
	langCaptionRegexp = regexp.MustCompile("([^\\s`]+)\\s*(.+)?")
	t4Re              = regexp.MustCompile(`\A( {4}|\t)`)
	t4Re2             = regexp.MustCompile(`^( {4}|\t)`)
	htmlRe            = regexp.MustCompile("<.*?>|&.*?;")
	MdImgRe           = regexp.MustCompile(`(!\[.*]\(.{10,}\))|([\w./:]*/static/upload/([a-z\d-.]+)\.(jpg|jpe|jpeg|gif|png))`)
)

// HasCodeBlock 检测是否有代码块
func HasCodeBlock(s string) (has bool) {
	n := 0
	for {
		i := strings.Index(s, codeBlockFlag)
		if i == -1 {
			return
		}
		n++
		if n == 2 {
			has = true
			return
		}
		s = s[i+len(codeBlockFlag):]
	}
}

// 代码表格
func tableCode(text, lang string) string {
	text = strings.TrimSpace(text)
	var codes []string
	var lines []string
	for i, line := range strings.Split(text, "\n") {
		lines = append(lines, fmt.Sprintf(`<span class="line-number">%d</span>`, i+1))
		codes = append(codes, fmt.Sprintf(`<span class="line">%s</span>`, line))
	}

	return fmt.Sprintf(`
<div class="highlight highlight-%s">
<table><tbody><tr>
<td class="gutter"><pre class="line-numbers">%s</pre></td>
<td class="code"><pre><code class="%s">%s</code></pre></td>
</tr></tbody></table></div>`, lang, strings.Join(lines, "\n"), lang, strings.Join(codes, "\n"))
}

// TrimPreTag 去除 pre 标签
// <pre class="chroma">(..保留的内容..)</pre>
func TrimPreTag(text string) string {
	firstIndex := strings.Index(text, ">")
	lastIndex := strings.LastIndex(text, "</pre>")
	return text[firstIndex+1 : lastIndex]
}

var mdp = goldmark.New(
	goldmark.WithRendererOptions(gmhtml.WithXHTML()),
	goldmark.WithExtensions(
		extension.GFM, extension.Table, extension.Strikethrough, extension.TaskList, extension.Linkify,
	),
)

// 文本格式化
// 代码块解析
// 格式
// ``` [language] [title]
// code snippet
// ```
// 注意首行与末行，前均无空格

func ContentFmt(input string) string {
	//if strings.Contains(input, "&") {
	//	input = htmlStd.UnescapeString(input)
	//}
	// 代码块处理，后端代码高亮
	codeRawMap := map[string]string{} // 代码块
	if HasCodeBlock(input) {
		input = codeBlockRegexp.ReplaceAllStringFunc(input, func(s string) string {
			s = strings.TrimSpace(s) // important
			// 获取并代码头部信息及处理代码高亮 html 代码
			lines := strings.Split(s, "\n")
			// 至少 3 行
			if len(lines) >= 3 {
				caption := ""        // title
				lang := ""           // 语言
				codeInfo := lines[0] // ``` [语言] [title]
				for _, v := range langCaptionRegexp.FindAllStringSubmatch(codeInfo, 1) {
					lang = v[1]
					caption = v[2] // fmt.Sprintf(`<figcaption><span>%s</span></figcaption>`, v[2])
				}
				// 最后一行 ``` 舍弃
				// 纯代码
				codeRaw := strings.Join(lines[1:len(lines)-1], "\n")
				// 替换掉每行多余的空格
				if t4Re.MatchString(codeRaw) {
					codeRaw = t4Re2.ReplaceAllString(codeRaw, "")
				}

				source := []string{`<figure class="code">`}

				langName, hlText := ColorCode(codeRaw, lang)

				//if len(caption) > 0 {
				//	source = append(source, caption)
				//}
				if len(langName) > 0 || len(caption) > 0 {
					source = append(source, `<figcaption><span>`+langName+": "+caption+`</span></figcaption>`)
				}

				source = append(source, tableCode(TrimPreTag(hlText), lang))
				source = append(source, "</figure>")

				codeTag := codeBlockTag + strconv.Itoa(len(codeRawMap)) + "]"
				codeRawMap[codeTag] = strings.Join(source, "\n")

				return codeTag
			}

			return s
		})
	}

	// 兼容直接贴 图片URL
	input = img2Regexp.ReplaceAllString(input, "\n![]($1)\n")
	// 替换本地上传的图片
	input = LocalImgRegexp.ReplaceAllString(input, "\n![]($1)\n")

	if strings.Contains(input, "@") {
		input = mentionRegexp.ReplaceAllString(input, ` @[$1](/name/$1) `)
	}

	// 处理 md
	var md string
	var buf bytes.Buffer
	if err := mdp.Convert(sdb.S2b(input), &buf); err == nil {
		md = buf.String()
	} else {
		log.Println(err)
		md = input
	}

	if strings.Contains(md, "<a ") {
		md = aTagRegexp.ReplaceAllStringFunc(md, func(m string) string {
			// 如果为 mdm 或 /t/* 则去掉 rel="nofollow
			href := hrefRegexp.FindString(m)
			if len(href) > 7 {
				hrefValue := href[6 : len(href)-1]
				if strings.HasPrefix(hrefValue, "#") || strings.HasPrefix(hrefValue, "/t/") || strings.HasPrefix(hrefValue, "/name/") {
					//m = strings.Replace(m, ` rel="nofollow"`, "", 1)
				} else {
					//m = strings.Replace(m, ` rel="nofollow"`, ` rel="nofollow" target="_blank"`, 1)
					m = strings.Replace(m, `">`, `" rel="nofollow" target="_blank">`, 1)
				}

			}
			return m
		})
	}

	// 代码还原
	if len(codeRawMap) > 0 {
		for k := range codeRawMap {
			md = strings.Replace(md, k, codeRawMap[k], 1)
		}
	}

	return md
}

// GetDesc 截取文章摘要 for robots
func GetDesc(input string) (des string) {
	input = htmlRe.ReplaceAllString(input, "")
	limit := 150

	if len(input) <= limit {
		des = input
		return
	}

	firstBrCon := strings.Split(input, "\n")[0]

	if len(firstBrCon) > limit {
		runeCon := []rune(firstBrCon)
		if len(runeCon) <= limit {
			des = firstBrCon
		} else {
			des = string(runeCon[:limit]) // limit 个字
		}
		return
	}
	des = firstBrCon

	return
}

// GetShortCon 截取文章摘要 for robots
func GetShortCon(input string) (des string) {
	input = strings.ReplaceAll(input, "\n", "")
	input = htmlRe.ReplaceAllString(input, "")
	limit := 50

	if len(input) <= limit {
		des = input
		return
	}

	runeCon := []rune(input)
	if len(runeCon) <= limit {
		des = string(runeCon)
	} else {
		des = string(runeCon[:limit]) + "..." // limit 个字
	}

	return
}

// GetMention []notInclude 排除name 列表
func GetMention(input string, notInclude []string) [][]byte {
	notIncludeMap := make(map[string]struct{}, len(notInclude))
	for _, v := range notInclude {
		notIncludeMap[v] = struct{}{}
	}
	sbMap := map[string][]byte{}
	for _, at := range mentionRegexp.FindAllString(input, -1) {
		sb := strings.TrimSpace(at)[1:]
		if _, ok := notIncludeMap[sb]; ok {
			continue
		}
		sbMap[sb] = sdb.S2b(sb)
	}
	if len(sbMap) > 0 {
		sb := make([][]byte, len(sbMap))
		i := 0
		for k := range sbMap {
			sb[i] = sbMap[k]
			i++
		}
		return sb
	}
	return [][]byte{}
}

// ColorCode 代码高亮，
// 把纯代码 import sys
// 转为 <pre class="chroma"><span class="kn">import</span> <span class="nn">sys</span>。。。。</pre>
func ColorCode(source, lang string) (langName, codeHtml string) {
	res := new(bytes.Buffer)

	// Determine lexer.
	lexer := lexers.Get(lang)
	if lexer == nil {
		// 完整的文件比较好识别，代码片段不好识别
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	} else {
		langName = lexer.Config().Name
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Fallback

	// see more https://github.com/alecthomas/chroma#the-html-formatter
	formatter := html.New(html.WithClasses(true))

	iterator, err := lexer.Tokenise(nil, source) // 拿到迭代器
	err = formatter.Format(res, style, iterator)
	if err != nil {
		return "plainText", `<pre class="chroma">` + source + `</pre>` // 原文返回
	}

	return langName, res.String() // 包含 <pre class="chroma">...</pre>
}

func FindAllImgInContent(text string) (imgLst []string) {
	for _, imgSrc := range MdImgRe.FindAllString(text, 3) {
		imgSrc = strings.TrimSpace(imgSrc)
		var imgSrcRaw string
		if strings.Index(imgSrc, "](") > 0 {
			indexL := strings.Index(imgSrc, "(")
			indexR := strings.LastIndex(imgSrc, ")")
			imgSrcRaw = imgSrc[indexL+1 : indexR]
		} else {
			imgSrcRaw = imgSrc
		}
		imgLst = append(imgLst, strings.TrimSpace(imgSrcRaw))
	}
	return
}
