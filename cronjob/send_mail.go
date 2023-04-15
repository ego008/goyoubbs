package cronjob

import (
	"crypto/tls"
	"fmt"
	"github.com/ego008/goutils/json"
	"github.com/ego008/sdb"
	"goyoubbs/model"
	"log"
	"net"
	"net/smtp"
	"strings"
)

func sendMail(db *sdb.DB, scf *model.SiteConf) {

	rs := db.Hscan("mail_queue", nil, 1)
	if !rs.OK() {
		return
	}
	queueKey := sdb.B2i(rs.Data[0])
	queueBody := rs.Data[1]

	// subject body
	obj := model.EmailInfo{}
	err := json.Unmarshal(queueBody, &obj)
	if err != nil {
		_ = db.Hdel("mail_queue", sdb.I2b(queueKey)) // 删除
		return
	}
	subject := obj.Subject
	body := obj.Body

	host := scf.SmtpHost       //
	port := scf.SmtpPort       //
	email := scf.SmtpEmail     //
	pwd := scf.SmtpPassword    // 这里填你的授权码
	toEmail := scf.SendToEmail // 目标地址

	header := make(map[string]string)

	fromName := strings.Split(email, "@")[0]
	header["From"] = fromName + "<" + email + ">"
	header["To"] = toEmail
	header["Subject"] = subject
	header["Content-Type"] = "text/html;chartset=UTF-8"

	//body := `当您遇到客户端无法收发信 https://www.youbbs.org/ ，报错无网络连接这是一封golang 发来的邮件，云南人证核验访客机实名认证登记稳定,为了<a href="https://www.youbbs.org/">保障您</a>客户端使用的顺畅，建议您将客户端自动收取的间隔时间设置长一些；如果您的邮箱多人同时使用`

	message := ""

	for k, v := range header {
		message += fmt.Sprintf("%s:%s\r\n", k, v)
	}

	message += "\r\n" + body

	auth := smtp.PlainAuth(
		"",
		email,
		pwd,
		host,
	)

	err = SendMailUsingTLS(
		fmt.Sprintf("%s:%d", host, port),
		auth,
		email,
		[]string{toEmail},
		[]byte(message),
	)

	if err != nil {
		log.Println(err)
		return
	}

	_ = db.Hdel("mail_queue", sdb.I2b(queueKey)) // 删除

}

// Dial return a smtp client
func Dial(addr string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", addr, nil)
	if err != nil {
		log.Println("Dialing Error:", err)
		return nil, err
	}
	//分解主机端口字符串
	host, _, _ := net.SplitHostPort(addr)
	return smtp.NewClient(conn, host)
}

// SendMailUsingTLS 参考net/smtp的func SendMail()
// 使用net.Dial连接tls(ssl)端口时,smtp.NewClient()会卡住且不提示err
// len(to)>1时,to[1]开始提示是密送
func SendMailUsingTLS(addr string, auth smtp.Auth, from string,
	to []string, msg []byte) (err error) {

	//create smtp client
	c, err := Dial(addr)
	if err != nil {
		log.Println("Create smpt client error:", err)
		return err
	}
	defer func() {
		_ = c.Close()
	}()

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(auth); err != nil {
				log.Println("Error during AUTH", err)
				return err
			}
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}
	return c.Quit()
}
