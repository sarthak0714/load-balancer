package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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

func (s *ServerPool) AddServer(svr *Server){
	s.servers=append(s.servers, svr)
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

func (s *ServerPool) HealthCheck(){
	for _,svr:= range s.servers{
		status:="up"
		alive:=isServerAlive(svr.URL)
		svr.SetAlive(alive)
		if !alive{
			status="down"
		}
		log.Printf("%s -> [%s]\n",svr.URL,status)
	}
}

func GetAttemptsFromContext(r *http.Request) int{
	if attempts,ok:=r.Context().Value(Attempts).(int);ok{
		return attempts
	}
	return 1
}	

func GetRetryFromContext(r * http.Request) int{
	if retry,ok:=r.Context().Value(Retry).(int); ok{
		return retry
	}
	return 0
}

func isServerAlive(u *url.URL) bool{
	to:=time.Second*2
	conn,err:=net.DialTimeout("tcp",u.Host,to)
	if err!=nil{
		log.Println("Unreachable error:",err)
		return false
	}
	defer conn.Close()
	return true
}

func healthCheck(){	
	t:=time.Minute*2
	for{
	select{
	case<-t.C:
		log.Println("Started Health Check")
		serverPool.HealthCheck()
		log.Println("Health Check Completed")
		}
	}

}

func loadBalance(w http.ResponseWriter, r* http.Request){
	attempts:=GetAttemptsFromContext(r)
	if attempts>3{
		log.Printf("%s (%s) Max Attempts reached, Terminating...\n",r.RemoteAddr,r.URL.Path)
		http.Error(w,"Service Unavailable",http.StatusServiceUnavailable)
		return
	}
	peer:=serverPool.GetNext()
	if peer!=nil{
		peer.ReverseProxy.ServeHTTP(w,r)\
		return
	}
	http.Error(w,"Service Unavailable",http.StatusServiceUnavailable)
}

var serverPool ServerPool

func main(){
	var svrList string
	var port int
	flag.StringVar(&svrList,"servers","","Loadbalanced Backends, sperate by commas")
	flag.IntVar(&port,"port",3000,"Port to Serve")
	flag.Parse()
	
	if len(svrList)==0{
		log.Fatal("Please provide Backends to Load Balance")
	}

	tokens:=strings.Split(svrList,",")
	for _,token := range tokens{
		serverUrl,err:=url.Parse(token)
		if err!=nil{
			log.Fatal(err)
		}
		proxy:=httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.ErrorHandler=func(w http.ResponseWriter,r *http.Request,e error){
			log.Printf("[%s] %s\n",serverUrl.Host,e.Error())
			retries:=GetRetryFromContext(r)
			if retries<4{
				select{
				case<-time.After(time.Microsecond*10):
					ctx:=context.WithValue(r.Context(),retries,retries+1)
					proxy.ServeHTTP(w,r.WithContext(ctx))
				}
				return
			}
			//shut down server after 4 retries
			serverPool.MarkServerStatus(serverUrl,false)
			attempts:=GetAttemptsFromContext(r)
			log.Printf("%s (%s) Attempting Retru %d\n",r.RemoteAddr,r.URL.Path,attempts)
			ctx:=context.WithValue(r.Context(),Attempts,attempts+1)
			loadBalance(w,r.WithContext(ctx))
			serverPool.AddServer(&Server{
				URL: serverUrl,
				Alive: true,
				ReverseProxy: proxy,
			})
			log.Printf("Server Configured: %S\n",serverUrl)
		}
		svr:=http.Server{
			Addr: fmt.Sprintf(":%d",port),
			Handler: http.HandlerFunc(loadBalance),
		}
		go healthCheck()

		log.Printf("Load Balancer Started at :%d\n",port)
		if err:=svr.ListenAndServe();err!=nil{
			log.Fatal(err)
		}
	}

}