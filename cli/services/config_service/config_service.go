package config_service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/s3_service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gopkg.in/yaml.v2"
)

var osReadFile = os.ReadFile
var osMkdirAll = os.MkdirAll
var osWriteFile = os.WriteFile
var initS3Client = initS3Service
var yamlMarshal = yaml.Marshal

type ConfigService struct{}

func (config ConfigService) GetBbeConfig(helperService interfaces.HelperServiceInterface) (*models.BbeConfig, error) {
	configDir := helperService.GetConfigDir()

	filePath := fmt.Sprintf("%s/%s", configDir, constants.BbeConfigFile)

	_, exists := helperService.CheckIfFileExists(filePath)
	if !exists {
		return nil, errors.New("Config file not found")
	}

	return config.readBbeConfig(helperService), nil
}

func (config ConfigService) GenerateBbeConfig(helperService interfaces.HelperServiceInterface, storage string) error {
	fileLocation := fmt.Sprintf("%s/%s", helperService.GetConfigDir(), constants.BbeConfigFile)
	_, exists := helperService.CheckIfFileExists(fileLocation)
	if exists {
		return nil
	}

	bbeConfig := &models.BbeConfig{}
	if storage == "local" {
		bbeConfig.Bbe.Storage.Type = "local"
	} else if storage == "aws" {
		client, err := initS3Client()
		if err != nil {
			return err
		}

		bbeConfig.Bbe.Storage.Type = "aws"
		bucketName, err := config.findOrCreateBucket(client)
		if err != nil {
			return err
		}
		bbeConfig.Bbe.Storage.Aws.BucketName = bucketName
	}

	return config.writeBbeConfig(helperService, bbeConfig)
}

func (config ConfigService) UpdateBbeClusterName(helperService interfaces.HelperServiceInterface, clusterName string) error {
	bbeConfig, err := config.GetBbeConfig(helperService)
	if err != nil {
		return err
	}

	bbeConfig.Bbe.Cluster.Name = clusterName
	bbeConfig.Bbe.Cluster.Context = fmt.Sprintf("admin@%s", clusterName)

	return config.writeBbeConfig(helperService, bbeConfig)
}

func (config ConfigService) UpdateBbeStorageType(helperService interfaces.HelperServiceInterface, storageType string) error {
	bbeConfig, err := config.GetBbeConfig(helperService)
	if err != nil {
		return err
	}

	bbeConfig.Bbe.Storage.Type = storageType

	return config.writeBbeConfig(helperService, bbeConfig)
}

func (config ConfigService) UpdateBbeAwsBucketName(helperService interfaces.HelperServiceInterface, bucketName string) error {
	bbeConfig, err := config.GetBbeConfig(helperService)
	if err != nil {
		return err
	}

	bbeConfig.Bbe.Storage.Aws.BucketName = bucketName

	return config.writeBbeConfig(helperService, bbeConfig)
}

func (config ConfigService) UpdateBbePackages(helperService interfaces.HelperServiceInterface, packages []models.LocalPackage) error {
	bbeConfig, err := config.GetBbeConfig(helperService)
	if err != nil {
		return err
	}

	bbeConfig.Bbe.Packages = packages

	return config.writeBbeConfig(helperService, bbeConfig)
}

