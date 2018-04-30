# goxforum
GoXForum 基于youBBS做了若干调整

```
go get github.com/goxforum/xforum
```

## 轻论坛功能

- 用户：用户名密码登录、微博、QQ 登录
- 用户上传文件存储：本地、七牛、又拍云
- 根据标题自动提取tag 或管理员手工设置tag
- 根据tag 获取相关文章
- 站内搜索标题、内容
- 内容里链接点击计数
- 自动安装HTTPS并自动更新

## 快速使用

即使你没有接触过golang， 按照下面步骤也能快速部署

以linux 64位系统为例，依次输入下面几行命令即可：

下载主程序包、静态文件包
（最新版下载[https://github.com/ego008/goyoubbs/releases](https://github.com/ego008/goyoubbs/releases) 选择适合你系统的包）
```
wget https://github.com/ego008/goyoubbs/releases/download/master/goyoubbs-linux-amd64.zip
wget https://github.com/ego008/goyoubbs/releases/download/master/site.zip
unzip goyoubbs-linux-amd64.zip
unzip site.zip
./goyoubbs
```

如果出现类似下面的提示，说明已正常启动：

```
2017/12/06 16:24:42 MainDomain: http://127.0.0.1:8082
2017/12/06 16:24:42 youdb Connect to mydata.db
2017/12/06 16:24:42 Web server Listen to http://127.0.0.1:8082
```
在浏览器打开上面提示里`Web server Listen to` 的网址 `http://127.0.0.1:8082` 就可以看到网站首页

## 开启HTTPS

为什么要用HTTPS？网站更安全、搜索引擎更喜欢、没有宽带运营商劫持放广告。。。

go youBBS 已经为开启HTTPS 做了最简化处理，提供两种方式开启

### 自动配置（推荐）

- 把你的域名 yourdomain.com 指向你的服务器
- 修改配置文件 `config/config.yaml` 下面三项即可：

```
Main:
    HttpPort: 80
    HttpsOn: true
    Domain: "yourdomain.com"
```

下面两个留空：
```
    TLSCrtFile: ""
    TLSKeyFile: ""
```

保存配置文件，运行主程序 `./goyoubbs`

打开浏览器，输入网址 `https://yourdomain.com`

### 用其它工具

用其它工具生成 `crt`, `key` 文件，在配置里填写这两个文件的路径，如：

```
Main:
    HttpPort: 80
    HttpsOn: true
    Domain: "youbbs.org"
    ...
    TLSCrtFile: "/root/ssl/chained.pem"
    TLSKeyFile: "/root/ssl/domain.key"
```

具体过程请移步 https://www.youbbs.org/t/2169

## 问题、建议、贡献

官方网站 -





