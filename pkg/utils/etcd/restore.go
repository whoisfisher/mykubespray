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

type RestoreManager struct {
	OSClient    *utils.OSClient
	BackupDir   string
	LocalPath   string
	ClusterName string
	S3Uploader  *oss.S3Uploader
	Config      *Config
}

func NewRestoreManager(host entity.Host, backupDir, localPath, clusterName string, uploader *oss.S3Uploader) *RestoreManager {
	osCOnf := utils.OSConf{}
	localExecutor := utils.NewLocalExecutor()
	sshExecutor := utils.NewExecutor(host)
	osclient := utils.NewOSClient(osCOnf, *sshExecutor, *localExecutor)

	return &RestoreManager{
		OSClient:    osclient,
		ClusterName: clusterName,
		Config:      NewConfig(),
		BackupDir:   backupDir,
		LocalPath:   localPath,
		S3Uploader:  uploader,
	}
}

func RestoreEtcdCluster(hosts []entity.Host, backupDir, localPath, clusterName, backupName string, uploader *oss.S3Uploader) error {

	mapRm := make(map[string]*RestoreManager)

	for _, host := range hosts {
		bm := NewRestoreManager(host, backupDir, localPath, clusterName, uploader)
		mapRm[host.Address] = bm
		err := bm.PauseKubeAPI()
		if err != nil {
			logger.GetLogger().Errorf("Failed to stop kube-apiserver: %v", err)
			return fmt.Errorf("Failed to stop kube-apiserver: %w", err)
		}
	}

	for _, host := range hosts {
		//bm := NewRestoreManager(host, backupDir, localPath, clusterName, uploader)
		err := mapRm[host.Address].RestoreEtcd(context.Background(), backupName)
		if err != nil {
			logger.GetLogger().Errorf("Failed to restore etcd: %v", err)
			return fmt.Errorf("Failed to restore etcd: %w", err)
		}
	}

	for _, host := range hosts {
		//bm := NewRestoreManager(host, backupDir, localPath, clusterName, uploader)
		err := mapRm[host.Address].ResumeKubeAPI()
		if err != nil {
			logger.GetLogger().Errorf("Failed to start kube-apiserver: %v", err)
			return fmt.Errorf("Failed to start kube-apiserver: %w", err)
		}
	}
	return nil
}

func (rm *RestoreManager) RestoreEtcd(ctx context.Context, backupFileName string) error {
	backupDir := fmt.Sprintf("%s/%s", rm.BackupDir, rm.ClusterName)
	if !rm.OSClient.SSExecutor.DirIsExist(backupDir) {
		if err := rm.OSClient.SSExecutor.MkDirALL(backupDir, func(s string) {
			logger.GetLogger().Infof("Create directory: %s", backupDir)
		}); err != nil {
			logger.GetLogger().Errorf("Failed to create directory: %s, %v", backupDir, err)
			return fmt.Errorf("Failed to create directory: %s, %w", backupDir, err)
		}
	}
	backupFilePath := fmt.Sprintf("%s/%s/%s", rm.BackupDir, rm.ClusterName, backupFileName)
	localFile := fmt.Sprintf("%s/%s", rm.LocalPath, backupFileName)

	if err := rm.S3Uploader.SimpleDownload(ctx, "data/"+rm.ClusterName+"/"+backupFileName, localFile); err != nil {
		logger.GetLogger().Errorf("Failed to download backup file from s3: %v", err)
		return fmt.Errorf("Failed to download backup file from s3: %w", err)
	}

	logger.GetLogger().Infof("Download backup file: %s", localFile)

	err := rm.OSClient.SSExecutor.Upload(localFile, backupFilePath)
	if err != nil {
		logger.GetLogger().Infof("Failed to upload file %s to %s: %v", localFile, backupFilePath, err)
		return fmt.Errorf("Failed to upload file %s to %s: %w", localFile, backupFilePath, err)
	}

	err = rm.OSClient.Chmod(backupFilePath, "0600")
	if err != nil {
		logger.GetLogger().Infof("Failed to chmod file %s: %v", backupFilePath, err)
		return fmt.Errorf("Failed to chmod file %s: %w", backupFilePath, err)
	}

	if err := rm.StopEtcd(); err != nil {
		logger.GetLogger().Errorf("Failed to stop etcd: %v", err)
		return fmt.Errorf("Failed to stop etcd: %w", err)
	}
	if err := rm.BackupEtcdDir(); err != nil {
		logger.GetLogger().Errorf("Backup /var/lib/etcd failure")
		return fmt.Errorf("Backup /var/lib/etcd failure")
	}

	if err := rm.restoreEtcdSnapshot(backupFilePath); err != nil {
		logger.GetLogger().Errorf("Failed to restore node:%s: %v", os.Getenv("ETCD_ADVERTISE_CLIENT_URLS"), err)
		return fmt.Errorf("Failed to restore node:%s: %w", os.Getenv("ETCD_ADVERTISE_CLIENT_URLS"), err)
	}

	if err := rm.StartEtcd(); err != nil {
		logger.GetLogger().Errorf("Failed to start etcd: %v", err)
		return fmt.Errorf("Failed to start etcd: %w", err)
	}

	if err := os.Remove(localFile); err != nil {
		logger.GetLogger().Errorf("Failed to delete local backup file %s:%v", localFile, err)
		return fmt.Errorf("Failed to delete local backup file %s:%w", localFile, err)
	}

	delCmd := fmt.Sprintf("rm -f %s", backupFilePath)
	err = rm.OSClient.SSExecutor.ExecuteCommandWithoutReturn(delCmd)
	if err != nil {
		logger.GetLogger().Errorf("Failed to delete remote backup file %s: %v", backupFilePath, err)
		return fmt.Errorf("Failed to delete remote backup file %s: %w", backupFilePath, err)
	}

	logger.GetLogger().Info("Restore etcd successfully")
	return nil
}

