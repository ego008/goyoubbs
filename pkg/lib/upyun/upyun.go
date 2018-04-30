package upyun

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	VERSION = "2.0"

	ED_AUTO    = "v0.api.upyun.com"
	ED_TELECOM = "v1.api.upyun.com"
	ED_CNC     = "v2.api.upyun.com"
	ED_CTT     = "v3.api.upyun.com"
)

type UpYun struct {
	httpClient  *http.Client
	trans       *http.Transport
	bucketName  string
	userName    string
	passWord    string
	apiDomain   string
	contentMd5  string
	fileSecret  string
	respHeaders map[string]string

	TimeOut int
}

/**
 * 初始化 UpYun 存储接口
 * @param bucketName 空间名称
 * @param userName 操作员名称
 * @param passWord 密码
 * return UpYun object
 */
func NewUpYun(bucketName, userName, passWord string) *UpYun {
	u := new(UpYun)
	u.TimeOut = 300
	u.httpClient = &http.Client{}
	u.httpClient.Transport = &http.Transport{Dial: timeoutDialer(u.TimeOut)}
	u.bucketName = bucketName
	u.userName = userName
	u.passWord = StringMd5(passWord)
	u.apiDomain = ED_AUTO
	return u
}

func (u *UpYun) Version() string {
	return VERSION
}

/**
* 切换 API 接口的域名
* @param domain {
        ED_AUTO         自动识别（默认）
        ED_TELECOM      电信,
        ED_CNC          联通,
        ED_CTT          移动
}
* return 无
*/
func (u *UpYun) SetApiDomain(domain string) {
	u.apiDomain = domain
}

/**
 * 设置连接超时时间
 * @param time 秒
 * return 无
 */
func (u *UpYun) SetTimeout(time int) {
	u.TimeOut = time
	u.httpClient.Transport = &http.Transport{Dial: timeoutDialer(u.TimeOut)}
}

/**
 * 设置待上传文件的 Content-MD5 值（如又拍云服务端收到的文件MD5值与用户设置的不一致，
 * 将回报 406 Not Acceptable 错误）
 * @param str （文件 MD5 校验码）
 * return 无
 */
func (u *UpYun) SetContentMD5(str string) {
	u.contentMd5 = str
}

/**
 * 连接签名方法
 * @param method 请求方式 {GET, POST, PUT, DELETE}
 * return 签名字符串
 */
func (u *UpYun) sign(method, uri, date string, length int64) string {
	var bufSign bytes.Buffer
	bufSign.WriteString(method)
	bufSign.WriteString("&")
	bufSign.WriteString(uri)
	bufSign.WriteString("&")
	bufSign.WriteString(date)
	bufSign.WriteString("&")
	bufSign.WriteString(strconv.FormatInt(length, 10))
	bufSign.WriteString("&")
	bufSign.WriteString(u.passWord)

	var buf bytes.Buffer
	buf.WriteString("UpYun ")
	buf.WriteString(u.userName)
	buf.WriteString(":")
	buf.WriteString(StringMd5(bufSign.String()))
	return buf.String()
}

/**
 * 连接处理逻辑
 * @param method 请求方式 {GET, POST, PUT, DELETE}
 * @param uri 请求地址
 * @param inFile 如果是POST上传文件，传递文件IO数据流
 * @param outFile 如果是GET下载文件，可传递文件IO数据流，这种情况函数也返回""
 * @param data byte数据
 * return 请求返回字符串，失败返回""
 */
