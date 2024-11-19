package etcd

import (
	"context"
	"fmt"
	"github.com/whoisfisher/mykubespray/pkg/entity"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"github.com/whoisfisher/mykubespray/pkg/utils"
	"github.com/whoisfisher/mykubespray/pkg/utils/oss"
	"os"
	"time"
)

type BackupManager struct {
	OSClient    *utils.OSClient
	ClusterName string
	Config      *Config
	BackupDir   string
	LocalPath   string
	S3Uploader  *oss.S3Uploader
}

func NewBackupManager(host entity.Host, backupDir, localPath, clusterName string, uploader *oss.S3Uploader) *BackupManager {
	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewExecutor(host)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)

	return &BackupManager{
		OSClient:    osclient,
		ClusterName: clusterName,
		Config:      NewConfig(),
		BackupDir:   backupDir,
		LocalPath:   localPath,
		S3Uploader:  uploader,
	}
}

func (bm *BackupManager) BackupEtcd(ctx context.Context) error {
	fileName := fmt.Sprintf("etcd-backup-%d.db", time.Now().Unix())
	backupPath := fmt.Sprintf("%s/%s", bm.BackupDir, bm.ClusterName)
	if !bm.OSClient.SSExecutor.DirIsExist(backupPath) {
		if err := bm.OSClient.SSExecutor.MkDirALL(backupPath, func(s string) {
			logger.GetLogger().Infof("Create directory: %s", backupPath)
		}); err != nil {
			logger.GetLogger().Errorf("Failed to create directory: %s, %v", backupPath, err)
			return fmt.Errorf("Failed to create directory: %s, %w", backupPath, err)
		}
	}
	backupFilePath := fmt.Sprintf("%s/%s", backupPath, fileName)
	err := bm.readEtcdEnvFile("/etc/etcd.env")
	if err != nil {
		logger.GetLogger().Errorf("Failed to read env %s: %v", "/etc/etcd.env", err)
		return fmt.Errorf("Failed to read env: %s: %w", "/etc/etcd.env", err)
	}
	cmd := fmt.Sprintf("ETCDCTL_API=3 etcdctl --cacert=%s --key=%s --cert=%s --endpoints=%s snapshot save %s",
		os.Getenv("ETCDCTL_CA_FILE"), os.Getenv("ETCDCTL_KEY_FILE"), os.Getenv("ETCDCTL_CERT_FILE"), os.Getenv("ETCD_ADVERTISE_CLIENT_URLS"), backupFilePath)

	output, err := bm.OSClient.SSExecutor.ExecuteShortCommand(cmd)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create snapshot for etcd : %v, output: %s", err, output)
		return fmt.Errorf("Failed to create snapshot for etcd : %w, output: %s", err, output)
	}

	logger.GetLogger().Infof("etcd save to: %s", backupFilePath)

	bm.LocalPath = fmt.Sprintf("%s/%s", bm.LocalPath, fileName)
	err = bm.OSClient.SSExecutor.Download(backupFilePath, bm.LocalPath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to fetch backup file %s to %s: %v", backupFilePath, bm.LocalPath, err)
		return fmt.Errorf("Failed to fetch backup file %s to %s: %w", backupFilePath, bm.LocalPath, err)
	}
	if _, err := bm.S3Uploader.SimpleUpload(ctx, bm.LocalPath, backupFilePath); err != nil {
		logger.GetLogger().Errorf("Failed to upload backup file to S3: %w", err)
		return fmt.Errorf("Failed to upload backup file to S3: %w", err)
	}

	logger.GetLogger().Infof("Upload backup file to S3: s3://%s%s", bm.S3Uploader.BucketName, backupFilePath)

	delCmd := fmt.Sprintf("rm -f %s", backupFilePath)
	err = bm.OSClient.SSExecutor.ExecuteCommandWithoutReturn(delCmd)
	if err != nil {
		logger.GetLogger().Errorf("Failed to delete remote backup file: %v", err)
		return fmt.Errorf("Failed to delete remote backup file: %w", err)
	}

	if err := os.Remove(bm.LocalPath); err != nil {
		logger.GetLogger().Errorf("Failed to delete local backup file: %v", err)
		return fmt.Errorf("Failed to delete local backup file: %w", err)
	}
	logger.GetLogger().Info("Backup etcd successfully")
	return nil
}

func (bm *BackupManager) readEtcdEnvFile(filePath string) error {
	file, err := bm.OSClient.ReadFile(filePath)
	if err != nil {
		logger.GetLogger().Errorf("Cannot read file %s: %v", filePath, err)
		return fmt.Errorf("Cannot read file %s: %w", filePath, err)
	}
	err = SetEnvVars(file)
	if err != nil {
		logger.GetLogger().Errorf("Error:", err)
		return err
	}
	logger.GetLogger().Info("ETCD_NAME:", os.Getenv("ETCD_NAME"))

	return nil
}
