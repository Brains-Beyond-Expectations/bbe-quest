package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/nicolajv/bbe-quest/helper"
	"github.com/nicolajv/bbe-quest/services/talos"
	"gopkg.in/yaml.v3"
)

type BbeConfig struct {
	Bbe struct {
		Storage string `yaml:"storage"`
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
	fileContents := BbeConfig{
		Bbe: struct {
			Storage string `yaml:"storage"`
		}{
			Storage: storage,
		},
	}

	yamlFile, err := yaml.Marshal(&fileContents)
	if err != nil {
		panic(err)
	}

	return os.WriteFile(fmt.Sprintf("%s/bbe.yaml", helper.GetConfigDir()), yamlFile, 0644)
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

func SyncConfigsWithAws() error {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("eu-west-1"))
	if err != nil {
		return err
	}

	client := ssm.NewFromConfig(cfg)

	configFiles := []string{
		"bbe.yaml",
		"talosconfig",
		"controlplane.yaml",
		"worker.yaml",
	}

	for _, file := range configFiles {
		err := syncConfigFileWithAws(client, file)
		if err != nil {
			return err
		}
	}

	return nil
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

func syncConfigFileWithAws(client *ssm.Client, name string) error {
	ctx := context.Background()

	filePath := fmt.Sprintf("%s/%s", helper.GetConfigDir(), name)

	modTime, exists := helper.CheckIfFileExists(filePath)

	output, err := client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String("/bbe/" + name),
		WithDecryption: aws.Bool(true),
	})

	if err != nil && !exists {
		return err
	}

	if !exists && err != nil {
		return os.WriteFile(filePath, []byte(*output.Parameter.Value), 0644)
	}

	if exists && err != nil {
		content, err := os.ReadFile(filePath)
		if err != nil {
			return err
		}
		_, err = client.PutParameter(ctx, &ssm.PutParameterInput{
			Name:  aws.String("/bbe/" + name),
			Type:  types.ParameterTypeSecureString,
			Value: aws.String(string(content)),
			Tier:  types.ParameterTierAdvanced,
		})
		return err
	}

	parameterTime := output.Parameter.LastModifiedDate
	if parameterTime.After(*modTime) {
		return os.WriteFile(name, []byte(*output.Parameter.Value), 0644)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	_, err = client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String("/bbe/" + name),
		Type:      types.ParameterTypeSecureString,
		Value:     aws.String(string(content)),
		Overwrite: aws.Bool(true),
		Tier:      types.ParameterTierAdvanced,
	})
	return err
}