func (u *UpYun) httpAction(method, uri string, headers map[string]string, inFile, outFile *os.File, data []byte) (string, error) {
	uri = "/" + u.bucketName + "/" + uri
	url := "http://" + u.apiDomain + uri
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	var contentLength int64 = 0
	if method == "PUT" || method == "POST" {
		method = "POST"
		if inFile != nil {
			contentLength = FileSize(inFile)
			req.Body = inFile
		}
		if len(data) > 0 {
			contentLength = int64(len(data))
			req.Body = ioutil.NopCloser(bytes.NewReader(data))
		}
		req.Header.Add("Content-Length", strconv.FormatInt(contentLength, 10))
		if u.contentMd5 != "" {
			req.Header.Add("Content-MD5", u.contentMd5)
			u.contentMd5 = ""
		}
		if u.fileSecret != "" {
			req.Header.Add("Content-Secret", u.fileSecret)
			u.fileSecret = ""
		}
	} else if method == "HEAD" {
		req.Body = nil
	}

	req.Method = method

	date := time.Now().UTC().Format(time.RFC1123)
	req.Header.Add("Date", date)

	signStr := u.sign(method, uri, date, contentLength)
	req.Header.Add("Authorization", signStr)

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rc := resp.StatusCode
	if rc != 200 {
		return "", errors.New(fmt.Sprintf("%d: %s", rc, resp.Status))
	}

	u.respHeaders = make(map[string]string)
	for k, v := range resp.Header {
		if strings.Contains(k, "x-upyun") {
			u.respHeaders[k] = v[0]
		}
	}

	if method == "GET" && outFile != nil {
		_, err := io.Copy(outFile, resp.Body)
		if err != nil {
			return "", errors.New("copy to output file error: " + err.Error())
		}
	}

	buf, err := ioutil.ReadAll(resp.Body)

	return string(buf), err
}

/**
 * 获取总体空间的占用信息
 * return 空间占用量，失败返回0.0
 */
func (u *UpYun) GetBucketUsage() (float64, error) {
	return u.GetFolderUsage("/")
}

/**
 * 获取某个子目录的占用信息
 * @param $path 目标路径
 * return 空间占用量和error，失败空间占用量返回0.0
 */
func (u *UpYun) GetFolderUsage(path string) (float64, error) {
	r, err := u.httpAction("GET", path+"?usage", nil, nil, nil, nil)
	if err != nil {
		return 0.0, err
	}
	v, _ := strconv.ParseFloat(r, 64)
	return v, nil
}

/**
 * 设置待上传文件的 访问密钥（注意：仅支持图片空！，设置密钥后，无法根据原文件URL直接访问，需带 URL 后面加上 （缩略图间隔标志符+密钥） 进行访问）
 * 如缩略图间隔标志符为 ! ，密钥为 bac，上传文件路径为 /folder/test.jpg ，那么该图片的对外访问地址为： http://空间域名/folder/test.jpg!bac
 * @param $str （文件 MD5 校验码）
 * return null
 */
func (u *UpYun) SetFileSecret(str string) {
	u.fileSecret = str
}

/**
 * 上传文件
 * @param remoteFilePath 文件路径（包含文件名）
 * @param localFilePath 文件IO数据流
 * @param autoMkdir 是否自动创建父级目录(最深10级目录)
 * @param data byte数据
 * return error
 */
func (u *UpYun) WriteFile(remoteFilePath, localFilePath string, autoMkdir bool, data []byte) error {
	var headers map[string]string
	if autoMkdir {
		headers = make(map[string]string)
		headers["Mkdir"] = "true"
	}
	var err error
	var inFile *os.File
	if len(localFilePath) > 0 {
		inFile, err = os.Open(localFilePath)
		if err != nil {
			return err
		}
	} else {
		if len(data) == 0 {
			return errors.New("data is empty")
		}
	}
	_, err = u.httpAction("PUT", remoteFilePath, headers, inFile, nil, data)
	return err
}

/**
 * 获取上传文件后的信息（仅图片空间有返回数据）
 * @param key 信息字段名（x-upyun-width、x-upyun-height、x-upyun-frames、x-upyun-file-type）
 * return string or ""
 */
func (u *UpYun) GetWritedFileInfo(key string) string {
	if u.respHeaders == nil {
		return ""
	}
	return u.respHeaders[strings.ToLower(key)]
}

