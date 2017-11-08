# goyoubbs
golang 实现的youBBS https://www.youbbs.org

## 简单安装
如果你没有接触过golang 按照下面步骤也能快速部署

1）下载并解压 https://github.com/ego008/goyoubbs/archive/v3.0.zip

2）准备网站必须的文件
新建一个文件夹保存部署所需的文件，如mysite
把下面的文件夹copy 到mysite：
config
databackup
static
view

根据你的服务器环境copy 对应的主程序包到mysite
goyoubbs_linux -- linux 64bit 系统
goyoubbs.exe   -- windows 系统
goyoubbs_mac   -- mac 系统

3）运行程序
切换到mysite 文件夹，运行主程序包
默认端口是8082，在浏览器打开
`http://127.0.0.1:8082` 即可看到网站内容

4）修改配置文件 `config/config.yaml` 更改网站基本信息

若有疑问可以到官方论坛提问 https://www.youbbs.org





