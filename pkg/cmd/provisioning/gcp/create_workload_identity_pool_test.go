package gcp

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iam/v1"

	"github.com/openshift/cloud-credential-operator/pkg/cmd/provisioning"
	mockgcp "github.com/openshift/cloud-credential-operator/pkg/gcp/mock"
)

const (
	testName      = "test-name"
	testProject   = "test-project"
	testDirPrefix = "test-dir"
)

func TestCreateWorkloadIdentityPool(t *testing.T) {

	tests := []struct {
		name          string
		mockGCPClient func(mockCtrl *gomock.Controller) *mockgcp.MockClient
		setup         func(*testing.T) string
		verify        func(t *testing.T, tempDirName string)
		cleanup       func(*testing.T)
		generateOnly  bool
		expectError   bool
	}{
		{
			name: "Generate only",
			mockGCPClient: func(mockCtrl *gomock.Controller) *mockgcp.MockClient {
				mockGCPClient := mockgcp.NewMockClient(mockCtrl)
				return mockGCPClient
			},
			setup: func(t *testing.T) string {
				tempDirName, err := ioutil.TempDir(os.TempDir(), testDirPrefix)
				require.NoError(t, err, "Failed to create temp directory")
				return tempDirName
			},
			verify: func(t *testing.T, targetDir string) {
				files, err := ioutil.ReadDir(targetDir)
				require.NoError(t, err, "Unexpected error listing files in targetDir")
				assert.Equal(t, 1, provisioning.CountNonDirectoryFiles(files), "Should be exactly 1 shell script")
			},
			generateOnly: true,
			expectError:  false,
		},
		{
			name: "Success creating workload identity pool",
			mockGCPClient: func(mockCtrl *gomock.Controller) *mockgcp.MockClient {
				mockGCPClient := mockgcp.NewMockClient(mockCtrl)
				mockCreateWorkloadIdentityPoolSuccess(mockGCPClient)
				return mockGCPClient
			},
			setup: func(t *testing.T) string {
				tempDirName, err := ioutil.TempDir(os.TempDir(), testDirPrefix)
				require.NoError(t, err, "Failed to create temp directory")
				return tempDirName
			},
			verify: func(t *testing.T, targetDir string) {
				files, err := ioutil.ReadDir(targetDir)
				require.NoError(t, err, "Unexpected error listing files in targetDir")
				assert.Zero(t, provisioning.CountNonDirectoryFiles(files), "Should be no generated files when not in generate mode")
			},
			generateOnly: false,
			expectError:  false,
		},
		{
			name: "Failure creating workload identity pool",
			mockGCPClient: func(mockCtrl *gomock.Controller) *mockgcp.MockClient {
				mockGCPClient := mockgcp.NewMockClient(mockCtrl)
				mockCreateWorkloadIdentityPoolFailure(mockGCPClient)
				return mockGCPClient
			},
			setup: func(t *testing.T) string {
				tempDirName, err := ioutil.TempDir(os.TempDir(), testDirPrefix)
				require.NoError(t, err, "Failed to create temp directory")
				return tempDirName
			},
			verify:       func(t *testing.T, targetDir string) {},
			generateOnly: false,
			expectError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockGCPClient := test.mockGCPClient(mockCtrl)

			tempDirName := test.setup(t)
			defer os.RemoveAll(tempDirName)

			err := createWorkloadIdentityPool(context.TODO(), mockGCPClient, testName, testProject, tempDirName, test.generateOnly)

			if test.expectError {
				assert.Error(t, err, "expected error returned")
			} else {
				assert.NoError(t, err, "unexpected error")
			}

			test.verify(t, tempDirName)
		})
	}
}

func mockCreateWorkloadIdentityPoolSuccess(mockGCPClient *mockgcp.MockClient) {
	mockGCPClient.EXPECT().CreateWorkloadIdentityPool(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		&iam.Operation{
			Done: true,
		}, nil).Times(1)
}

func mockCreateWorkloadIdentityPoolFailure(mockGCPClient *mockgcp.MockClient) {
	mockGCPClient.EXPECT().CreateWorkloadIdentityPool(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(
		nil,
		&googleapi.Error{
			Code:    409,
			Message: "Requested entity already exists, alreadyExists",
		}).Times(1)
}
