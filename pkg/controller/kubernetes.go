package controller

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"github.com/whoisfisher/mykubespray/pkg/service"
)

type KubernetesController struct {
	Ctx               context.Context
	kubernetesService service.KubernetesService
}

func NewKubernetesController() *KubernetesController {
	return &KubernetesController{
		kubernetesService: service.NewKubernetesService(),
	}
}

var kubernetesController KubernetesController

func init() {
	kubernetesController = *NewKubernetesController()
}

func ApplyYAMLs(ctx *gin.Context) {
	var kubernetesConf entity.KubernetesFilesConf
	if err := ctx.ShouldBind(&kubernetesConf); err != nil {
		logger.GetLogger().Errorf("KubernetesFilesConf bind failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	results := kubernetesController.kubernetesService.ApplyYAMLs(kubernetesConf)
	if !results.OverallSuccess {
		err := errors.New("Apply yaml failed")
		ginx.NewRender(ctx).Data(results, err)
	}
	ginx.NewRender(ctx).Data(results, nil)
}
