package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/getlantern/systray"
	"github.com/getlantern/systray/example/icon"
	"github.com/skratchdot/open-golang/open"
)

type key int

const (
	requestIDKey key = 0
)

var (
	listenAddr string
	healthy    int32
)

func main() {
	onExit := func() {
		now := time.Now()
		ioutil.WriteFile(fmt.Sprintf(`on_exit_%d.txt`, now.UnixNano()), []byte(now.String()), 0644)
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTitle("Gitify")
	systray.SetTooltip("Gitify Service")
	mQuitOrig := systray.AddMenuItem("Quit", "Quit Gitify")
	go func() {
		<-mQuitOrig.ClickedCh
		fmt.Println("Requesting quit")
		systray.Quit()
		fmt.Println("Finished quitting")
	}()

	// We can manipulate the systray in other goroutines
	go func() {
		systray.SetTemplateIcon(icon.Data, icon.Data)
		systray.SetTitle("Gitify")
		systray.SetTooltip("Gitify Service")

		mUrl := systray.AddMenuItem("Open Gitify", "Gitify Website")

		for {
			select {

			case <-mUrl.ClickedCh:
				open.Run("https://www.getlantern.org")

			}
		}
	}()

	go func() {
		// Run your application/server code in here. Most likely you will
		// want to start an HTTP server that the user can hit with a browser
		// by clicking the tray icon.

		// Be sure to call this to link the tray icon to the target url

		flag.StringVar(&listenAddr, "listen-addr", ":5000", "server listen address")
		flag.Parse()
	
		logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	
		logger.Println("gitifyServer server")
	
		logger.Println("Server is starting...")
	
		router := http.NewServeMux()
		router.Handle("/", index())
		router.Handle("/gitClone", gitClone())
		router.Handle("/openVSCode", openVsCode())
		router.Handle("/gitPush", gitPush())
		router.Handle("/healthz", healthz())
	
		nextRequestID := func() string {
			return fmt.Sprintf("%d", time.Now().UnixNano())
		}
	
		server := &http.Server{
			Addr:         listenAddr,
			Handler:      tracing(nextRequestID)(logging(logger)(router)),
			ErrorLog:     logger,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		}
	
		done := make(chan bool)
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
	
		go func() {
			<-quit
			logger.Println("Server is shutting down...")
			atomic.StoreInt32(&healthy, 0)
	
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
	
			server.SetKeepAlivesEnabled(false)
			if err := server.Shutdown(ctx); err != nil {
				logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
			}
			close(done)
		}()
	
		logger.Println("Server is ready to handle requests at", listenAddr)
		atomic.StoreInt32(&healthy, 1)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
		}
	
		<-done
		logger.Println("Server stopped")

	}()
}
