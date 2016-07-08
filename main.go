package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	config "github.com/kazoup/config/srv/proto/config"
	"github.com/kazoup/web-proxy/proxy"
	"github.com/kazoup/web-proxy/server"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/client"
	"github.com/micro/go-micro/cmd"
	"github.com/micro/go-micro/selector"
	"golang.org/x/net/context"
)

var (
	re = regexp.MustCompile("^[a-zA-Z0-9]+$")
	// Default address to bind to
	Address = ":8000"
	// The namespace to serve
	// Example:
	// Namespace + /[Service]/foo/bar
	// Host: Namespace.Service Endpoint: /foo/bar
	Namespace = "go.micro.web"
	// Base path sent to web service.
	// This is stripped from the request path
	// Allows the web service to define absolute paths
	BasePathHeader = "X-Micro-Web-Base-Path"
	statsURL       string
)

type srv struct {
	*mux.Router
}

func (s *srv) proxy() http.Handler {
	sel := selector.NewSelector(
		selector.Registry((*cmd.DefaultOptions().Registry)),
	)
	sel.Init()
	director := func(r *http.Request) {
		kill := func() {
			r.URL.Host = ""
			r.URL.Path = ""
			r.URL.Scheme = ""
			r.Host = ""
			r.RequestURI = ""
		}

		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 2 {
			kill()
			return
		}

		if !re.MatchString(parts[1]) {
			kill()
			return
		}
		next, err := sel.Select(Namespace + "." + parts[1])
		if err != nil {
			kill()
			return
		}

		s, err := next()
		if err != nil {
			kill()
			return
		}
		r.Header.Set(BasePathHeader, "/"+parts[1])
		r.URL.Host = fmt.Sprintf("%s:%d", s.Address, s.Port)
		r.URL.Path = "/" + strings.Join(parts[2:], "/")
		r.URL.Scheme = "http"
		r.Host = r.URL.Host
	}

	return &proxy.Proxy{
		Default:  &httputil.ReverseProxy{Director: director},
		Director: director,
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	srvReq := client.NewRequest(
		"go.micro.srv.config",
		"Config.Status",
		&config.StatusRequest{},
	)
	srvRsp := &config.StatusResponse{}

	if err := client.Call(context.Background(), srvReq, srvRsp); err != nil {
		// TODO: ???
	}

	// TODO: URLS are hardcoded, deploying to several machines...
	// Two servers, two IPs, two web-proxy srv
	// DNS is probably best approach, as it will be kick some instance, and let micro to do
	// the load balancing?

	if srvRsp.APPLIANCE_IS_DEMO {
		//https://demo.kazoup.com/demo/login/?user=info@kazoup.com:1Zk75k:xqI71h9y_27_gcrdGI91s-ryG4g
		http.Redirect(w, r, "http://localhost:8000/demo/login", 301)
	}

	// App login
	if srvRsp.APPLIANCE_IS_REGISTERED {
		http.Redirect(w, r, "http://localhost:8000/login", 301)
	}

	// App registration
	if !srvRsp.APPLIANCE_IS_REGISTERED {
		http.Redirect(w, r, "http://localhost:8000/register", 301)
	}

	// App first time driven configuration
	if !srvRsp.APPLIANCE_IS_CONFIGURED {
		http.Redirect(w, r, "http://localhost:8000/wizard", 301)
	}

	//TODO: default if config service behaves unexpectedly
	//http.Redirect(w, r, "http://www.golang.org", 301)
}

func main() {
	cmd.Init()
	var h http.Handler
	r := mux.NewRouter()
	s := &srv{r}
	h = s

	s.PathPrefix("/{service:[a-zA-Z0-9]+}").Handler(s.proxy())
	s.HandleFunc("/", indexHandler)

	var opts []server.Option

	srv := server.NewServer(Address)
	srv.Init(opts...)
	srv.Handle("/", h)

	// Initialise Server
	service := micro.NewService(
		micro.Name("go.micro.srv.proxy"),
		micro.Version("latest"),
	)

	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	// Init service
	service.Init()
	// Run server
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}

	if err := srv.Stop(); err != nil {
		log.Fatal(err)
	}
}
