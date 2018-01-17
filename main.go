package main

import (
	"context"
	"crypto/tls"
	"flag"
	"github.com/ego008/goyoubbs/cronjob"
	"github.com/ego008/goyoubbs/getold"
	"github.com/ego008/goyoubbs/router"
	"github.com/ego008/goyoubbs/system"
	"github.com/xi2/httpgzip"
	"goji.io"
	"goji.io/pat"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	configFile := flag.String("config", "config/config.yaml", "full path of config.yaml file")
	getOldSite := flag.String("getoldsite", "0", "get or not old site, 0 or 1, 2")
	flag.Parse()

	c := system.LoadConfig(*configFile)
	app := &system.Application{}
	app.Init(c, os.Args[0])

	if *getOldSite == "1" || *getOldSite == "2" {
		bh := &getold.BaseHandler{
			App: app,
		}
		if *getOldSite == "1" {
			bh.GetRemote()
		} else if *getOldSite == "2" {
			bh.GetLocal()
		}
		app.Close()
		return
	}

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

	// graceful
	// subscribe to SIGINT signals
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

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
		}

		go func() {
			// 如何获取 TLSCrtFile、TLSKeyFile 文件参见 https://www.youbbs.org/t/2169
			log.Fatal(srv.ListenAndServeTLS(mcf.TLSCrtFile, mcf.TLSKeyFile))
		}()

		log.Println("Web server Listen port", mcf.HttpsPort)
		log.Println("Web server URL", "https://"+mcf.Domain)

	} else {
		// http
		srv = &http.Server{Addr: ":" + strconv.Itoa(mcf.HttpPort), Handler: root}
		go func() {
			log.Fatal(srv.ListenAndServe())
		}()

		log.Println("Web server Listen port", mcf.HttpPort)
	}

	<-stopChan // wait for SIGINT
	log.Println("Shutting down server...")

	// shut down gracefully, but wait no longer than 10 seconds before halting
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	srv.Shutdown(ctx)
	app.Close()

	log.Println("Server gracefully stopped")
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	target := "https://" + r.Host + r.URL.Path
	if len(r.URL.RawQuery) > 0 {
		target += "?" + r.URL.RawQuery
	}
	// consider HSTS if your clients are browsers
	w.Header().Set("Connection", "close")
	http.Redirect(w, r, target, 301)
}

func stlAge(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// add max-age to get A+
		w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
