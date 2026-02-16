package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() 	string
	IsAlive() 	bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newSimpleServer(addr string) *simpleServer {
	serverUrl, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port: port,
		roundRobinCount: 0,
		servers: servers,
	}
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount&len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount&len(lb.servers)]
	}
	lb.roundRobinCount++
	
	return server
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("forwarding request to address %s\n", targetServer.Address())
	targetServer.Serve(rw, r)
}

func main() {
	IP_ADDR := os.Getenv("IP_ADDR")
	PORT := os.Getenv("PORT")
	
	servers := []Server{
		newSimpleServer("https://www.duckduckgo.com"),
		newSimpleServer("https://www.google.com"),
		newSimpleServer("https://www.github.com"),
	}
	
	lb := NewLoadBalancer("8000", servers)
	
	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}
	
	fmt.Printf("serving on '%s:%s'\n", IP_ADDR, PORT)
	http.HandleFunc("/", handleRedirect)
	http.ListenAndServe(IP_ADDR+":"+PORT, nil)
}