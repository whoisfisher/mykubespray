package controller

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/whoisfisher/mykubespray/pkg/aop"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"github.com/whoisfisher/mykubespray/pkg/service"
	"github.com/whoisfisher/mykubespray/pkg/utils"
	"os"
)

type KubekeyController struct {
	Ctx            context.Context
	kubekeyService service.KubekeyService
}

func NewKubekeyController() *KubekeyController {
	return &KubekeyController{
		kubekeyService: service.NewKubekeyService(),
	}
}

var kubekeyController KubekeyController

func init() {
	kubekeyController = *NewKubekeyController()
}

func CreateCluster(ctx *gin.Context) {
	var conf entity.KubekeyConf
	ws, err := aop.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		logger.GetLogger().Errorf("Create websocket channel failed: %s", err.Error())
		ws.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
	err = ws.ReadJSON(&conf)
	if err != nil {
		logger.GetLogger().Errorf("Failed to read postgres info: %s", err.Error())
		ws.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
	logChan := make(chan utils.LogEntry)
	go func() {
		for logEntry := range logChan {
			if logEntry.IsError {
				fmt.Fprintf(os.Stderr, "[ERROR] %s\n", logEntry.Message)
				ws.WriteMessage(websocket.TextMessage, []byte(logEntry.Message))
			} else {
				fmt.Printf("[INFO] %s\n", logEntry.Message)
				ws.WriteMessage(websocket.TextMessage, []byte(logEntry.Message))
			}
		}
	}()
	kubekeyController.kubekeyService.CreateCluster(conf, logChan)

}
