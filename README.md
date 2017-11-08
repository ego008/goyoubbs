# goyoubbs

golang 实现的youBBS https://www.youbbs.org

## 轻论坛功能

- 用户：用户名密码登录、微博、QQ 登录
- 用户上传文件存储：本地、七牛、又拍云
- 根据标题自动提取tag 或管理员手工设置tag
- 根据tag 获取相关文章
- 站内搜索标题、内容
- 内容里链接点击计数

## 简单安装

如果你没有接触过golang 按照下面步骤也能快速部署

1）下载并解压 https://github.com/ego008/goyoubbs/archive/master.zip

2）本地运行，切换到上面解压的文件夹，运行网站主程序

默认有下面几个包

- goyoubbs_linux -- linux 64bit 系统
- goyoubbs.exe   -- windows 系统
- goyoubbs_mac   -- mac 系统

如果你的电脑是windows 系统只需双击 `goyoubbs.exe`

在浏览器打开 `http://127.0.0.1:8082` 即可看到网站内容

3）部署到服务器

准备网站必须的文件，新建一个文件夹保存部署所需的文件，如mysite

把下面的文件夹拷贝到mysite：

- config
- databackup
- static
- view

根据你的服务器环境拷贝对应的主程序包到mysite

- goyoubbs_linux -- linux 64bit 系统
- goyoubbs.exe   -- windows 系统
- goyoubbs_mac   -- mac 系统

默认端口是8082，修改配置文件 `config/config.yaml` 更改网站运行及网站基本信息

把文件夹`mysite` 的内容上传到服务器、运行。。。

若有疑问可以到官方论坛提问 https://www.youbbs.org





