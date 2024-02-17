package main

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

const (
	Attempts = iota
	Retry

)

type Server struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	servers []*Server
	current uint64
}
func (s *Server) SetAlive(alive bool) {
	s.mux.RLock()
	s.Alive = alive
	s.mux.RUnlock()
}

func (s *Server) IsAlive() (alive bool) {
	s.mux.RLock()
	alive = s.Alive
	s.mux.RUnlock()
	return
}

func (s *ServerPool) MarkServerStatus(serverUrl *url.URL,alive bool){
	for _,svr:= range s.servers{
		if svr.URL.String()==serverUrl.String(){
			svr.SetAlive(alive)
			break
		}
	} 
}

func (s *ServerPool) NextIdx() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.servers)))
}

func (s *ServerPool) GetNext() *Server {
	next:=s.NextIdx()
	l:=len(s.servers)+next
	for i:= range l{
		idx:= i%len(s.servers)
		if s.servers[idx].IsAlive(){
			if i!=next{
				atomic.AddUint64(&s.current,uint64(idx))
			}
			return s.servers[idx]
		}
	}
	return nil
}

