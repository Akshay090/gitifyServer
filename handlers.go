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
	"syscall"

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
	RootPath    string `json:"RootPath"`
	GitMsg      string `json:"GitMsg"`
}

type repoStatus struct {
	Exist bool `json:"Exist"`
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func repoExists() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		if r.Method == "POST" {
			decoder := json.NewDecoder(r.Body)
			var msg gitData
			err := decoder.Decode(&msg)
			if err != nil {
				panic(err)
			}

			repoRoot := msg.RootPath
			repoBase := filepath.Join(repoRoot, msg.Domain, msg.GitUserName)
			repoPath := filepath.Join(repoBase, msg.ProjectName)
			logger.Println("repoPath", repoPath)

			if repoExist, _ := exists(repoPath); repoExist {
				logger.Println("repo exist")
				repoRes := repoStatus{true}

				js, err := json.Marshal(repoRes)
				if err != nil {
					panic(err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
			} else {
				logger.Println("repo not exist")
				repoRes := repoStatus{false}

				js, err := json.Marshal(repoRes)
				if err != nil {
					panic(err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write(js)
			}

		}
	})
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

			repoRoot := msg.RootPath
			repoBase := filepath.Join(repoRoot, msg.Domain, msg.GitUserName)
			logger.Println("Repo Path", repoBase)

			if _, err := os.Stat(repoBase); os.IsNotExist(err) {
				logger.Println("Not exist creating")
				os.MkdirAll(repoBase, os.ModePerm)

			}
			cmd := exec.Command("git", "clone", msg.RepoURL)
			cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
			cmd.Dir = repoBase

			_, err = cmd.Output()

			if err != nil {
				logger.Println("Git Clone error", err)
				// panic(err)
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

		repoRoot := msg.RootPath
		repoPath := filepath.Join(repoRoot, msg.Domain, msg.GitUserName, msg.ProjectName)
		cmd := exec.Command("code", repoPath)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW

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

		repoRoot := msg.RootPath
		repoPath := filepath.Join(repoRoot, msg.Domain, msg.GitUserName, msg.ProjectName)

		logger.Println("git add . in gitPush()")
		cmd := exec.Command("git", "add", ".")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW

		cmd.Dir = repoPath
		stdout, err := cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			// return // Gives exit status 128 error but works !!
		}
		logger.Println(string(stdout))

		commitMsg := msg.GitMsg

		cmd = exec.Command("git", "commit", "-m", commitMsg)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
		cmd.Dir = repoPath
		stdout, err = cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			// return
		}
		logger.Println(string(stdout))

		cmd = exec.Command("git", "push", "-u", "origin", "master")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
		cmd.Dir = repoPath
		stdout, err = cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			// return
		}
		logger.Println(string(stdout))
	})
}

func gitPull() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		logger := log.New(os.Stdout, "http: ", log.LstdFlags)

		decoder := json.NewDecoder(r.Body)
		var msg gitData
		err := decoder.Decode(&msg)
		if err != nil {
			panic(err)
		}

		repoRoot := msg.RootPath
		repoPath := filepath.Join(repoRoot, msg.Domain, msg.GitUserName, msg.ProjectName)

		logger.Println("In gitPull")
		cmd := exec.Command("git", "pull")
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW

		cmd.Dir = repoPath
		stdout, err := cmd.Output()

		if err != nil {
			logger.Println(err.Error())
			// return // Gives exit status 128 error but works !!
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
