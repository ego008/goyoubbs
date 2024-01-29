package main

import (
	"crypto/tls"
	"embed"
	"flag"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"goyoubbs/controller"
	"goyoubbs/cronjob"
	mdw "goyoubbs/middleware"
	"goyoubbs/model"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	// 挂载路由
	controller.RouterReload(myApp)
	mux := myApp.Mux

	var hd fasthttp.RequestHandler
	if myApp.Cf.Site.IsDevMod {
		hd = mdw.RspNoCache(mux.Handler)
	} else {
		hd = mux.Handler
	}

	log.Printf("Serving sdb from directory %q", *sdbDir)

	srv := &fasthttp.Server{
		Handler:         hd,
		Name:            "gyb Service",
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		// MaxConnsPerIP:
		// MaxRequestsPerConn:
		DisableKeepalive:              true,
		DisableHeaderNamesNormalizing: true,
		ReadTimeout:                   200 * time.Second, // important
		WriteTimeout:                  300 * time.Second,
		IdleTimeout:                   time.Minute,
		MaxRequestBodySize:            2000 << 20, // 100MB，上传文件最大值
	}

	// server model
	if len(*autoTLS) > 0 && len(*domain) > 0 {
		// Let's Encrypt, auto cert
		go func() {
			log.Printf("TCP address to listen to %q", *addr)
			if err := fasthttp.ListenAndServe(*addr, redirectHandler); err != nil {
				log.Fatalf("HTTP server ListenAndServe: %v", err)
			}
		}()

		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(*domain), // Replace with your domain.
			Cache:      autocert.DirCache("./certs"),
		}

		cfg := &tls.Config{
			GetCertificate: m.GetCertificate,
			NextProtos: []string{
				"http/1.1", acme.ALPNProto,
			},
		}

		// Let's Encrypt tls-alpn-01 only works on port 443.
		ln, err := net.Listen("tcp4", "0.0.0.0:443") /* #nosec G102 */
		if err != nil {
			log.Fatalf("net Listen: %v", err)
		}

		lnTls := tls.NewListener(ln, cfg)

		go func() {
			if err = srv.Serve(lnTls); err != nil {
				log.Fatalf("HTTPS server: %v", err)
			}
		}()
	} else if len(*certFile) > 0 && len(*keyFile) > 0 {
		// TLS with ertFile & keyFile
		//go func() {
		//	log.Printf("TCP address to listen to %q", *addr)
		//	if err := fasthttp.ListenAndServe(*addr, redirectHandler); err != nil {
		//		log.Fatalf("HTTP server ListenAndServe: %v", err)
		//	}
		//}()

		go func() {
			if err := srv.ListenAndServeTLS("0.0.0.0"+*addr, *certFile, *keyFile); err != nil {
				log.Fatalf("HTTPS ListenAndServeTLS: %v", err)
			}
		}()

	} else {
		// only http
		go func() {
			log.Printf("TCP address to listen to %q", *addr)
			if err := srv.ListenAndServe(*addr); err != nil {
				log.Fatalf("HTTP server ListenAndServe: %v", err)
			}
		}()
	}

	// graceful stop
	// subscribe to SIGINT signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(
		signalChan,
		syscall.SIGHUP,  // kill -SIGHUP XXXX
		syscall.SIGINT,  // kill -SIGINT XXXX or Ctrl+c
		syscall.SIGTERM, // kill -SIGTERM XXXX
		syscall.SIGQUIT, // kill -SIGQUIT XXXX
	)

	<-signalChan
	log.Printf("os.Interrupt - shutting down...\n")
	if err := srv.Shutdown(); err != nil {
		log.Println("Shutdown err", err)
		defer os.Exit(1)
	} else {
		myApp.Close() // !important 留意上下文位置
		log.Println("gracefully stopped")
	}

	//go func() {
	//	<-signalChan
	//	log.Fatal("os.Kill - terminating...\n")
	//}()

	defer os.Exit(0)
	return
}

func redirectHandler(ctx *fasthttp.RequestCtx) {
	ctx.Redirect("https://"+*domain+string(ctx.RequestURI()), fasthttp.StatusFound)
}