func (rm *RestoreManager) restoreEtcdSnapshot(snapshotPath string) error {
	dataDir := "/var/lib/etcd"

	command, err := rm.getRestoreCommand(dataDir, snapshotPath)
	if err != nil {
		logger.GetLogger().Errorf("Error getting restore command: %v", err)
		return fmt.Errorf("Error getting restore command: %w", err)
	}
	output, err := rm.OSClient.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Errorf("Failed to restore snapshot for etcd : %v, %s", err, output)
		return fmt.Errorf("Failed to restore snapshot for etcd : %w, %s", err, output)
	}

	logger.GetLogger().Infof("Successfully to restore snapshot: %s", snapshotPath)
	return nil
}

func (rm *RestoreManager) readEtcdEnvFile(filePath string) error {
	file, err := rm.OSClient.ReadFile(filePath)
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

func (rm *RestoreManager) PauseKubeAPI() error {
	manifestPath := "/etc/kubernetes/manifests/kube-apiserver.yaml"
	tempPath := fmt.Sprintf("/etc/kubernetes/manifests/kube-apiserver.yaml.bak")
	command := fmt.Sprintf("mv -f %s %s", manifestPath, tempPath)
	if rm.OSClient.WhoAmI() != "root" {
		command = utils.SudoPrefixWithPassword(command, rm.OSClient.SSExecutor.Host.Password)
	}
	err := rm.OSClient.SSExecutor.ExecuteCommandWithoutReturn(command)
	if err != nil {
		logger.GetLogger().Errorf("Backup and stop kube-apiserver failure: %v", err)
		return fmt.Errorf("Backup and stop kube-apiserver failure: %w", err)
	}
	logger.GetLogger().Info("Kube-apiserver stopped")
	return nil
}

func (rm *RestoreManager) ResumeKubeAPI() error {
	manifestPath := "/etc/kubernetes/manifests/kube-apiserver.yaml"
	tempPath := fmt.Sprintf("/etc/kubernetes/manifests/kube-apiserver.yaml.bak")

	command := fmt.Sprintf("mv -f %s %s", tempPath, manifestPath)
	if rm.OSClient.WhoAmI() != "root" {
		command = utils.SudoPrefixWithPassword(command, rm.OSClient.SSExecutor.Host.Password)
	}
	err := rm.OSClient.SSExecutor.ExecuteCommandWithoutReturn(command)
	if err != nil {
		logger.GetLogger().Errorf("Restore and start kube-apiserver failure")
		return fmt.Errorf("Restore and start kube-apiserver failure")
	}
	logger.GetLogger().Infof("kube-apiserver resume")
	//if err := rm.checkApiServerStatus(); err != nil {
	//	logger.GetLogger().Errorf("Error checking kube-apiserver status")
	//	return fmt.Errorf("Error checking kube-apiserver status")
	//}
	//if err := rm.checkNodeStatus(); err != nil {
	//	logger.GetLogger().Errorf("Error checking node %s status: %v", rm.OSClient.SSExecutor.Host.Name, err)
	//	return fmt.Errorf("Error checking node %s status: %w", rm.OSClient.SSExecutor.Host.Name, err)
	//}
	return nil
}

func (rm *RestoreManager) StopEtcd() error {
	err := rm.OSClient.StopService("etcd")
	if err != nil {
		logger.GetLogger().Errorf("Failed to stop etcd: %v", err)
		return fmt.Errorf("Failed to stop etcd: %w", err)
	}
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	res := rm.OSClient.StatusService("etcd")
	if !res {
		logger.GetLogger().Infof("Successfully stopped etcd")
		return nil
	}

	for {
		select {
		case <-timeout:
			logger.GetLogger().Errorf("Timeout while stopping etcd")
			return fmt.Errorf("Timeout while stopping etcd")
		case <-ticker.C:
			res = rm.OSClient.StatusService("etcd")
			if !res {
				logger.GetLogger().Infof("Successfully to stop etcd")
				return nil
			}
		}
	}
}

func (rm *RestoreManager) StartEtcd() error {
	err := rm.OSClient.StartService("etcd")
	if err != nil {
		logger.GetLogger().Errorf("Failed to start etcd: %v", err)
		return fmt.Errorf("Failed to start etcd: %w", err)
	}
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	res := rm.OSClient.StatusService("etcd")
	if res {
		logger.GetLogger().Infof("Successfully start etcd")
		return nil
	}

	for {
		select {
		case <-timeout:
			logger.GetLogger().Errorf("Timeout while starting etcd")
			return fmt.Errorf("Timeout while starting etcd")
		case <-ticker.C:
			res := rm.OSClient.StatusService("etcd")
			if res {
				logger.GetLogger().Infof("Successfully to start etcd")
				return nil
			}
		}
	}
}

func (rm *RestoreManager) BackupEtcdDir() error {
	manifestPath := "/var/lib/etcd"
	tempPath := fmt.Sprintf("/data/%s/etcd-%s", rm.ClusterName, time.Now().Format("20060102150405"))
	if !rm.OSClient.SSExecutor.DirIsExist(tempPath) {
		err := rm.OSClient.SSExecutor.MkDirALL(tempPath, func(s string) {
			logger.GetLogger().Infof("Create directory %s", tempPath)
		})
		if err != nil {
			logger.GetLogger().Errorf("Failed to create directory %s: %v", tempPath, err)
			return fmt.Errorf("Failed to create directory %s: %w", tempPath, err)
		}
	}

	command := fmt.Sprintf("mv -f %s %s", manifestPath, tempPath)
	if rm.OSClient.WhoAmI() != "root" {
		command = utils.SudoPrefixWithPassword(command, rm.OSClient.SSExecutor.Host.Password)
	}
	err := rm.OSClient.SSExecutor.ExecuteCommandWithoutReturn(command)
	if err != nil {
		logger.GetLogger().Errorf("Backup %s to %s failure", manifestPath, tempPath)
		return fmt.Errorf("Backup %s to %s failure", manifestPath, tempPath)
	}

	logger.GetLogger().Infof("Backup %s to %s success", manifestPath, tempPath)
	return nil
}

func (rm *RestoreManager) getRestoreCommand(dataDir, snapshotPath string) (string, error) {
	if rm.OSClient.SSExecutor.FileIsExists("/etc/etcd.env") {
		err := rm.readEtcdEnvFile("/etc/etcd.env")
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
	if len(key) == 0 {
		logger.GetLogger().Errorf("Error getting KEY FILE")
		return "", fmt.Errorf("Error gettting KEY FILE")
	}
	cert := os.Getenv("ETCDCTL_CERT_FILE")
	if len(cert) == 0 {
		logger.GetLogger().Errorf("Error getting CERT FILE")
		return "", fmt.Errorf("Error gettting CERT FILE")
	}
	endpoints := os.Getenv("ETCD_ADVERTISE_CLIENT_URLS") // 应取ETCDCTL_ENDPOINTS,但是ETCD_ADVERTISE_CLIENT_URLS
	if len(endpoints) == 0 {
		logger.GetLogger().Errorf("Error getting ENDPOINTS")
		return "", fmt.Errorf("Error gettting ENDPOINTS")
	}
	name := os.Getenv("ETCD_NAME")
	if len(name) == 0 {
		logger.GetLogger().Errorf("Error getting ETCD NAME")
		return "", fmt.Errorf("Error gettting ETCD NAME")
	}
	initialCluster := os.Getenv("ETCD_INITIAL_CLUSTER")
	if len(initialCluster) == 0 {
		logger.GetLogger().Errorf("Error getting ETCD INITIAL CLUSTER")
		return "", fmt.Errorf("Error gettting ETCD INITIAL CLUSTER")
	}
	initialAdvertisePeerUrls := os.Getenv("ETCD_INITIAL_ADVERTISE_PEER_URLS")
	if len(initialAdvertisePeerUrls) == 0 {
		logger.GetLogger().Errorf("Error getting ETCD INITIAL ADVERTISE PEER CLUSTER")
		return "", fmt.Errorf("Error getting ETCD INITIAL ADVERTISE PEER CLUSTER")
	}
	command := fmt.Sprintf("ETCDCTL_API=3 etcdctl --cacert=%s --key=%s --cert=%s --endpoints=%s --name=%s --initial-cluster=%s --initial-advertise-peer-urls=%s --data-dir=%s snapshot restore %s",
		caCert, key, cert, endpoints, name, initialCluster, initialAdvertisePeerUrls, dataDir, snapshotPath)
	return command, nil
}

func (rm *RestoreManager) checkApiServerStatus() error {
	command := fmt.Sprintf("crictl ps -a | grep kube-apiserver | grep -i running")
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	res, err := rm.OSClient.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Infof("Please wait for a moment, checking kube-apiserver status: %v.", err)
	}
	if len(res) > 0 {
		logger.GetLogger().Infof("Successfully start kube-apiserver.")
		return nil
	}

	for {
		select {
		case <-timeout:
			logger.GetLogger().Errorf("Timeout while waiting to start kube-apiserver")
			return fmt.Errorf("Timeout while waiting to start kube-apiserver")
		case <-ticker.C:
			res, err := rm.OSClient.SSExecutor.ExecuteShortCommand(command)
			if err != nil {
				logger.GetLogger().Infof("Please wait for a moment, checking kube-apiserver status: %v.", err)
			}
			if len(res) > 0 {
				logger.GetLogger().Infof("Successfully start kube-apiserver.")
				return nil
			} else {
				logger.GetLogger().Infof("Checking apiserver status, please wait for a moment")
			}
		}
	}
}

func (rm *RestoreManager) checkNodeStatus() error {
	command := fmt.Sprintf("kubectl get node -owide | grep -i %s | grep -i Ready", rm.OSClient.SSExecutor.Host.Name)
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	res, err := rm.OSClient.SSExecutor.ExecuteShortCommand(command)
	if err != nil {
		logger.GetLogger().Infof("Please wait for a moment, checking node %s status: %v", rm.OSClient.SSExecutor.Host.Name, err)
	} else if len(res) > 0 {
		logger.GetLogger().Infof("The node is already in a ready state.")
		return nil
	}

	for {
		select {
		case <-timeout:
			logger.GetLogger().Errorf("Timeout while checking node %s status", rm.OSClient.SSExecutor.Host.Name)
			return fmt.Errorf("Timeout while checking node %s status", rm.OSClient.SSExecutor.Host.Name)
		case <-ticker.C:
			res, err := rm.OSClient.SSExecutor.ExecuteShortCommand(command)
			if err != nil {
				logger.GetLogger().Infof("Please wait for a moment, checking node %s status: %v", rm.OSClient.SSExecutor.Host.Name, err)
			} else if len(res) > 0 {
				logger.GetLogger().Infof("The node is already in a ready state.")
				return nil
			}
		}
	}
}
