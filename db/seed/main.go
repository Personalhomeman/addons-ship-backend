package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/bitrise-io/addons-ship-backend/dataservices"
	"github.com/bitrise-io/addons-ship-backend/models"
	"github.com/go-yaml/yaml"
	uuid "github.com/satori/go.uuid"
)

func main() {
	err := dataservices.InitializeConnection(dataservices.ConnectionParams{}, true)
	if err != nil {
		fmt.Printf("Failed to initialize: %#v", err)
	}
	defer dataservices.Close()

	db := dataservices.GetDB()

	var testData testData

	yamlFile, err := ioutil.ReadFile("test_data.yml")
	if err != nil {
		log.Printf("Failed to read test_data.yml: %s", err)
	}
	err = yaml.Unmarshal(yamlFile, &testData)
	if err != nil {
		log.Fatalf("Failed to parse test_data.yml: %v", err)
	}

	db.Exec("TRUNCATE TABLE apps CASCADE")

	// create test apps
	for _, appData := range testData.Apps {
		app := models.App{
			Record:          models.Record{ID: appData.ID},
			AppSlug:         appData.Slug,
			APIToken:        appData.APIToken,
			BitriseAPIToken: appData.BitriseAPIToken,
			Plan:            appData.Plan,
		}
		if err := db.Create(&app).Error; err != nil {
			fmt.Printf("Failed to seed db with app: %#v, app: %#v", err, app)
			os.Exit(1)
		}
		app.AppSettings.App = &app
		if err := db.Create(&app.AppSettings).Error; err != nil {
			fmt.Printf("Failed to create app setting for app at seeding: %#v, app: %#v", err, app)
			os.Exit(1)
		}
	}

	// create test app versions
	for _, appVersionData := range testData.AppVersions {
		appStoreInfoBytes, err := json.Marshal(appVersionData.AppStoreInfo)
		if err != nil {
			fmt.Printf("Failed to marshal app store info: %#v, app store info: %#v", err, appVersionData.AppStoreInfo)
			os.Exit(1)
		}
		artifactInfoBytes, err := json.Marshal(models.ArtifactInfo{Version: appVersionData.Version})
		if err != nil {
			fmt.Printf("Failed to marshal srtifact info: %#v, app version data: %#v", err, appVersionData)
			os.Exit(1)
		}
		appVersion := models.AppVersion{
			Record:           models.Record{ID: appVersionData.ID},
			AppID:            appVersionData.AppID,
			Platform:         appVersionData.Platform,
			BuildSlug:        appVersionData.BuildSlug,
			BuildNumber:      appVersionData.BuildNumber,
			LastUpdate:       appVersionData.LastUpdate,
			Scheme:           appVersionData.Scheme,
			Configuration:    appVersionData.Configuration,
			AppStoreInfoData: appStoreInfoBytes,
			ArtifactInfoData: artifactInfoBytes,
		}
		if err := db.Create(&appVersion).Error; err != nil {
			fmt.Printf("Failed to seed db with app version: %#v, app version: %#v", err, appVersion)
			os.Exit(1)
		}
	}

	// create test screenshots
	for _, screenshotData := range testData.Screenshots {
		screenshot := models.Screenshot{
			Record: models.Record{ID: screenshotData.ID},
			UploadableObject: models.UploadableObject{
				Filename: screenshotData.Filename,
				Filesize: screenshotData.Filesize,
			},
			AppVersionID: screenshotData.AppVersionID,
			DeviceType:   screenshotData.DeviceType,
			ScreenSize:   screenshotData.ScreenSize,
		}
		if err := db.Create(&screenshot).Error; err != nil {
			fmt.Printf("Failed to seed db with screenshot: %#v, screenshot: %#v", err, screenshot)
			os.Exit(1)
		}
	}

	// create test feature graphics
	for _, featureGraphicData := range testData.FeatureGraphics {
		featureGraphic := models.FeatureGraphic{
			Record: models.Record{ID: featureGraphicData.ID},
			UploadableObject: models.UploadableObject{
				Filename: featureGraphicData.Filename,
				Filesize: featureGraphicData.Filesize,
			},
			AppVersionID: featureGraphicData.AppVersionID,
		}
		if err := db.Create(&featureGraphic).Error; err != nil {
			fmt.Printf("Failed to seed db with feawture graphic: %#v, feature graphic: %#v", err, featureGraphic)
			os.Exit(1)
		}
	}

	// create test app version events
	for _, appVersionEventData := range testData.AppVersionEvents {
		appEvent := models.AppVersionEvent{
			Record:       models.Record{ID: appVersionEventData.ID, CreatedAt: appVersionEventData.CreatedAt},
			Status:       appVersionEventData.Status,
			Text:         appVersionEventData.Text,
			AppVersionID: appVersionEventData.AppVersionID,
		}

		if err := db.Create(&appEvent).Error; err != nil {
			fmt.Printf("Failed to seed db with app event: %#v, app event: %#v", err, appEvent)
			os.Exit(1)
		}
	}

	// create test app contacts
	for _, appContactData := range testData.AppContacts {
		notificationPreferenceBytes, err := json.Marshal(models.NotificationPreferences(appContactData.NotificationPreferencesData))
		if err != nil {
			fmt.Printf("Failed to marshal notification preferences: %#v, app concat data: %#v", err, appContactData)
			os.Exit(1)
		}

		appContact := models.AppContact{
			Record:                      models.Record{ID: appContactData.ID},
			AppID:                       appContactData.AppID,
			Email:                       appContactData.Email,
			ConfirmedAt:                 appContactData.ConfirmedAt,
			ConfirmationToken:           appContactData.ConfirmationToken,
			NotificationPreferencesData: notificationPreferenceBytes,
		}

		if err := db.Create(&appContact).Error; err != nil {
			fmt.Printf("Failed to seed db with app contact: %#v, app contact: %#v", err, appContact)
			os.Exit(1)
		}
	}
}

