package controller

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"github.com/whoisfisher/mykubespray/pkg/service"
)

type OSController struct {
	Ctx       context.Context
	osService service.OSService
}

func NewOSController() *OSController {
	return &OSController{
		osService: service.NewOSService(),
	}
}

var osController OSController

func init() {
	osController = *NewOSController()
}

func MountDisk(ctx *gin.Context) {
	var diskConf entity.DiskConf
	if err := ctx.ShouldBind(&diskConf); err != nil {
		logger.GetLogger().Errorf("DiskConf bind failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	err := osController.osService.Mount(diskConf)
	if err != nil {
		logger.GetLogger().Errorf("Mount disk failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	ginx.NewRender(ctx).Data("Mount disk success", nil)
}

func AddHosts(ctx *gin.Context) {
	var recordConf entity.RecordConf
	if err := ctx.ShouldBind(&recordConf); err != nil {
		logger.GetLogger().Errorf("RecordConf bind failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	err := osController.osService.AddHost(recordConf)
	if err != nil {
		logger.GetLogger().Errorf("add hosts failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	ginx.NewRender(ctx).Data("add hosts success", nil)
}

func CopyFile(ctx *gin.Context) {
	var certConf entity.CertConf
	if err := ctx.ShouldBind(&certConf); err != nil {
		logger.GetLogger().Errorf("RecordConf bind failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	err := osController.osService.CopyFile(certConf)
	if err != nil {
		logger.GetLogger().Errorf("copy cert failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	ginx.NewRender(ctx).Data("copy cert success", nil)
}

func ChangeExpiredPassword(ctx *gin.Context) {
	var passwordConf entity.PasswordConf
	if err := ctx.ShouldBind(&passwordConf); err != nil {
		logger.GetLogger().Errorf("PasswordConf bind failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	err := osController.osService.ChangeExpiredPassword(passwordConf)
	if err != nil {
		logger.GetLogger().Errorf("update password failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	ginx.NewRender(ctx).Data("update password success", nil)
}

func CheckPasswordInfo(ctx *gin.Context) {
	var passwordConf entity.Host
	if err := ctx.ShouldBind(&passwordConf); err != nil {
		logger.GetLogger().Errorf("PasswordConf bind failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	info, err := osController.osService.CheckPasswordInfo(passwordConf)
	if err != nil {
		logger.GetLogger().Errorf("check password failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	ginx.NewRender(ctx).Data(info, nil)
}

func UpdatePassword(ctx *gin.Context) {
	var passwordConf entity.PasswordConf
	if err := ctx.ShouldBind(&passwordConf); err != nil {
		logger.GetLogger().Errorf("PasswordConf bind failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	err := osController.osService.UpdatePassword(passwordConf)
	if err != nil {
		logger.GetLogger().Errorf("update password failed: %s", err.Error())
		ginx.Dangerous(err)
	}
	ginx.NewRender(ctx).Data("update password success", nil)
}
