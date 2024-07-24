package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/whoisfisher/mykubespray/pkg/aop"
	"os"
)

func New(version string) *gin.Engine {

	PrintAccessLog := viper.GetBool("bind.print_access_log")
	RunMode := viper.GetString("app.run_mode")
	gin.SetMode(RunMode)

	loggerMid := aop.Logrus()
	recoveryMid := aop.Recovery()
	r := gin.New()
	r.Use(recoveryMid)
	if PrintAccessLog {
		r.Use(loggerMid)
	}
	configRoute(r, version)
	return r
}

func configRoute(r *gin.Engine, version string) {
	httpRouter := r.Group("/api/v1")
	configHttpRouter(httpRouter, version)
	//
	ws := r.Group("/api/ws/v1")
	configWebsocketRouter(ws)
}

func configWebsocketRouter(rg *gin.RouterGroup) {
	rg.Use(aop.Cors())
}

func configHttpRouter(rg *gin.RouterGroup, version string) {
	rg.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	rg.GET("/pid", func(c *gin.Context) {
		c.String(200, fmt.Sprintf("%d", os.Getpid()))
	})
	rg.GET("/addr", func(c *gin.Context) {
		c.String(200, c.Request.RemoteAddr)
	})
	rg.GET("/version", func(c *gin.Context) {
		c.String(200, version)
	})
}
