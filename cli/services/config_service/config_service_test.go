package config_service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v2"
)

func Test_GetBbeConfig_Succeeds(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	mockOs := &mocks.MockOs{}
	osReadFile = mockOs.ReadFile
	mockOs.On("ReadFile", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return([]byte{}, nil)

	config, err := configService.GetBbeConfig(mockHelperService)

	assert.NotNil(t, config)
	assert.NoError(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 2)
}

func Test_GetBbeConfig_Fails_IfYouDoNotHaveBbeConfig(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	config, err := configService.GetBbeConfig(mockHelperService)

	assert.Nil(t, config)
	assert.Error(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_GenerateBbeConfig_Succeeds_ReturnsOnExistingConfig(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, true)
	mockHelperService.On("GetConfigDir").Return("")

	err := configService.GenerateBbeConfig(mockHelperService, "local")

	assert.NoError(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_GenerateBbeConfig_Succeeds_CreatesLocalConfig(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	mockOs := &mocks.MockOs{}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile

	err := configService.GenerateBbeConfig(mockHelperService, "local")

	assert.NoError(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 2)
}

func Test_GenerateBbeConfig_Succeeds_CreatesAwsConfig_WithNoExistingBucket(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	mockS3Service := &mocks.MockS3Service{}
	mockS3Service.On("ListBuckets", mock.Anything, mock.Anything, mock.Anything).Return(&s3.ListBucketsOutput{}, nil)
	mockS3Service.On("CreateBucket", mock.Anything, mock.Anything, mock.Anything).Return(&s3.CreateBucketOutput{}, nil)
	mockS3Service.On("PutBucketEncryption", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutBucketEncryptionOutput{}, nil)
	mockS3Service.On("PutBucketVersioning", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutBucketVersioningOutput{}, nil)

	initS3Client = func() (interfaces.S3ServiceInterface, error) {
		return mockS3Service, nil
	}

	mockOs := &mocks.MockOs{}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile

	err := configService.GenerateBbeConfig(mockHelperService, "aws")

	assert.NoError(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 2)

	mockS3Service.AssertNumberOfCalls(t, "ListBuckets", 1)
	mockS3Service.AssertNumberOfCalls(t, "CreateBucket", 1)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketEncryption", 1)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketVersioning", 1)
}

func Test_GenerateBbeConfig_Succeeds_CreatesAwsConfig_WithExistingBucket(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	mockS3Service := &mocks.MockS3Service{}
	mockS3Service.On("ListBuckets", mock.Anything, mock.Anything, mock.Anything).Return(&s3.ListBucketsOutput{
		Buckets: []types.Bucket{{Name: aws.String("bbe-config-1738850879")}},
	}, nil)

	initS3Client = func() (interfaces.S3ServiceInterface, error) {
		return mockS3Service, nil
	}

	mockOs := &mocks.MockOs{}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile

	err := configService.GenerateBbeConfig(mockHelperService, "aws")

	assert.NoError(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 2)

	mockS3Service.AssertNumberOfCalls(t, "ListBuckets", 1)
	mockS3Service.AssertNumberOfCalls(t, "CreateBucket", 0)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketEncryption", 0)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketVersioning", 0)
}

func Test_GenerateBbeConfig_Fails_WithAwsConfigWithNoAwsCredentials(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	mockS3Service := &mocks.MockS3Service{}
	mockS3Service.On("ListBuckets", mock.Anything, mock.Anything, mock.Anything).Return(&s3.ListBucketsOutput{}, nil)

	initS3Client = func() (interfaces.S3ServiceInterface, error) {
		return nil, errors.New("test error")
	}

	err := configService.GenerateBbeConfig(mockHelperService, "aws")

	assert.Error(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_GenerateBbeConfig_Fails_WhenFailingToQueryS3(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	mockS3Service := &mocks.MockS3Service{}
	mockS3Service.On("ListBuckets", mock.Anything, mock.Anything, mock.Anything).Return(&s3.ListBucketsOutput{}, errors.New("test error"))

	initS3Client = func() (interfaces.S3ServiceInterface, error) {
		return mockS3Service, nil
	}

	err := configService.GenerateBbeConfig(mockHelperService, "aws")

	assert.Error(t, err)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_UpdateBbeClusterName_Succeeds(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	mockOs := &mocks.MockOs{}
	config := models.BbeConfig{}
	yamlFile, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	config.Bbe.Cluster.Name = "testCluster"
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOs.On("ReadFile", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(yamlFile, nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile
	osReadFile = mockOs.ReadFile

	err = configService.UpdateBbeClusterName(mockHelperService, "clusterName")

	assert.NoError(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 3)
}

func Test_UpdateBbeClusterName_Fails_WhenUnableToGetBbeConfig(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	err := configService.UpdateBbeClusterName(mockHelperService, "clusterName")

	assert.Error(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_UpdateBbeStorageType_Succeeds(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	mockOs := &mocks.MockOs{}
	config := models.BbeConfig{}
	config.Bbe.Storage.Type = "testStorage"
	yamlFile, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOs.On("ReadFile", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(yamlFile, nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile
	osReadFile = mockOs.ReadFile

	err = configService.UpdateBbeStorageType(mockHelperService, "storageType")

	assert.NoError(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 3)
}

func Test_UpdateBbeStorageType_Fails_WhenUnableTo_GetBbeConfig(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	err := configService.UpdateBbeStorageType(mockHelperService, "storageType")

	assert.Error(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_UpdateBbeAwsBucketName_Succeeds(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	mockOs := &mocks.MockOs{}
	config := models.BbeConfig{}
	config.Bbe.Storage.Aws.BucketName = "testBucket"
	yamlFile, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOs.On("ReadFile", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(yamlFile, nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile
	osReadFile = mockOs.ReadFile

	err = configService.UpdateBbeAwsBucketName(mockHelperService, "bucketName")

	assert.NoError(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 3)
}

func Test_UpdateBbeAwsBucketName_Fails_WhenUnableTo_GetBbeConfig(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	err := configService.UpdateBbeAwsBucketName(mockHelperService, "bucketName")

	assert.Error(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_UpdateBbePackages_Succeeds(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	mockOs := &mocks.MockOs{}
	config := models.BbeConfig{}
	config.Bbe.Packages = []models.LocalPackage{{Name: "package1"}, {Name: "package2"}}
	config.Bbe.Storage.Aws.BucketName = "testBucket"
	yamlFile, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOs.On("ReadFile", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(yamlFile, nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile
	osReadFile = mockOs.ReadFile

	err = configService.UpdateBbePackages(mockHelperService, []models.LocalPackage{{Name: "package1"}, {Name: "package2"}})

	assert.NoError(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 3)
}

func Test_UpdateBbePackages_Fails_WhenUnableTo_GetBbeConfig(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(nil, false)
	mockHelperService.On("GetConfigDir").Return("")

	err := configService.UpdateBbePackages(mockHelperService, []models.LocalPackage{{Name: "package1"}, {Name: "package2"}})

	assert.Error(t, err)
	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_CheckForTalosConfigs_Succeeds_WithAllFilesExisting(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.TalosConfigFile)).Return(&now, true)
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.ControlplaneConfigFile)).Return(&now, true)
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.WorkerConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	exists := configService.CheckForTalosConfigs(mockHelperService)

	assert.True(t, exists)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 3)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_CheckForTalosConfigs_Succeeds_WithAllFilesMissing(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.TalosConfigFile)).Return(&now, false)
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.ControlplaneConfigFile)).Return(&now, false)
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.WorkerConfigFile)).Return(&now, false)
	mockHelperService.On("GetConfigDir").Return("")

	exists := configService.CheckForTalosConfigs(mockHelperService)

	assert.False(t, exists)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 1)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_CheckForTalosConfigs_Succeeds_WithOneFileMissing(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.TalosConfigFile)).Return(&now, true)
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.ControlplaneConfigFile)).Return(&now, false)
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.WorkerConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	exists := configService.CheckForTalosConfigs(mockHelperService)

	assert.False(t, exists)

	mockHelperService.AssertNumberOfCalls(t, "CheckIfFileExists", 2)
	mockHelperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_SyncConfigsWithAws_Succeeds_WithConfigsOnlyLocally(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("")

	mockS3Service := &mocks.MockS3Service{}
	mockS3Service.On("ListBuckets", mock.Anything, mock.Anything, mock.Anything).Return(&s3.ListBucketsOutput{}, nil)
	mockS3Service.On("CreateBucket", mock.Anything, mock.Anything, mock.Anything).Return(&s3.CreateBucketOutput{}, nil)
	mockS3Service.On("PutBucketEncryption", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutBucketEncryptionOutput{}, nil)
	mockS3Service.On("PutBucketVersioning", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutBucketVersioningOutput{}, nil)

	noSuchKeyError := &types.NoSuchKey{}
	mockS3Service.On(("GetObject"), mock.Anything, mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{}, noSuchKeyError)

	mockS3Service.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

	initS3Client = func() (interfaces.S3ServiceInterface, error) {
		return mockS3Service, nil
	}

	mockOs := &mocks.MockOs{}
	config := models.BbeConfig{}
	yamlFile, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	mockOs.On("ReadFile", mock.Anything).Return(yamlFile, nil)
	osWriteFile = mockOs.WriteFile
	osReadFile = mockOs.ReadFile

	err = configService.SyncConfigsWithAws(mockHelperService, &config)

	assert.NoError(t, err)

	mockS3Service.AssertNumberOfCalls(t, "ListBuckets", 1)
	mockS3Service.AssertNumberOfCalls(t, "CreateBucket", 1)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketEncryption", 1)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketVersioning", 1)

	mockS3Service.AssertNumberOfCalls(t, "GetObject", 4)
	mockS3Service.AssertNumberOfCalls(t, "PutObject", 4)

	mockOs.AssertNumberOfCalls(t, "ReadFile", 4)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 0)
}

func Test_SyncConfigsWithAws_Succeeds_WithConfigsRemotely(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&now, false)
	mockHelperService.On("GetConfigDir").Return("")

	mockS3Service := &mocks.MockS3Service{}
	mockS3Service.On("ListBuckets", mock.Anything, mock.Anything, mock.Anything).Return(&s3.ListBucketsOutput{
		Buckets: []types.Bucket{{Name: aws.String("bbe-config-1738850879")}},
	}, nil)

	s3ObjectOutput := &s3.GetObjectOutput{}
	s3ObjectOutput.Body = io.NopCloser(bytes.NewReader([]byte{}))
	mockS3Service.On(("GetObject"), mock.Anything, mock.Anything, mock.Anything).Return(s3ObjectOutput, nil)
	mockS3Service.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

	initS3Client = func() (interfaces.S3ServiceInterface, error) {
		return mockS3Service, nil
	}

	mockOs := &mocks.MockOs{}
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osWriteFile = mockOs.WriteFile

	config := models.BbeConfig{}
	err := configService.SyncConfigsWithAws(mockHelperService, &config)

	assert.NoError(t, err)

	mockS3Service.AssertNumberOfCalls(t, "ListBuckets", 1)
	mockS3Service.AssertNumberOfCalls(t, "CreateBucket", 0)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketEncryption", 0)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketVersioning", 0)

	mockS3Service.AssertNumberOfCalls(t, "GetObject", 4)
	mockS3Service.AssertNumberOfCalls(t, "PutObject", 0)

	mockOs.AssertNumberOfCalls(t, "ReadFile", 0)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 4)
}
func Test_SyncConfigsWithAws_Succeeds_WithMismatchedTimestamps(t *testing.T) {
	configService := ConfigService{}

	mockHelperService := &mocks.MockHelperService{}
	yesterday := time.Now().Add(-time.Hour * 24)
	tomorrow := time.Now().Add(time.Hour * 24)
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&yesterday, true).Once()
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&tomorrow, true).Once()
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&yesterday, true).Once()
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&tomorrow, true).Once()
	mockHelperService.On("GetConfigDir").Return("")

	mockS3Service := &mocks.MockS3Service{}
	mockS3Service.On("ListBuckets", mock.Anything, mock.Anything, mock.Anything).Return(&s3.ListBucketsOutput{
		Buckets: []types.Bucket{{Name: aws.String("bbe-config-1738850879")}},
	}, nil)

	s3ObjectOutput := &s3.GetObjectOutput{}
	s3ObjectOutput.Body = io.NopCloser(bytes.NewReader([]byte{}))
	s3ObjectOutput.LastModified = aws.Time(yesterday)
	mockS3Service.On(("GetObject"), mock.Anything, mock.Anything, mock.Anything).Return(s3ObjectOutput, nil).Once()
	s3ObjectOutput.LastModified = aws.Time(tomorrow)
	mockS3Service.On(("GetObject"), mock.Anything, mock.Anything, mock.Anything).Return(s3ObjectOutput, nil).Once()
	s3ObjectOutput.LastModified = aws.Time(yesterday)
	mockS3Service.On(("GetObject"), mock.Anything, mock.Anything, mock.Anything).Return(s3ObjectOutput, nil).Once()
	s3ObjectOutput.LastModified = aws.Time(tomorrow)
	mockS3Service.On(("GetObject"), mock.Anything, mock.Anything, mock.Anything).Return(s3ObjectOutput, nil).Once()
	mockS3Service.On("PutObject", mock.Anything, mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

	initS3Client = func() (interfaces.S3ServiceInterface, error) {
		return mockS3Service, nil
	}

	mockOs := &mocks.MockOs{}
	yamlFile, err := yaml.Marshal(models.BbeConfig{})
	if err != nil {
		panic(err)
	}
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOs.On("ReadFile", mock.Anything).Return(yamlFile, nil)
	osWriteFile = mockOs.WriteFile
	osReadFile = mockOs.ReadFile

	config := models.BbeConfig{}
	err = configService.SyncConfigsWithAws(mockHelperService, &config)

	assert.NoError(t, err)

	mockS3Service.AssertNumberOfCalls(t, "ListBuckets", 1)
	mockS3Service.AssertNumberOfCalls(t, "CreateBucket", 0)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketEncryption", 0)
	mockS3Service.AssertNumberOfCalls(t, "PutBucketVersioning", 0)

	mockS3Service.AssertNumberOfCalls(t, "GetObject", 4)
	mockS3Service.AssertNumberOfCalls(t, "PutObject", 2)

	mockOs.AssertNumberOfCalls(t, "ReadFile", 4)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 2)
}

