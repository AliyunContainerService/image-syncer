package server

import "github.com/gin-gonic/gin"

type Server interface {
	Run() error
}

type HTTPServer struct {
	server *gin.Engine
}

func NewHTTPServer() (Server, error) {
	server := gin.Default()
	//v1 := server.Group("/api/v1", func(*gin.Context) {})

	return &HTTPServer{
		server: server,
	}, nil
}

func (s *HTTPServer) Run() error {
	//return server.Run(":9001")
	return nil
}
