package models_test

import (
	"encoding/json"
	"testing"

	"github.com/bitrise-io/addons-ship-backend/models"
	"github.com/c2fo/testify/require"
)

func Test_AppVersion_AppStoreInfo(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		testAppVersion := &models.AppVersion{AppStoreInfoData: json.RawMessage(`{"short_description":"Some shorter description"}`)}
		appStoreInfo, err := testAppVersion.AppStoreInfo()
		require.NoError(t, err)
		require.Equal(t, models.AppStoreInfo{ShortDescription: "Some shorter description"}, appStoreInfo)
	})

	t.Run("error unmarshaling store info", func(t *testing.T) {
		testAppVersion := &models.AppVersion{}
		appStoreInfo, err := testAppVersion.AppStoreInfo()
		require.EqualError(t, err, "unexpected end of JSON input")
		require.Equal(t, models.AppStoreInfo{}, appStoreInfo)
	})
}

func Test_AppVersion_ArtifactInfo(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		testAppVersion := &models.AppVersion{ArtifactInfoData: json.RawMessage(`{"minimum_os":"11.0"}`)}
		artifactInfo, err := testAppVersion.ArtifactInfo()
		require.NoError(t, err)
		require.Equal(t, models.ArtifactInfo{MinimumOS: "11.0"}, artifactInfo)
	})

	t.Run("error unmarshaling store info", func(t *testing.T) {
		testAppVersion := &models.AppVersion{}
		artifactInfo, err := testAppVersion.ArtifactInfo()
		require.EqualError(t, err, "unexpected end of JSON input")
		require.Equal(t, models.ArtifactInfo{}, artifactInfo)
	})
}