type app struct {
	ID              uuid.UUID `yaml:"id"`
	Slug            string    `yaml:"slug"`
	Plan            string    `yaml:"plan"`
	BitriseAPIToken string    `yaml:"bitrise_api_token"`
	APIToken        string    `yaml:"api_token"`
}

type appStoreInfo struct {
	ShortDescription string `yaml:"short_description"`
	FullDescription  string `yaml:"full_description"`
	WhatsNew         string `yaml:"whats_new"`
	PromotionalText  string `yaml:"promotional_text"`
	Keywords         string `yaml:"keywords"`
	ReviewNotes      string `yaml:"review_notes"`
	SupportURL       string `yaml:"support_url"`
	MarketingURL     string `yaml:"marketing_url"`
}

type appVersion struct {
	ID            uuid.UUID    `yaml:"id"`
	AppID         uuid.UUID    `yaml:"app_id"`
	Version       string       `yaml:"version"`
	Platform      string       `yaml:"platform"`
	BuildNumber   string       `yaml:"build_number"`
	BuildSlug     string       `yaml:"build_slug"`
	LastUpdate    time.Time    `yaml:"last_update"`
	Scheme        string       `yaml:"scheme"`
	Configuration string       `yaml:"configuration"`
	AppStoreInfo  appStoreInfo `yaml:"app_store_info"`
}

type appVersionEvent struct {
	ID           uuid.UUID `yaml:"id"`
	AppVersionID uuid.UUID `yaml:"app_version_id"`
	Status       string    `yaml:"status"`
	Text         string    `yaml:"event_text"`
	CreatedAt    time.Time `yaml:"created_at"`
}

type screenshot struct {
	ID           uuid.UUID `yaml:"id"`
	AppVersionID uuid.UUID `yaml:"app_version_id"`
	Filename     string    `yaml:"filename"`
	Filesize     int64     `yaml:"filesize"`
	Uploaded     bool      `yaml:"uploaded"`
	DeviceType   string    `yaml:"device_type"`
	ScreenSize   string    `yaml:"screen_size"`
}

type featureGraphic struct {
	ID           uuid.UUID `yaml:"id"`
	AppVersionID uuid.UUID `yaml:"app_version_id"`
	Filename     string    `yaml:"filename"`
	Filesize     int64     `yaml:"filesize"`
	Uploaded     bool      `yaml:"uploaded"`
}

type notificationPreferences struct {
	NewVersion        bool `yaml:"new_version"`
	SuccessfulPublish bool `yaml:"successful_publish"`
	FailedPublish     bool `yaml:"failed_publish"`
}

type appContact struct {
	ID                          uuid.UUID               `yaml:"id"`
	AppID                       uuid.UUID               `yaml:"app_id"`
	Email                       string                  `yaml:"email"`
	NotificationPreferencesData notificationPreferences `yaml:"notification_preferences"`
	ConfirmedAt                 time.Time               `yaml:"confirmed_at"`
	ConfirmationToken           *string                 `yaml:"confirmation_token"`
}

type testData struct {
	Apps             []app             `yaml:"apps"`
	AppVersions      []appVersion      `yaml:"app_versions"`
	Screenshots      []screenshot      `yaml:"screenshots"`
	FeatureGraphics  []featureGraphic  `yaml:"feature_graphics"`
	AppVersionEvents []appVersionEvent `yaml:"app_version_events"`
	AppContacts      []appContact      `yaml:"app_contacts"`
}
