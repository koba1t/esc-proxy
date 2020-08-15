// golang proxy server
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	escv1alpha1 "github.com/koba1t/ESC/api/v1alpha1"
	//https://pkg.go.dev/github.com/cenkalti/backoff/v4?tab=doc
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

	// create k8s client
	ctx := context.Background()
	cl, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
		os.Exit(1)
	}
	userland := &escv1alpha1.Userland{}
	nn := client.ObjectKey{
		Namespace: "default",
		Name:      "name",
	}
	_ = cl.Get(ctx, nn, userland)

	// Reverse proxy director
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

	errorHandle := func(rw http.ResponseWriter, req *http.Request, err error) {
		fmt.Printf("[ErrorHandle] http: proxy error: %v\n", err)

		username := req.Header.Get(usernameHeader)

		// create userland resource
		escuser := &escv1alpha1.Userland{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespaceName,
				Name:      username,
			},
			Spec: escv1alpha1.UserlandSpec{
				TemplateName: escTemplateName,
			},
		}
		e := cl.Create(context.Background(), escuser)
		if e != nil {
			fmt.Printf("Userland create error: %v\n", e)
		}

		rw.WriteHeader(http.StatusBadGateway)
		//https://golang.org/pkg/net/http/
	}

	rp := &httputil.ReverseProxy{
		Director:     director,
		ErrorHandler: errorHandle,
	}

	srv := &http.Server{
		Addr:    ":8080",
		Handler: rp,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
