package main

import (
	"context"
	"crypto/tls"
	"flag"
	"github.com/ego008/goyoubbs/cronjob"
	"github.com/ego008/goyoubbs/getold"
	"github.com/ego008/goyoubbs/router"
	"github.com/ego008/goyoubbs/system"
	"goji.io"
	"goji.io/pat"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"xi2.org/x/httpgzip"
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

	// static file server
	staticPath := app.Cf.Main.PubDir
	if len(staticPath) == 0 {
		staticPath = "static"
	}
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

	if app.Cf.Main.HttpsOn {
		// https
		log.Println("Register sll for domain:", app.Cf.Main.Domain)

		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(app.Cf.Main.Domain),
			Cache:      autocert.DirCache("certs"),
		}

		srv = &http.Server{
			Addr:    ":" + strconv.Itoa(app.Cf.Main.HttpsPort),
			Handler: httpgzip.NewHandler(root, nil),
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
				NextProtos:     []string{http2.NextProtoTLS, "http/1.1"},
			},
			MaxHeaderBytes: int(app.Cf.Site.UploadMaxSizeByte),
		}

		go func() {
			log.Fatal(srv.ListenAndServeTLS("", ""))
		}()

		log.Println("Web server Listen port", app.Cf.Main.HttpsPort)
		log.Println("Web server URL", "https://"+app.Cf.Main.Domain)

		// rewrite
		go func() {
			if err := http.ListenAndServe(":"+strconv.Itoa(app.Cf.Main.HttpPort), http.HandlerFunc(redirectHandler)); err != nil {
				log.Println("Http2https server failed ", err)
			}
		}()

	} else {
		// http
		srv = &http.Server{Addr: ":" + strconv.Itoa(app.Cf.Main.HttpPort), Handler: root}
		go func() {
			log.Fatal(srv.ListenAndServe())
		}()

		log.Println("Web server Listen port", app.Cf.Main.HttpPort)
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
