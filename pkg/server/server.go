package server

import (
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/codeready-toolchain/devcluster/pkg/configuration"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// DevClusterServer bundles configuration, and HTTP server objects in a single
// location.
type DevClusterServer struct {
	config      *configuration.Config
	router      *gin.Engine
	httpServer  *http.Server
	routesSetup sync.Once
}

// New creates a new DevClusterServer object with reasonable defaults.
func New(config *configuration.Config) *DevClusterServer {
	srv := &DevClusterServer{
		router: gin.Default(),
	}
	gin.DefaultWriter = io.MultiWriter(os.Stdout)

	srv.config = config

	srv.httpServer = &http.Server{
		Addr: srv.config.GetHTTPAddress(),
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: srv.config.GetHTTPWriteTimeout(),
		ReadTimeout:  srv.config.GetHTTPReadTimeout(),
		IdleTimeout:  srv.config.GetHTTPIdleTimeout(),
		Handler:      srv.router,
	}
	if srv.config.GetHTTPCompressResponses() {
		srv.router.Use(gzip.Gzip(gzip.DefaultCompression))
	}
	srv.router.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		//AllowOrigins:  []string{"https://foo.com"},
		AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		//AllowOriginFunc: func(origin string) bool {
		//	return origin == "https://github.com"
		//},
		//MaxAge: 12 * time.Hour,
	}))

	return srv
}

// Config returns the app server's config object.
func (srv *DevClusterServer) Config() *configuration.Config {
	return srv.config
}

// HTTPServer returns the app server's HTTP server.
func (srv *DevClusterServer) HTTPServer() *http.Server {
	return srv.httpServer
}

// Engine returns the app server's HTTP router.
func (srv *DevClusterServer) Engine() *gin.Engine {
	return srv.router
}

// GetRegisteredRoutes returns all registered routes formatted with their
// methods, paths, queries and names. It is a good idea to print this
// information on server start to give you an idea of what routes are
// available in the system.
func (srv *DevClusterServer) GetRegisteredRoutes() string {
	var sb strings.Builder

	for _, routeInfo := range srv.router.Routes() {
		sb.WriteString("ROUTE: ")
		sb.WriteString("\tRoute Path: ")
		sb.WriteString(routeInfo.Path)
		sb.WriteString("\n\tMethod: ")
		sb.WriteString(routeInfo.Method)
		sb.WriteString("\n")
	}
	return sb.String()
}
