package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync/atomic"
	"time"

	git "github.com/go-git/go-git"

)

type key int

const (
	requestIDKey key = 0
)

var (
	Version      string = ""
	GitTag       string = ""
	GitCommit    string = ""
	GitTreeState string = ""
	listenAddr   string
	healthy      int32
)

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":5000", "server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	logger.Println("Simple go server")
	logger.Println("Version:", Version)
	logger.Println("GitTag:", GitTag)
	logger.Println("GitCommit:", GitCommit)
	logger.Println("GitTreeState:", GitTreeState)

	logger.Println("Server is starting...")

	router := http.NewServeMux()
	router.Handle("/", index())
	router.Handle("/gitClone", gitClone())
	router.Handle("/openVsCode", openVsCode())
	router.Handle("/gitCommit", gitCommit())
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
}

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello, World!")
	})
}

type gitData struct {
	Repo string `json:"repo"`
}

func gitClone() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		if r.Method == "POST" {

			// Read body
			b, err := ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			// Unmarshal
			var msg gitData
			err = json.Unmarshal(b, &msg)
			if err != nil {
				http.Error(w, err.Error(), 500)
				panic(err)
			}

			logger.Println(msg.Repo)

			directory := "sample"
			url := "https://github.com/gophercises/renamer"

			repo, err := git.PlainClone(directory, false, &git.CloneOptions{
				URL:               url,
				RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			})

			if err != nil {
				logger.Println("Git Clone error", err)
				panic(err)
			}

			_, err = repo.Head()

			if err != nil {
				logger.Println("Git Clone repo head error", err)
				panic(err)
			}

			output, err := json.Marshal(msg)
			if err != nil {
				http.Error(w, err.Error(), 500)
				panic(err)
			}
			w.Header().Set("content-type", "application/json")
			w.Write(output)

		}
	})
}

func openVsCode() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		// decoder := json.NewDecoder(req.Body)
		// var t test_struct
		// err := decoder.Decode(&t)
		// if err != nil {
		//     panic(err)
		// }
		// log.Println(t.Test)
		repoPath := "C:\\Users\\Akshay\\go\\src\\github.com\\Akshay090\\gitifyService\\sample"
		cmd := exec.Command("code", repoPath)
		stdout, err := cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			return
		}

		logger.Println(string(stdout))
	})
}

func gitCommit() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		// decoder := json.NewDecoder(req.Body)
		// var t test_struct
		// err := decoder.Decode(&t)
		// if err != nil {
		//     panic(err)
		// }
		// log.Println(t.Test)
		repoPath := "C:\\Users\\Akshay\\go\\src\\github.com\\Akshay090\\gitifyService\\sample"
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoPath
		_, err := cmd.Output()
		if err != nil {
			logger.Println(err.Error())
			return
		}

		cmd = exec.Command("git", "commit", "-m", "commit from gitify")
		cmd.Dir = repoPath
		_, err = cmd.Output()
		if err != nil {
			logger.Println(err.Error())
			return
		}
		// logger.Println(string(stdout))
	})
}


func gitPush() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		// decoder := json.NewDecoder(req.Body)
		// var t test_struct
		// err := decoder.Decode(&t)
		// if err != nil {
		//     panic(err)
		// }
		// log.Println(t.Test)
		repoPath := "C:\\Users\\Akshay\\go\\src\\github.com\\Akshay090\\gitifyService\\sample"
		cmd := exec.Command("git", "push", "-u", "origin", "master")
		cmd.Dir = repoPath
		_, err := cmd.Output()
		if err != nil {
			logger.Println(err.Error())
			return
		}
		
		cmd = exec.Command("git", "commit", "-m", "commit from gitify")
		cmd.Dir = repoPath
		_, err = cmd.Output()
		if err != nil {
			logger.Println(err.Error())
			return
		}
		// logger.Println(string(stdout))
	})
}

func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
