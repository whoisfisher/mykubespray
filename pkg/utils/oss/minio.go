package oss

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/whoisfisher/mykubespray/pkg/logger"
	"io"
	"os"
	"path/filepath"
	"strings"
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
		logger.GetLogger().Errorf("Failed to init minio client: %v", err)
		return nil, fmt.Errorf("Failed to init minio client: %w", err)
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

func (s *S3Uploader) initClient() (*minio.Client, error) {
	client, err := minio.New(s.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.AccessKeyID, s.SecretAccessKey, ""),
		Secure: s.UseSSL,
		Region: s.Region,
	})
	if err != nil {
		logger.GetLogger().Errorf("Failed create minio client: %v", err)
		return nil, fmt.Errorf("Failed create minio client: %w", err)
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
		logger.GetLogger().Errorf("Failed to retrieve file information %s: %v", filePath, err)
		return 0, fmt.Errorf("Failed to retrieve file information %s: %w", filePath, err)
	}
	totalSize := fileInfo.Size()

	uploadInfo, err := s.client.PutObject(ctx, s.BucketName, objectName, file, totalSize, minio.PutObjectOptions{})
	if err != nil {
		logger.GetLogger().Errorf("Failed to upload file %s: %v", filePath, err)
		return 0, fmt.Errorf("Failed to upload file %s: %w", filePath, err)
	}

	if s.notifyProgress != nil {
		s.notifyProgress(uploadInfo.Size, totalSize)
	}

	logger.GetLogger().Infof("Successfully to upload file %s to s3://%s/%s%s", filePath, s.Endpoint, s.BucketName, objectName)
	return uploadInfo.Size, nil
}

func (s *S3Uploader) ChunkedUpload(ctx context.Context, filePath, objectName string) (int64, error) {
	if err := s.ensureBucketExists(ctx); err != nil {
		logger.GetLogger().Errorf("Failed to verify if the bucket exists: %v", err)
		return 0, fmt.Errorf("Failed to verify if the bucket exists: %w", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to open file %s: %v", filePath, err)
		return 0, fmt.Errorf("Failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		logger.GetLogger().Errorf("Failed to retrieve file information %s: %v", filePath, err)
		return 0, fmt.Errorf("Failed to retrieve file information %s: %w", filePath, err)
	}
	totalSize := fileInfo.Size()

	const chunkSize = 5 * 1024 * 1024

	numChunks := int(totalSize / chunkSize)
	if totalSize%chunkSize != 0 {
		numChunks++
	}

	var uploadedSize int64
	for i := 0; i < numChunks; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize
		if end > totalSize {
			end = totalSize
		}

		chunk := io.NewSectionReader(file, start, end-start)

		uploadInfo, err := s.client.PutObject(ctx, s.BucketName, objectName, chunk, end-start, minio.PutObjectOptions{})
		if err != nil {
			logger.GetLogger().Errorf("Failed to upload chunk %d of file %s: %v", i+1, filePath, err)
			return uploadedSize, fmt.Errorf("Failed to upload chunk %d of file %s: %w", i+1, filePath, err)
		}

		uploadedSize += uploadInfo.Size

		if s.notifyProgress != nil {
			s.notifyProgress(uploadedSize, totalSize)
		}

		logger.GetLogger().Infof("Successfully uploaded chunk %d of file %s to s3://%s/%s%s", i+1, filePath, s.Endpoint, s.BucketName, objectName)
	}

	logger.GetLogger().Infof("Successfully uploaded file %s to s3://%s/%s%s", filePath, s.Endpoint, s.BucketName, objectName)
	return uploadedSize, nil
}

func (s *S3Uploader) UploadDirectory(ctx context.Context, localDir, remoteDir string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var uploadErrors []error

	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		objectName := filepath.Join(remoteDir, relPath)
		wg.Add(1)

		go func(filePath, objectName string) {
			defer wg.Done()
			if _, err := s.SimpleUpload(ctx, filePath, objectName); err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Errorf("failed to upload %s: %v", filePath, err))
				mu.Unlock()
			}
		}(path, objectName)

		return nil
	})

	wg.Wait()

	if len(uploadErrors) > 0 {
		logger.GetLogger().Errorf("encountered errors during upload: %v", uploadErrors)
		return fmt.Errorf("encountered errors during upload: %w", uploadErrors)
	}

	return err
}

func (s *S3Uploader) SimpleDownload(ctx context.Context, objectName, destPath string) error {
	if err := s.ensureBucketExists(ctx); err != nil {
		logger.GetLogger().Errorf("Failed to verify if the bucket exists: %v", err)
		return fmt.Errorf("Failed to verify if the bucket exists: %w", err)
	}

	object, err := s.client.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		logger.GetLogger().Errorf("Failed to retrieve object %s: %w", objectName, err)
		return fmt.Errorf("Failed to retrieve object %s: %w", objectName, err)
	}
	defer object.Close()

	localFile, err := os.Create(destPath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create local file: %v", err)
		return fmt.Errorf("Failed to create local file: %w", err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, object)
	if err != nil {
		logger.GetLogger().Errorf("Failed to copy data: %v", err)
		return fmt.Errorf("Failed to copy data: %w", err)
	}

	logger.GetLogger().Infof("Successfully to download file %s to %s", objectName, destPath)
	return nil
}

