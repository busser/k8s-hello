// k8s-hello is a web server meant to be deployed inside a Kubernetes cluster
// in order to showcase basic Kubernetes features.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Command-line arguments.
var (
	httpAddr     = flag.String("http", ":80", "Listen address")
	initDelay    = flag.Duration("init", 1*time.Millisecond, "Time before accecpting requests")
	k8sNamespace = flag.String("namespace", "unknown", "Namespace inside which the server is deployed")
	k8sNode      = flag.String("node", "unknown", "Name of node on which server is running")
	k8sPod       = flag.String("pod", "unknown", "Name of pod in which server is running")
	suicideCode  = flag.Int("killcode", 1, "Code to return upon server suicide")
)

func main() {
	flag.Parse()
	http.Handle("/", NewServer(*initDelay))
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}

// Server implements the k8s-server.
// It serves basic information about the server's deployment, a healthcheck,
// that will report good health (readiness) after a given delay, and ways of
// remotely damaging the server's health or killing it.
type Server struct {
	router    *http.ServeMux
	initDelay time.Duration

	mu      sync.RWMutex // protects the healthy variable
	healthy bool
}

// NewServer returns an initialized k8s-hello server.
// The server will report itself as unhealthy/unready at first.
// After the initialization delay has passed, the server will report
// as being healthy/ready.
func NewServer(initDelay time.Duration) *Server {
	s := &Server{router: http.NewServeMux(), initDelay: initDelay}

	s.router.HandleFunc("/", s.HandleIndex(*k8sNamespace, *k8sNode, *k8sPod))
	s.router.HandleFunc("/healthz", s.HandleHealth())
	s.router.HandleFunc("/damage", s.HandleDamage())
	s.router.HandleFunc("/heal", s.HandleHeal())
	s.router.HandleFunc("/kill", s.HandleKill(*suicideCode))

	// Report server being healthy/ready after delay is passed.
	go func() {
		time.Sleep(initDelay)
		s.mu.Lock()
		s.healthy = true
		s.mu.Unlock()
	}()

	return s
}

// ServeHTTP uses the server's router to handle HTTP requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// HandleIndex provides basic information about the server's deployment.
func (s *Server) HandleIndex(namespace, node, pod string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Requested URL: %s\n", r.URL)
		fmt.Fprintf(w, "Kubernetes:\n")
		fmt.Fprintf(w, "  Namespace: %s\n", namespace)
		fmt.Fprintf(w, "  Node:      %s\n", node)
		fmt.Fprintf(w, "  Pod:       %s\n", pod)
	}
}

// HandleHealth reports on the server's health.
// It responds with an HTTP code of 200 if the server is
// healthy, 500 otherwise.
func (s *Server) HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		if s.healthy {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(500)
		}
	}
}

// HandleDamage makes the server report itself as unhealthy.
func (s *Server) HandleDamage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.healthy = false
	}
}

// HandleHeal makes the server report itself as unhealthy.
func (s *Server) HandleHeal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.healthy = true
	}
}

// HandleKill makes the server exit with the given error code.
func (s *Server) HandleKill(code int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		os.Exit(code)
	}
}
