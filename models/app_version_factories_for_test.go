// +build database

package models_test

import (
	"testing"

	"github.com/bitrise-io/addons-ship-backend/dataservices"
	"github.com/bitrise-io/addons-ship-backend/models"
	"github.com/c2fo/testify/require"
)

func createTestAppVersion(t *testing.T, appVersion *models.AppVersion) *models.AppVersion {
	err := dataservices.GetDB().Create(appVersion).Error
	require.NoError(t, err)
	return appVersion
}