func (s *S3Uploader) ChunkedDownload(ctx context.Context, objectName, destPath string) error {
	if err := s.ensureBucketExists(ctx); err != nil {
		logger.GetLogger().Errorf("Failed to verify if the bucket exists: %v", err)
		return fmt.Errorf("Failed to verify if the bucket exists: %w", err)
	}

	objectStat, err := s.client.StatObject(ctx, s.BucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		logger.GetLogger().Errorf("Failed to retrieve object %s: %v", objectStat, err)
		return fmt.Errorf("Failed to retrieve object %s: %w", objectStat, err)
	}

	partSize := int64(5 * 1024 * 1024)
	totalSize := objectStat.Size
	numParts := (totalSize + partSize - 1) / partSize

	localFile, err := os.Create(destPath)
	if err != nil {
		logger.GetLogger().Errorf("Failed to create local file: %v", err)
		return fmt.Errorf("Failed to create local file: %w", err)
	}
	defer localFile.Close()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var downloadErrors []error

	for i := int64(0); i < numParts; i++ {
		wg.Add(1)

		start := i * partSize
		end := start + partSize - 1
		if end > totalSize-1 {
			end = totalSize - 1
		}

		go func(start, end int64) {
			defer wg.Done()

			object, err := s.client.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
			if err != nil {
				mu.Lock()
				downloadErrors = append(downloadErrors, fmt.Errorf("Failed to retrieve chunk object : %w", err))
				mu.Unlock()
				return
			}
			defer object.Close()

			object.Seek(start, io.SeekStart)
			buf := make([]byte, end-start+1)
			_, err = object.Read(buf)
			if err != nil && err != io.EOF {
				mu.Lock()
				downloadErrors = append(downloadErrors, fmt.Errorf("Failed to read chunk object: %w", err))
				mu.Unlock()
				return
			}

			_, err = localFile.Write(buf)
			if err != nil {
				mu.Lock()
				downloadErrors = append(downloadErrors, fmt.Errorf("Failed to write chunk object: %w", err))
				mu.Unlock()
			}
		}(start, end)
	}

	wg.Wait()

	if len(downloadErrors) > 0 {
		return downloadErrors[0]
	}

	logger.GetLogger().Infof("Successfully to download file %s to %s", objectName, destPath)
	return nil
}

func (s *S3Uploader) DownloadDirectory(ctx context.Context, remoteDir, localDir string) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var downloadErrors []error

	if err := s.ensureBucketExists(ctx); err != nil {
		logger.GetLogger().Errorf("Failed to verify if the bucket exists: %v", err)
		return fmt.Errorf("Failed to verify if the bucket exists: %w", err)
	}

	objectCh := s.client.ListObjects(ctx, s.BucketName, minio.ListObjectsOptions{
		Prefix:    remoteDir,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			logger.GetLogger().Errorf("Failed to retrieve object list: %v", object.Err)
			return fmt.Errorf("Failed to retrieve object list: %w", object.Err)
		}

		relPath := strings.TrimPrefix(object.Key, remoteDir)
		localPath := filepath.Join(localDir, relPath)

		if err := os.MkdirAll(filepath.Dir(localPath), os.ModePerm); err != nil {
			logger.GetLogger().Errorf("Failed  to create directory: %v", err)
			return fmt.Errorf("Failed  to create directory: %w", err)
		}

		wg.Add(1)

		go func(objectName, localPath string) {
			defer wg.Done()

			if err := s.SimpleDownload(ctx, objectName, localPath); err != nil {
				mu.Lock()
				downloadErrors = append(downloadErrors, err)
				mu.Unlock()
			}
		}(object.Key, localPath)
	}

	wg.Wait()

	if len(downloadErrors) > 0 {
		return downloadErrors[0]
	}
	logger.GetLogger().Infof("Successfully to download all files to  %s", localDir)
	return nil
}

func (s *S3Uploader) ensureBucketExists(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.BucketName)
	if err != nil {
		logger.GetLogger().Errorf("Failed to verify if the bucket exists: %v", err)
		return fmt.Errorf("Failed to verify if the bucket exists: %w", err)
	}

	if !exists {
		if err := s.client.MakeBucket(ctx, s.BucketName, minio.MakeBucketOptions{Region: s.Region}); err != nil {
			logger.GetLogger().Errorf("Failed to create bucket %s: %v", s.BucketName, err)
			return fmt.Errorf("Failed to create bucket %s: %w", s.BucketName, err)
		}
		logger.GetLogger().Infof("Bucket %s has been created", s.BucketName)
	}
	logger.GetLogger().Infof("Bucket %s already exists", s.BucketName)
	return nil
}
