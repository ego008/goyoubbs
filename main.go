package main

import (
	"context"
	"crypto/tls"
	"flag"
	"github.com/xi2/httpgzip"
	"goji.io"
	"goji.io/pat"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"goyoubbs/cronjob"
	"goyoubbs/router"
	"goyoubbs/system"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	configFile := flag.String("config", "config/config.yaml", "full path of config.yaml file")
	flag.Parse()

	c := system.LoadConfig(*configFile)
	app := &system.Application{}
	app.Init(c, os.Args[0])

	// cron job
	cr := cronjob.BaseHandler{App: app}
	go cr.MainCronJob()

	root := goji.NewMux()

	mcf := app.Cf.Main
	scf := app.Cf.Site

	// static file server
	staticPath := mcf.PubDir
	if len(staticPath) == 0 {
		staticPath = "static"
	}

	root.Handle(pat.New("/.well-known/acme-challenge/*"),
		http.StripPrefix("/.well-known/acme-challenge/", http.FileServer(http.Dir(staticPath))))
	root.Handle(pat.New("/static/*"),
		http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	root.Handle(pat.New("/*"), router.NewRouter(app))

	// normal http
	// http.ListenAndServe(listenAddr, root)

	var srv *http.Server

	if mcf.HttpsOn {
		// https
		log.Println("Register sll for domain:", mcf.Domain)
		log.Println("TLSCrtFile : ", mcf.TLSCrtFile)
		log.Println("TLSKeyFile : ", mcf.TLSKeyFile)

		root.Use(stlAge)

		tlsCf := &tls.Config{
			NextProtos: []string{http2.NextProtoTLS, "http/1.1"},
		}

		if mcf.Domain != "" && mcf.TLSCrtFile == "" && mcf.TLSKeyFile == "" {

			domains := strings.Split(mcf.Domain, ",")
			certManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(domains...),
				Cache:      autocert.DirCache("certs"),
				Email:      scf.AdminEmail,
			}
			tlsCf.GetCertificate = certManager.GetCertificate
			//tlsCf.ServerName = domains[0]

			go func() {
				// 必须是 80 端口
				log.Fatal(http.ListenAndServe(":http", certManager.HTTPHandler(nil)))
			}()

		} else {
			// rewrite
			go func() {
				if err := http.ListenAndServe(":"+strconv.Itoa(mcf.HttpPort), http.HandlerFunc(redirectHandler)); err != nil {
					log.Println("Http2https server failed ", err)
				}
			}()
		}

		srv = &http.Server{
			Addr:           ":" + strconv.Itoa(mcf.HttpsPort),
			Handler:        httpgzip.NewHandler(root, nil),
			TLSConfig:      tlsCf,
			MaxHeaderBytes: int(app.Cf.Site.UploadMaxSizeByte),
			ReadTimeout:    5 * time.Second,
			WriteTimeout:   10 * time.Second,
			BaseContext:    func(_ net.Listener) context.Context { return ctx },
		}
		srv.RegisterOnShutdown(cancel)

		go func() {
			// 如何获取 TLSCrtFile、TLSKeyFile 文件参见 https://www.youbbs.org/t/2169
			if err := srv.ListenAndServeTLS(mcf.TLSCrtFile, mcf.TLSKeyFile); err != http.ErrServerClosed {
				// it is fine to use Fatal here because it is not main gorutine
				log.Fatalf("HTTPS server ListenAndServe: %v", err)
			}
		}()

		log.Println("Web server Listen port", mcf.HttpsPort)
		log.Println("Web server URL", "https://"+mcf.Domain)

	} else {
		// http
		srv = &http.Server{
			Addr:         ":" + strconv.Itoa(mcf.HttpPort),
			Handler:      root,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			BaseContext:  func(_ net.Listener) context.Context { return ctx },
		}
		srv.RegisterOnShutdown(cancel)

		go func() {
			if err := srv.ListenAndServe(); err != http.ErrServerClosed {
				// it is fine to use Fatal here because it is not main gorutine
				log.Fatalf("HTTP server ListenAndServe: %v", err)
			}
		}()

		log.Println("Web server Listen port", mcf.HttpPort)
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
		syscall.SIGUSR2,
	)

	<-signalChan // wait for SIGINT
	log.Print("os.Interrupt - shutting down...\n")

	go func() {
		<-signalChan
		log.Fatal("os.Kill - terminating...\n")
	}()

	// 等待 30秒 ，等请求结束，同时不允许新的请求
	gracefulCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	srv.SetKeepAlivesEnabled(false)
	if err := srv.Shutdown(gracefulCtx); err != nil {
		log.Printf("shutdown error: %v\n", err)
		defer os.Exit(1)
		return
	} else {
		app.Close() // !important 留意上下文位置
		log.Printf("gracefully stopped\n")
	}

	// manually cancel context if not using httpServer.RegisterOnShutdown(cancel)
	// cancel()

	defer os.Exit(0)
	return
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	target := "https://" + r.Host + r.URL.Path
	if len(r.URL.RawQuery) > 0 {
		target += "?" + r.URL.RawQuery
	}
	// consider HSTS if your clients are browsers
	w.Header().Set("Connection", "close")
	http.Redirect(w, r, target, 302)
}

func stlAge(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// add max-age to get A+
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
