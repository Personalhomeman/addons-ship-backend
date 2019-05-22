package models_test

import (
	"testing"

	"github.com/bitrise-io/addons-ship-backend/models"
	"github.com/c2fo/testify/require"
	uuid "github.com/satori/go.uuid"
)

func Test_Screenshot_AWSPath(t *testing.T) {
	testScreenshot := models.Screenshot{
		Filename:   "screenshot1.png",
		DeviceType: "iPhone XS Max",
		ScreenSize: "6.5 inch",
		AppVersion: models.AppVersion{
			Record: models.Record{
				ID: uuid.FromStringOrNil("de438ddc-98e5-4226-a5f4-fd2d53474879"),
			},
			App: models.App{AppSlug: "test-app-slug"},
		},
	}

	require.Equal(t, "test-app-slug/de438ddc-98e5-4226-a5f4-fd2d53474879/iPhone XS Max (6.5 inch)/screenshot1.png", testScreenshot.AWSPath())
}