func (config ConfigService) writeBbeConfig(helperService interfaces.HelperServiceInterface, bbeConfig *models.BbeConfig) error {
	fileLocation := fmt.Sprintf("%s/%s", helperService.GetConfigDir(), constants.BbeConfigFile)

	// Filter out any empty packages to avoid writing them to the config file
	var filteredPackages []models.LocalPackage
	for _, pkg := range bbeConfig.Bbe.Packages {
		if pkg.Name != "" || pkg.Version != "" {
			filteredPackages = append(filteredPackages, pkg)
		}
	}
	bbeConfig.Bbe.Packages = filteredPackages

	yamlFile, err := yamlMarshal(bbeConfig)
	if err != nil {
		return err
	}

	err = osMkdirAll(filepath.Dir(fileLocation), os.ModePerm)
	if err != nil {
		return err
	}

	err = osWriteFile(fileLocation, yamlFile, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (config ConfigService) CheckForTalosConfigs(helperService interfaces.HelperServiceInterface) bool {
	configDir := helperService.GetConfigDir()

	configFiles := []string{
		fmt.Sprintf("%s/%s", configDir, constants.TalosConfigFile),
		fmt.Sprintf("%s/%s", configDir, constants.ControlplaneConfigFile),
		fmt.Sprintf("%s/%s", configDir, constants.WorkerConfigFile),
	}

	for _, file := range configFiles {
		_, exists := helperService.CheckIfFileExists(file)
		if !exists {
			return false
		}
	}

	return true
}

func (config ConfigService) SyncConfigsWithAws(helperService interfaces.HelperServiceInterface, bbeConfig *models.BbeConfig) error {
	client, err := initS3Client()
	if err != nil {
		return err
	}

	if bbeConfig == nil || bbeConfig.Bbe.Storage.Aws.BucketName == "" {
		bucketName, err := config.findOrCreateBucket(client)
		if err != nil {
			return err
		}

		bbeConfig.Bbe.Storage.Aws.BucketName = bucketName
	}

	configFiles := []string{
		constants.BbeConfigFile,
		constants.TalosConfigFile,
		constants.ControlplaneConfigFile,
		constants.WorkerConfigFile,
	}

	for _, file := range configFiles {
		err := config.syncConfigFileWithAws(helperService, client, bbeConfig, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func (config ConfigService) findOrCreateBucket(client interfaces.S3ServiceInterface) (string, error) {
	ctx := context.Background()
	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return "", err
	}

	for _, bucket := range output.Buckets {
		bucketName := aws.ToString(bucket.Name)
		if len(bucketName) >= 10 && bucketName[:10] == "bbe-config" {
			logger.Infof("Found existing configuration on AWS: %s", *bucket.Name)
			return *bucket.Name, nil
		}
	}

	logger.Info("No existing configuration found on AWS, creating a new one")
	attempts := 0
	for attempts < 3 {
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		bucketName := "bbe-config-" + timestamp

		err = config.createS3Bucket(ctx, client, bucketName)
		if err != nil {
			logger.Error("Failed to create configuration bucket", err)
		}

		if err == nil {
			logger.Infof("Created new configuration bucket on AWS: %s", bucketName)
			return bucketName, nil
		}

		attempts++
	}

	return "", errors.New("failed to create AWS configuration after 3 attempts")
}

func (config ConfigService) readBbeConfig(helperService interfaces.HelperServiceInterface) *models.BbeConfig {
	configDir := helperService.GetConfigDir()

	file, err := osReadFile(fmt.Sprintf("%s/%s", configDir, constants.BbeConfigFile))
	if err != nil {
		panic(err)
	}

	var bbeConfig models.BbeConfig
	err = yaml.Unmarshal(file, &bbeConfig)
	if err != nil {
		panic(err)
	}

	return &bbeConfig
}

func (config ConfigService) syncConfigFileWithAws(helperService interfaces.HelperServiceInterface, client interfaces.S3ServiceInterface, bbeConfig *models.BbeConfig, name string) error {
	ctx := context.Background()

	filePath := fmt.Sprintf("%s/%s", helperService.GetConfigDir(), name)

	localModTime, exists := helperService.CheckIfFileExists(filePath)

	output, s3Err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bbeConfig.Bbe.Storage.Aws.BucketName),
		Key:    aws.String(name),
	})
	if s3Err != nil {
		var noSuchKey *types.NoSuchKey
		if !errors.As(s3Err, &noSuchKey) {
			return s3Err
		}
	} else {
		defer output.Body.Close()
	}

	// File exists neither locally nor in S3
	if !exists && s3Err != nil {
		logger.Infof("No local config file found and no config file found in AWS")
		return nil
	}

	// File exists in S3 but not locally
	if !exists && s3Err == nil {
		s3FileContents, err := io.ReadAll(output.Body)
		if err != nil {
			return err
		}

		err = osWriteFile(filePath, s3FileContents, 0644)
		if err != nil {
			return err
		}

		logger.Infof("Config file %s synced from AWS", name)
		return nil
	}

	// File exists locally but not in S3
	if exists && s3Err != nil {
		content, err := osReadFile(filePath)
		if err != nil {
			return err
		}
		_, err = client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bbeConfig.Bbe.Storage.Aws.BucketName),
			Key:    aws.String(name),
			Body:   bytes.NewReader(content),
		})
		if err != nil {
			return err
		}

		logger.Infof("Config file %s synced to AWS", name)
		return nil
	}

	// File exists both locally and in S3

	s3FileContents, err := io.ReadAll(output.Body)
	if err != nil {
		return err
	}

	content, s3Err := osReadFile(filePath)
	if s3Err != nil {
		return s3Err
	}

	if bytes.Equal(s3FileContents, content) {
		logger.Infof("Config file %s is already in sync", name)
		return nil
	}

	s3ModTime := output.LastModified

	if s3ModTime.After(*localModTime) {
		err = osWriteFile(filePath, s3FileContents, 0644)
		if err != nil {
			return err
		}

		logger.Infof("Config file %s on AWS was newer than the local copy and has been synced", name)
		return nil
	}

	_, s3Err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bbeConfig.Bbe.Storage.Aws.BucketName),
		Key:    aws.String(name),
		Body:   bytes.NewReader(content),
	})
	if s3Err != nil {
		return s3Err
	}

	logger.Infof("Local config file %s was newer than the copy on AWS and has been synced", name)
	return nil
}

func (config ConfigService) createS3Bucket(ctx context.Context, client interfaces.S3ServiceInterface, bucketName string) error {
	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint("eu-west-1"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	_, err = client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucketName),
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm: types.ServerSideEncryptionAes256,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to enable encryption: %w", err)
	}

	_, err = client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})

	if err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}

	return nil
}

func initS3Service() (interfaces.S3ServiceInterface, error) {
	return s3_service.Initialize(context.Background())
}