func Test_WriteBbeConfig_Fails_On_MkDir(t *testing.T) {
	// Mock HelperServiceInterface
	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("/mock/config/dir")

	// Mock the os functions to avoid actual file system changes
	mockOs := &mocks.MockOs{}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(errors.New("Fail on folder creation"))
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile

	// Create the configService instance
	configService := ConfigService{}

	// Prepare the bbeConfig with non-empty and empty packages
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
	}

	// Call the writeBbeConfig method
	err := configService.writeBbeConfig(mockHelperService, bbeConfig)

	// Assert no error occurred
	assert.Error(t, err)

	// Check that the necessary function calls were made
	mockOs.AssertNumberOfCalls(t, "MkdirAll", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 0)
}

func Test_WriteBbeConfig_Fails_On_OsWrite(t *testing.T) {
	// Mock HelperServiceInterface
	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("/mock/config/dir")

	// Mock the os functions to avoid actual file system changes
	mockOs := &mocks.MockOs{}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("Fail on OS writing"))
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile

	// Create the configService instance
	configService := ConfigService{}

	// Prepare the bbeConfig with non-empty and empty packages
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
	}

	// Call the writeBbeConfig method
	err := configService.writeBbeConfig(mockHelperService, bbeConfig)

	// Assert no error occurred
	assert.Error(t, err)

	// Check that the necessary function calls were made
	mockOs.AssertNumberOfCalls(t, "MkdirAll", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)
}