/**
 * 读取文件
 * @param file 文件路径（包含文件名）
 * @param outFile 可传递文件IO数据流（结果返回true or false）
 * return error
 */
func (u *UpYun) ReadFile(remoteFilePath, localFilePath string) error {
	var outFile *os.File
	outFile, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	_, err = u.httpAction("GET", remoteFilePath, nil, nil, outFile, nil)
	return err
}

/**
 * 获取文件信息
 * @param file 文件路径（包含文件名）
 * return map("type": file | folder, "size": file size, "date": unix time) 或 nil
 */
func (u *UpYun) GetFileInfo(file string) (map[string]string, error) {
	_, err := u.httpAction("HEAD", file, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if u.respHeaders == nil {
		return nil, nil
	}
	m := make(map[string]string)
	if v, ok := u.respHeaders["x-upyun-file-type"]; ok {
		m["type"] = v
	}
	if v, ok := u.respHeaders["x-upyun-file-size"]; ok {
		m["size"] = v
	}
	if v, ok := u.respHeaders["x-upyun-file-date"]; ok {
		m["date"] = v
	}
	return m, nil
}

type DirInfo struct {
	Name string
	Type string
	Size int64
	Time int64
}

/**
 * 读取目录列表
 * @param path 目录路径
 * return DirInfo数组 或 nil
 */
func (u *UpYun) ReadDir(path string) ([]*DirInfo, error) {
	r, err := u.httpAction("GET", path, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	if r == "" {
		return nil, nil
	}

	dirs := make([]*DirInfo, 0, 8)
	rs := strings.Split(r, "\n")
	for i := 0; i < len(rs); i++ {
		ri := strings.TrimSpace(rs[i])
		rid := strings.Split(ri, "\t")
		d := new(DirInfo)
		d.Name = rid[0]
		if len(rid) > 3 && rid[3] != "" {
			if rid[1] == "N" {
				d.Type = "file"
			} else {
				d.Type = "folder"
			}
			d.Time, _ = strconv.ParseInt(rid[3], 10, 64)
		}
		if len(rid) > 2 {
			d.Size, _ = strconv.ParseInt(rid[2], 10, 64)
		}
		dirs = append(dirs, d)
	}
	return dirs, nil
}

/**
 * 删除文件
 * @param file 文件路径（包含文件名）
 * return error
 */
func (u *UpYun) DeleteFile(file string) error {
	_, err := u.httpAction("DELETE", file, nil, nil, nil, nil)
	return err
}

/**
 * 创建目录
 * @param path 目录路径
 * @param auto_mkdir=false 是否自动创建父级目录
 * return error
 */
func (u *UpYun) MkDir(path string, autoMkdir bool) error {
	headers := make(map[string]string)
	headers["Folder"] = "true"
	if autoMkdir {
		headers["Mkdir"] = "true"
	}
	_, err := u.httpAction("PUT", path, headers, nil, nil, nil)
	return err
}

/**
 * 删除目录
 * @param path 目录路径
 * return error
 */
func (u *UpYun) RmDir(dir string) error {
	_, err := u.httpAction("DELETE", dir, nil, nil, nil, nil)
	return err
}

func FileSize(f *os.File) int64 {
	if f == nil {
		return 0
	}
	if fi, err := f.Stat(); err == nil {
		return fi.Size()
	}
	return 0
}

func StringMd5(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func FileMd5(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", nil
	}
	defer f.Close()

	h := md5.New()
	io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func timeoutDialer(timeout int) func(string, string) (net.Conn, error) {
	return func(netw, addr string) (c net.Conn, err error) {
		delta := time.Duration(timeout) * time.Second
		c, err = net.DialTimeout(netw, addr, delta)
		if err != nil {
			return nil, err
		}
		c.SetDeadline(time.Now().Add(delta))
		return c, nil
	}
}
