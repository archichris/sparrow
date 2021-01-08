package proxy

import (
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sparrow/auth"
	"strings"
)

func Test() {
	return
}

func init() {
	auth.HandleFunc("/prometheus", GenerateProxyFunc("/prometheus", "127.0.0.1:9090"))
	auth.HandleFunc("/classic", GenerateProxyFunc("", "127.0.0.1:9090"))
}

func GenerateProxyFunc(prefix, host string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		transport := http.DefaultTransport

		// step 1
		outReq := new(http.Request)
		*outReq = *r // this only does shallow copies of maps

		if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			if prior, ok := outReq.Header["X-Forwarded-For"]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
			outReq.Header.Set("X-Forwarded-For", clientIP)
		}

		outReq.URL.Host = host
		outReq.URL.Scheme = "http"
		if prefix != "" {
			path := "/"
			if outReq.URL.Path != prefix {
				path = strings.TrimPrefix(outReq.URL.Path, prefix)
			}
			outReq.URL.Path = strings.TrimPrefix(outReq.URL.Path, prefix)
			outReq.RequestURI = path
		}

		outReq.Host = host

		// step 2
		res, err := transport.RoundTrip(outReq)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		// step 3
		for key, value := range res.Header {
			for _, v := range value {
				if key == "Location" && prefix != "" {
					w.Header().Add(key, filepath.Join(prefix, v))
				} else {
					w.Header().Add(key, v)
				}

			}
		}

		w.WriteHeader(res.StatusCode)
		io.Copy(w, res.Body)
		res.Body.Close()
	}
}