func Test_WriteBbeConfig_Fails_On_Yaml_Marshal(t *testing.T) {
	// Mock HelperServiceInterface
	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("CheckIfFileExists", fmt.Sprintf("/%s", constants.BbeConfigFile)).Return(&now, true)
	mockHelperService.On("GetConfigDir").Return("/mock/config/dir")

	// Mock the os functions to avoid actual file system changes
	mockOs := &mocks.MockOs{}
	mockOs.On("MkdirAll", mock.Anything, mock.Anything).Return(nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOs.On("YamlMarshal", mock.Anything).Return([]byte(""), errors.New("Marshal has failed"))
	osMkdirAll = mockOs.MkdirAll
	osWriteFile = mockOs.WriteFile
	yamlMarshal = mockOs.YamlMarshal

	// Create the configService instance
	configService := ConfigService{}

	// Prepare the bbeConfig with non-empty and empty packages
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
	}

	// Call the writeBbeConfig method
	err := configService.writeBbeConfig(mockHelperService, bbeConfig)

	// Assert no error occurred
	assert.Error(t, err)

	// Check that the necessary function calls were made
	mockOs.AssertNumberOfCalls(t, "YamlMarshal", 1)
	mockOs.AssertNumberOfCalls(t, "MkdirAll", 0)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 0)
}
