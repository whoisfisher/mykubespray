package oss

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type S3Uploader struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
	UseSSL          bool
	client          *minio.Client
	cacheDir        string
	notifyProgress  func(int64, int64)
}

// New 创建并初始化 S3Uploader
func New(endpoint, accessKeyID, secretAccessKey, bucketName, region string, useSSL bool, cacheDir string, notifyProgress func(int64, int64)) (*S3Uploader, error) {
	uploader := &S3Uploader{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		BucketName:      bucketName,
		Region:          region,
		UseSSL:          useSSL,
		cacheDir:        cacheDir,
		notifyProgress:  notifyProgress,
	}

	client, err := uploader.initClient()
	if err != nil {
		return nil, fmt.Errorf("初始化 MinIO 客户端失败: %w", err)
	}
	uploader.client = client
	return uploader, nil
}

func NewS3(endpoint, accessKeyID, secretAccessKey, bucketName, region string, useSSL bool) (*S3Uploader, error) {
	uploader := &S3Uploader{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		BucketName:      bucketName,
		Region:          region,
		UseSSL:          useSSL,
		cacheDir:        "/tmp",
		notifyProgress: func(i int64, i2 int64) {
			progress := (float64(i) / float64(i2)) * 100
			logger.GetLogger().Infof("Uploaded: %.2f%%", progress)
		},
	}

	client, err := uploader.initClient()
	if err != nil {
		logger.GetLogger().Errorf("Failed to init minio client: %v", err)
		return nil, fmt.Errorf("Failed to init minio client: %v", err)
	}
	uploader.client = client
	return uploader, nil
}

// initClient 初始化 MinIO 客户端
func (s *S3Uploader) initClient() (*minio.Client, error) {
	client, err := minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKeyID, s.SecretAccessKey, ""),
		Secure: s.UseSSL,
		Region: s.Region,
	})
	if err != nil {
		logger.GetLogger().Errorf("Failed create minio client: %v", err)
		return nil, fmt.Errorf("Failed create minio client: %v", err)
	}
	return client, nil
}

func (s *S3Uploader) SimpleUpload(ctx context.Context, filePath, objectName string) (int64, error) {
	if err := s.ensureBucketExists(ctx); err != nil {
		logger.GetLogger().Errorf("Failed to verify if the bucket exists: %v", err)
		return 0, fmt.Errorf("Failed to verify if the bucket exists: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to open file %s: %v", filePath, err)
		return 0, fmt.Errorf("Failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("Failed to retrieve file information %s: %v", filePath, err)
	}
	totalSize := fileInfo.Size()

	// 上传文件
	uploadInfo, err := s.client.PutObject(ctx, s.BucketName, objectName, file, totalSize, minio.PutObjectOptions{})
	if err != nil {
		logger.GetLogger().Errorf("Failed to upload file %s: %v", filePath, err)
		return 0, fmt.Errorf("Failed to upload file %s: %v", filePath, err)
	}

	if s.notifyProgress != nil {
		s.notifyProgress(uploadInfo.Size, totalSize)
	}

	logger.GetLogger().Infof("Successfully to upload file %s to s3://%s%s", filePath, s.BucketName, objectName)
	return uploadInfo.Size, nil
}

// UploadDirectory 上传整个目录的所有文件
func (s *S3Uploader) UploadDirectory(ctx context.Context, localDir, remoteDir string) error {
	var wg sync.WaitGroup
	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 忽略目录本身，只上传文件
		if info.IsDir() {
			return nil
		}

		// 创建 MinIO 上的相对路径
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		objectName := filepath.Join(remoteDir, relPath)
		wg.Add(1)

		go func(filePath, objectName string) {
			defer wg.Done()
			if _, err := s.SimpleUpload(ctx, filePath, objectName); err != nil {
				log.Printf("上传失败: %v", err)
			}
		}(path, objectName)

		return nil
	})

	wg.Wait()

	return err
}

// downloadFile 下载文件
func (s *S3Uploader) Download(ctx context.Context, objectName, destPath string) error {
	if err := s.ensureBucketExists(ctx); err != nil {
		return fmt.Errorf("确保 bucket 存在失败: %v", err)
	}

	object, err := s.client.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("获取对象失败: %v", err)
	}
	defer object.Close()

	// 创建本地文件
	localFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer localFile.Close()

	// 直接复制数据
	_, err = io.Copy(localFile, object)
	if err != nil {
		return fmt.Errorf("复制数据失败: %w", err)
	}

	log.Printf("文件 %s 成功下载到 %s", objectName, destPath)
	return nil
}

// ensureBucketExists 确保 bucket 存在
func (s *S3Uploader) ensureBucketExists(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.BucketName)
	if err != nil {
		return fmt.Errorf("检查 bucket 是否存在失败: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.BucketName, minio.MakeBucketOptions{Region: s.Region}); err != nil {
			return fmt.Errorf("创建 bucket 失败: %w", err)
		}
		log.Printf("bucket %s 已创建", s.BucketName)
	}
	return nil
}
