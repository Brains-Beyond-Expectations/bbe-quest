package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/helper"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/talos"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gopkg.in/yaml.v3"
)

type BbeConfig struct {
	Bbe struct {
		Storage struct {
			Type string `yaml:"type"` // "local" or "aws"
			Aws  struct {
				BucketName string `yaml:"bucket_name"`
			} `yaml:"aws"`
		} `yaml:"storage"`
	} `yaml:"bbe"`
}

func GetBbeConfig() (*BbeConfig, error) {
	configDir := helper.GetConfigDir()

	configFiles := []string{
		fmt.Sprintf("%s/bbe.yaml", configDir),
	}

	for _, file := range configFiles {
		_, exists := helper.CheckIfFileExists(file)
		if !exists {
			return nil, errors.New("Config file not found")
		}
	}

	return readBbeConfig(), nil
}

func GenerateBbeConfig(storage string) error {
	fileLocation := fmt.Sprintf("%s/bbe.yaml", helper.GetConfigDir())
	_, exists := helper.CheckIfFileExists(fileLocation)
	if exists {
		return nil
	}

	bbeConfig := &BbeConfig{}
	if storage == "local" {
		bbeConfig.Bbe.Storage.Type = "local"
	} else if storage == "aws" {

		client, err := initS3Client()
		if err != nil {
			return err
		}

		bbeConfig, err = findOrCreateBucket(client, nil)
		if err != nil {
			return err
		}
	}

	_, err := UpdateBbeConfig(bbeConfig)
	return err
}

func UpdateBbeConfig(newConfig *BbeConfig) (*BbeConfig, error) {
	fileLocation := fmt.Sprintf("%s/bbe.yaml", helper.GetConfigDir())

	currentConfig, err := GetBbeConfig()
	if err != nil {
		currentConfig = &BbeConfig{}
		currentConfig.Bbe.Storage.Type = "local"
	}

	// Only update fields that are set in the new config
	if newConfig.Bbe.Storage.Type != "" {
		currentConfig.Bbe.Storage.Type = newConfig.Bbe.Storage.Type
	}
	if newConfig.Bbe.Storage.Aws.BucketName != "" {
		currentConfig.Bbe.Storage.Aws.BucketName = newConfig.Bbe.Storage.Aws.BucketName
	}

	// Write updated config back to file
	yamlFile, err := yaml.Marshal(currentConfig)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(fileLocation, yamlFile, 0644)
	if err != nil {
		return nil, err
	}

	return currentConfig, nil
}
func CheckForTalosConfigs() bool {
	configDir := helper.GetConfigDir()

	configFiles := []string{
		fmt.Sprintf("%s/talosconfig", configDir),
		fmt.Sprintf("%s/controlplane.yaml", configDir),
		fmt.Sprintf("%s/worker.yaml", configDir),
	}

	for _, file := range configFiles {
		_, exists := helper.CheckIfFileExists(file)
		if !exists {
			return false
		}
	}

	return true
}

func InitTalosConfig(controlPlaneIp string, clusterName string) error {
	err := talos.GenerateConfig(controlPlaneIp, clusterName)
	if err != nil {
		return errors.New("Error while generating config")
	}

	return nil
}

func SyncConfigsWithAws(bbeConfig *BbeConfig) error {
	client, err := initS3Client()
	if err != nil {
		return err
	}

	if bbeConfig == nil || bbeConfig.Bbe.Storage.Aws.BucketName == "" {
		bbeConfig, err = findOrCreateBucket(client, bbeConfig)
		if err != nil {
			return err
		}
	}

	configFiles := []string{
		"bbe.yaml",
		"talosconfig",
		"controlplane.yaml",
		"worker.yaml",
	}

	for _, file := range configFiles {
		err := syncConfigFileWithAws(client, bbeConfig, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func initS3Client() (*s3.Client, error) {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"))
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg), nil
}

func findOrCreateBucket(client *s3.Client, bbeConfig *BbeConfig) (*BbeConfig, error) {
	ctx := context.Background()
	output, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	for _, bucket := range output.Buckets {
		bucketName := aws.ToString(bucket.Name)
		if len(bucketName) >= 10 && bucketName[:10] == "bbe-config" {
			bbeConfig := &BbeConfig{}
			bbeConfig.Bbe.Storage.Type = "aws"
			bbeConfig.Bbe.Storage.Aws.BucketName = aws.ToString(bucket.Name)
			bbeConfig, err := UpdateBbeConfig(bbeConfig)
			if err != nil {
				return nil, err
			}
			logger.Infof("Found existing configuration on AWS: %s", bbeConfig.Bbe.Storage.Aws.BucketName)
			return bbeConfig, nil
		}
	}

	logger.Info("No existing configuration found on AWS, creating a new one")
	attempts := 0
	for attempts < 3 {
		timestamp := fmt.Sprintf("%d", time.Now().Unix())
		bucketName := "bbe-config-" + timestamp

		err = createS3Bucket(ctx, client, bucketName)
		if err != nil {
			logger.Error("Failed to create configuration file", err)
		}

		if err == nil {
			var err error
			if bbeConfig == nil {
				bbeConfig = &BbeConfig{}
				bbeConfig.Bbe.Storage.Type = "aws"
				bbeConfig.Bbe.Storage.Aws.BucketName = bucketName
				err = GenerateBbeConfig(bbeConfig.Bbe.Storage.Type)
				if err != nil {
					return nil, err
				}
			} else {
				fmt.Println("Hewwo?")
				bbeConfig.Bbe.Storage.Aws.BucketName = bucketName
				_, err := UpdateBbeConfig(bbeConfig)
				if err != nil {
					return nil, err
				}
			}

			logger.Infof("Created new configuration on AWS: %s", bucketName)
			return bbeConfig, nil
		}

		attempts++
	}

	return nil, errors.New("failed to create AWS configuration after 3 attempts")
}

func readBbeConfig() *BbeConfig {
	configDir := helper.GetConfigDir()

	file, err := os.ReadFile(fmt.Sprintf("%s/bbe.yaml", configDir))
	if err != nil {
		panic(err)
	}

	var bbeConfig BbeConfig
	err = yaml.Unmarshal(file, &bbeConfig)
	if err != nil {
		panic(err)
	}

	return &bbeConfig
}

func syncConfigFileWithAws(client *s3.Client, bbeConfig *BbeConfig, name string) error {
	ctx := context.Background()

	filePath := fmt.Sprintf("%s/%s", helper.GetConfigDir(), name)

	localModTime, exists := helper.CheckIfFileExists(filePath)

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
		return errors.New("No local config file found and no config file found in AWS")
	}

	// File exists in S3 but not locally
	if !exists && s3Err == nil {
		s3FileContents, err := io.ReadAll(output.Body)
		if err != nil {
			return err
		}

		err = os.WriteFile(filePath, s3FileContents, 0644)
		if err != nil {
			return err
		}

		logger.Infof("Config file %s synced from AWS", name)
		return nil
	}

	// File exists locally but not in S3
	if exists && s3Err != nil {
		content, err := os.ReadFile(filePath)
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

	content, s3Err := os.ReadFile(filePath)
	if s3Err != nil {
		return s3Err
	}

	if bytes.Equal(s3FileContents, content) {
		logger.Infof("Config file %s is already in sync", name)
		return nil
	}

	s3ModTime := output.LastModified
	if s3ModTime.After(*localModTime) {
		err = os.WriteFile(filePath, s3FileContents, 0644)
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

func createS3Bucket(ctx context.Context, client *s3.Client, bucketName string) error {
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
