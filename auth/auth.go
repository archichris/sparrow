package auth

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

var httpRoute = map[string]func(w http.ResponseWriter, r *http.Request){}
var loginPage = "/login.html"
var authRoutes = []string{"auth", "framework", "login.html"}

type AuthHandler struct {
	RootPath string
}

func Test() {
	return
}

func HandleFunc(route string, f func(w http.ResponseWriter, r *http.Request)) {
	httpRoute[route] = f
}

func init() {
	HandleFunc("/auth", Signin)
}

func getHandleFunc(target string) func(w http.ResponseWriter, r *http.Request) {
	rt := ""
	l := 0
	for m, _ := range httpRoute {
		if strings.HasPrefix(target, m) {
			if m == "/" || m == target || strings.HasPrefix(target, m+"/") {
				t := len(m)
				if t > l {
					l = t
					rt = m
				}
			}
		}
	}
	if rt != "" {
		return httpRoute[rt]
	}
	return nil
}

func (a *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := r.URL.EscapedPath()
	reqRoot := strings.Split(target, "/")[1]
	fmt.Println("mylog", target)

	i := 0
	p := ""

	for i, p = range authRoutes {
		if p == reqRoot {
			break
		}
	}

	if target == "/" || target == "index.html" {
		w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
	}

	if i+1 == len(authRoutes) {
		if _, err := CheckToken(w, r); err != nil {
			http.ServeFile(w, r, filepath.Join(a.RootPath, loginPage))
			return
		}
	}

	f := getHandleFunc(target)
	if f != nil {
		f(w, r)
	} else {
		http.FileServer(http.Dir(a.RootPath)).ServeHTTP(w, r)
	}
}
