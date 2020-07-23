// golang proxy server
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
)

func main() {
	usernameHeader := os.Getenv("USERNAME_HEADER")
	if usernameHeader == "" {
		// oauth2-proxy set username for X-Auth-Request-User
		// X-Auth-Request-User: koba1t
		usernameHeader = "X-Auth-Request-User"
	}
	localClusterDomain := os.Getenv("LOCAL_CLUSTER_DOMAIN")
	if localClusterDomain == "" {
		localClusterDomain = "cluster.local"
	}
	namespaceName := os.Getenv("TARGET_NAMESPACE_NAME")
	if namespaceName == "" {
		// If not set value, using default namespace.
		namespaceName = "default"
	}
	escTemplateName := os.Getenv("ESC_TEMPLATE_NAME")
	if escTemplateName == "" {
		log.Fatal("template name is not set")
	}

	director := func(req *http.Request) {
		username := req.Header.Get(usernameHeader)
		if username == "" {
			fmt.Printf("Username is not set at %s\n", usernameHeader)
			return
		}

		req.URL.Scheme = "http"
		req.URL.Host = escTemplateName + "-" + username + "-svc." + namespaceName + ".svc." + localClusterDomain

		fmt.Printf("ReverseProxy for %s\n", req.URL.Host)
	}

	modifyResponse := func(res *http.Response) error {
		fmt.Println("modifyResponse called")
		return nil
	}

	errorHandle := func(rw http.ResponseWriter, req *http.Request, err error) {
		fmt.Printf("http: proxy error: %v\n", err)
		fmt.Println("ErrorHandle called")
		rw.WriteHeader(http.StatusBadGateway)
		//https://golang.org/pkg/net/http/
	}

	rp := &httputil.ReverseProxy{
		Director:       director,
		ModifyResponse: modifyResponse,
		ErrorHandler:   errorHandle,
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: rp,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
