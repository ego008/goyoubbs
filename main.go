package main

import (
	"context"
	"crypto/tls"
	"embed"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goyoubbs/controller"
	"goyoubbs/cronjob"
	mdw "goyoubbs/middleware"
	"goyoubbs/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

//go:embed static
var staticFs embed.FS

var (
	addr     = flag.String("addr", ":8080", "TCP address to listen to")
	sdbDir   = flag.String("sdbDir", "localdb", "Directory to serve sdb from")
	autoTLS  = flag.String("autoTLS", "", "user Let's Encrypt. Leave empty for disabling")
	domain   = flag.String("domain", "", "set domain when user Let's Encrypt")
	certFile = flag.String("certFile", "", "Path to TLS certificate file")
	keyFile  = flag.String("keyFile", "", "Path to TLS key file")
)

func main() {
	flag.Parse()

	myApp := &model.Application{}
	myApp.Init(*addr, *sdbDir, &staticFs)

	// cron job
	cr := cronjob.BaseHandler{App: myApp}
	go cr.MainCronJob()

	// 初始化 Gin 引擎
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// 开发模式下挂载无缓存中间件（需要确保 mdw.RspNoCache 适配了 gin.HandlerFunc）
	if myApp.Cf.Site.IsDevMod {
		router.Use(mdw.RspNoCache())
	}

	// 挂载路由（请注意修改 controller.RouterReload 使其接收 *gin.Engine 或将 router 挂载到 myApp）
	myApp.Mux = router // 假设在 model.Application 中添加了 GinEngine 字段
	controller.RouterReload(myApp)

	log.Printf("Serving sdb from directory %q", *sdbDir)

	// 配置 http.Server 参数（替代 fasthttp.Server）
	srv := &http.Server{
		Addr:           *addr,
		Handler:        router,
		ReadTimeout:    200 * time.Second,
		WriteTimeout:   300 * time.Second,
		IdleTimeout:    time.Minute,
		MaxHeaderBytes: 1 << 20, // 1MB header size limit
	}

	// 定义 HTTP 到 HTTPS 重定向 Handler
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + *domain + r.URL.Path
		if len(r.URL.RawQuery) > 0 {
			target += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, target, http.StatusFound)
	})

	// Server mode 匹配
	if len(*autoTLS) > 0 && len(*domain) > 0 {
		// Let's Encrypt, auto cert
		go func() {
			log.Printf("HTTP redirect server listening on %q", *addr)
			if err := http.ListenAndServe(*addr, redirectHandler); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server ListenAndServe: %v", err)
			}
		}()

		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(*domain),
			Cache:      autocert.DirCache("./certs"),
		}

		cfg := &tls.Config{
			GetCertificate: m.GetCertificate,
			NextProtos: []string{
				"h2", "http/1.1", acme.ALPNProto,
			},
		}

		// Let's Encrypt tls-alpn-01 only works on port 443
		ln, err := net.Listen("tcp4", "0.0.0.0:443")
		if err != nil {
			log.Fatalf("net Listen: %v", err)
		}

		lnTls := tls.NewListener(ln, cfg)

		go func() {
			if err := srv.Serve(lnTls); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS server: %v", err)
			}
		}()
	} else if len(*certFile) > 0 && len(*keyFile) > 0 {
		// TLS with certFile & keyFile
		go func() {
			if err := srv.ListenAndServeTLS(*certFile, *keyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS ListenAndServeTLS: %v", err)
			}
		}()
	} else {
		// Only HTTP
		go func() {
			log.Printf("TCP address to listen to %q", *addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server ListenAndServe: %v", err)
			}
		}()
	}

	// 优雅关机（Graceful Shutdown）
	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	<-signalChan
	log.Println("os.Interrupt - shutting down...")

	// 设置关机超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Shutdown err:", err)
		os.Exit(1)
	} else {
		myApp.Close() // 留意上下文位置
		log.Println("gracefully stopped")
	}

	os.Exit(0)
}
