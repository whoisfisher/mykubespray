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

func (bm *BackupManager) BackupEtcd() error {
	fileName := fmt.Sprintf("etcd-backup-%s.db", time.Now().Format("20060102150405"))
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
	cmd, err := bm.getBackupCommand(backupFilePath)
	if err != nil {
		logger.GetLogger().Errorf("Error getting backup command: %v", err)
		return fmt.Errorf("Error getting backup command: %w", err)
	}

	output, err := bm.OSClient.SSExecutor.ExecuteShortCommand(cmd)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create snapshot for etcd : %v, output: %s", err, output)
		return fmt.Errorf("Failed to create snapshot for etcd : %w, output: %s", err, output)
	}

	logger.GetLogger().Infof("etcd snapshot saved to: %s", backupFilePath)

	bm.LocalPath = fmt.Sprintf("%s/%s", bm.LocalPath, fileName)
	err = bm.OSClient.SSExecutor.Download(backupFilePath, bm.LocalPath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to fetch backup file %s to %s: %v", backupFilePath, bm.LocalPath, err)
		return fmt.Errorf("Failed to fetch backup file %s to %s: %w", backupFilePath, bm.LocalPath, err)
	}
	if _, err := bm.S3Uploader.SimpleUpload(context.TODO(), bm.LocalPath, backupFilePath); err != nil {
		logger.GetLogger().Errorf("Failed to upload backup file to S3: %v", err)
		return fmt.Errorf("Failed to upload backup file to S3: %w", err)
	}

	logger.GetLogger().Infof("Upload backup file to S3: s3://%s/%s%s", bm.S3Uploader.Endpoint, bm.S3Uploader.BucketName, backupFilePath)

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

func (bm *BackupManager) getBackupCommand(backupFilePath string) (string, error) {
	if bm.OSClient.SSExecutor.FileIsExists("/etc/etcd.env") {
		err := bm.readEtcdEnvFile("/etc/etcd.env")
		if err != nil {
			logger.GetLogger().Errorf("Failed to read env %s: %v", "/etc/etcd.env", err)
			return "", fmt.Errorf("Failed to read env %s: %w", "/etc/etcd.env", err)
		}
	} else {
		logger.GetLogger().Errorf("The current feature does not support the cluster")
		return "", fmt.Errorf("The current feature does not support the cluster")
	}
	caCert := os.Getenv("ETCDCTL_CA_FILE")
	if len(caCert) == 0 {
		logger.GetLogger().Errorf("Error getting CA FILE")
		return "", fmt.Errorf("Error gettting CA FILE")
	}
	key := os.Getenv("ETCDCTL_KEY_FILE")
	if len(caCert) == 0 {
		logger.GetLogger().Errorf("Error getting KEY FILE")
		return "", fmt.Errorf("Error gettting KEY FILE")
	}
	cert := os.Getenv("ETCDCTL_CERT_FILE")
	if len(caCert) == 0 {
		logger.GetLogger().Errorf("Error getting CERT FILE")
		return "", fmt.Errorf("Error gettting CERT FILE")
	}
	endpoints := os.Getenv("ETCD_ADVERTISE_CLIENT_URLS") // 应取ETCDCTL_ENDPOINTS,但是ETCD_ADVERTISE_CLIENT_URLS
	if len(caCert) == 0 {
		logger.GetLogger().Errorf("Error getting ENDPOINTS")
		return "", fmt.Errorf("Error gettting ENDPOINTS")
	}
	command := fmt.Sprintf("ETCDCTL_API=3 etcdctl --cacert=%s --key=%s --cert=%s --endpoints=%s snapshot save %s",
		caCert, key, cert, endpoints, backupFilePath)
	return command, nil
}
