package middleware

import (
	"log"
	"net/http"
)

func BasicLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s]%s from [Host:%s | IP:%s]\n", r.Method, r.RequestURI, r.Host, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
