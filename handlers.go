package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
)

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
	Domain      string `json:"Domain"`
	RepoURL     string `json:"RepoURL"`
	GitUserName string `json:"GitUserName"`
	ProjectName string `json:"ProjectName"`
}

func gitClone() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		if r.Method == "POST" {

			decoder := json.NewDecoder(r.Body)
			var msg gitData
			err := decoder.Decode(&msg)
			if err != nil {
				panic(err)
			}

			repoRoot := "C:\\Users\\Akshay\\gitify"
			repoBase := filepath.Join(repoRoot, msg.Domain, msg.GitUserName)
			logger.Println("Repo Path", repoBase)

			if _, err := os.Stat(repoBase); os.IsNotExist(err) {
				logger.Println("Not exist creating")
				os.MkdirAll(repoBase, os.ModePerm)

			}
			cmd := exec.Command("git", "clone", msg.RepoURL)
			cmd.Dir = repoBase
			_, err = cmd.Output()

			if err != nil {
				logger.Println("Git Clone error", err)
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

		decoder := json.NewDecoder(r.Body)
		var msg gitData
		err := decoder.Decode(&msg)
		if err != nil {
			panic(err)
		}

		repoRoot := "C:\\Users\\Akshay\\gitify"
		repoPath := filepath.Join(repoRoot, msg.Domain, msg.GitUserName, msg.ProjectName)
		cmd := exec.Command("code", repoPath)
		stdout, err := cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			return
		}

		logger.Println(string(stdout))
	})
}

func gitPush() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		decoder := json.NewDecoder(r.Body)
		var msg gitData
		err := decoder.Decode(&msg)
		if err != nil {
			panic(err)
		}

		repoRoot := "C:\\Users\\Akshay\\gitify"
		repoPath := filepath.Join(repoRoot, msg.Domain, msg.GitUserName, msg.ProjectName)

		logger.Println("git add . in gitPush()")
		cmd := exec.Command("git", "add", ".", repoPath)
		cmd.Dir = repoPath
		stdout, err := cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			// return // Gives exit status 128 error but works !!
		}
		logger.Println(string(stdout))

		cmd = exec.Command("git", "commit", "-m", "commit from gitify", repoPath)
		cmd.Dir = repoPath
		stdout, err = cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			// return
		}
		logger.Println(string(stdout))

		cmd = exec.Command("git", "push", "-u", "origin", "master", repoPath)
		cmd.Dir = repoPath
		stdout, err = cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			// return
		}
		logger.Println(string(stdout))
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
