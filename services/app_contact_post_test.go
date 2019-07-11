package services_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bitrise-io/addons-ship-backend/bitrise"
	"github.com/bitrise-io/addons-ship-backend/env"
	"github.com/bitrise-io/addons-ship-backend/models"
	"github.com/bitrise-io/addons-ship-backend/services"
	ctxpkg "github.com/bitrise-io/api-utils/context"
	"github.com/c2fo/testify/require"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

func Test_AppContactPost(t *testing.T) {
	httpMethod := "POST"
	url := "/apps/{app-slug}/contacts"
	handler := services.AppContactPost

	behavesAsServiceCravingHandler(t, httpMethod, url, handler, []string{"AppContactService", "BitriseAPI", "Mailer"}, ControllerTestCase{
		contextElements: map[ctxpkg.RequestContextKey]interface{}{
			services.ContextKeyAuthorizedAppID: uuid.NewV4(),
		},
		env: &env.AppEnv{
			AppContactService: &testAppContactService{},
			BitriseAPI:        &testBitriseAPI{},
			Mailer:            &testMailer{},
		},
		requestBody: `{}`,
	})

	behavesAsContextCravingHandler(t, httpMethod, url, handler, []ctxpkg.RequestContextKey{services.ContextKeyAuthorizedAppID}, ControllerTestCase{
		contextElements: map[ctxpkg.RequestContextKey]interface{}{
			services.ContextKeyAuthorizedAppID: uuid.NewV4(),
		},
		env: &env.AppEnv{
			AppContactService: &testAppContactService{},
			BitriseAPI:        &testBitriseAPI{},
			Mailer:            &testMailer{},
		},
		requestBody: `{}`,
	})

	t.Run("ok - minimal", func(t *testing.T) {
		performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
			contextElements: map[ctxpkg.RequestContextKey]interface{}{
				services.ContextKeyAuthorizedAppID: uuid.NewV4(),
			},
			env: &env.AppEnv{
				AppContactService: &testAppContactService{
					findFn: func(*models.AppContact) (*models.AppContact, error) {
						return nil, nil
					},
				},
				BitriseAPI: &testBitriseAPI{},
				Mailer:     &testMailer{},
			},
			requestBody:        `{}`,
			expectedStatusCode: http.StatusOK,
			expectedResponse:   services.AppContactPostResponse{},
		})
	})

	t.Run("ok - more complex", func(t *testing.T) {
		performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
			contextElements: map[ctxpkg.RequestContextKey]interface{}{
				services.ContextKeyAuthorizedAppID: uuid.FromStringOrNil("548bde58-2707-4c28-9474-4f35ba0176cb"),
			},
			env: &env.AppEnv{
				AddonHostURL: "http://ship.bitrise.io",
				AppContactService: &testAppContactService{
					createFn: func(contact *models.AppContact) (*models.AppContact, error) {
						contact.App = &models.App{APIToken: "test-api-token", AppSlug: "test-app-slug"}
						return contact, nil
					},
					findFn: func(*models.AppContact) (*models.AppContact, error) {
						return nil, gorm.ErrRecordNotFound
					},
				},
				BitriseAPI: &testBitriseAPI{
					getAppDetailsFn: func(apiToken, appSlug string) (*bitrise.AppDetails, error) {
						require.Equal(t, "test-api-token", apiToken)
						require.Equal(t, "test-app-slug", appSlug)
						return &bitrise.AppDetails{Title: "My awesome app"}, nil
					},
				},
				Mailer: &testMailer{
					sendEmailConfirmationFn: func(appTitle, addonBaseURL string, contact *models.AppContact) error {
						require.Equal(t, "My awesome app", appTitle)
						require.Equal(t, "http://ship.bitrise.io", addonBaseURL)
						require.NotEmpty(t, contact.ConfirmationToken)
						contact.ConfirmationToken = ""
						require.Equal(t, &models.AppContact{
							Email: "someones@email.addr",
							NotificationPreferencesData: json.RawMessage(`{"new_version":true,"successful_publish":false,"failed_publish":false}`),
							AppID: uuid.FromStringOrNil("548bde58-2707-4c28-9474-4f35ba0176cb"),
							App: &models.App{
								APIToken: "test-api-token",
								AppSlug:  "test-app-slug",
							},
						}, contact)
						return nil
					},
				},
			},
			requestBody:        `{"email":"someones@email.addr","notification_preferences":{"new_version":true}}`,
			expectedStatusCode: http.StatusOK,
			expectedResponse: services.AppContactPostResponse{
				Data: &models.AppContact{
					Email: "someones@email.addr",
					NotificationPreferencesData: json.RawMessage(`{"new_version":true,"successful_publish":false,"failed_publish":false}`),
					App: &models.App{
						Record:   models.Record{ID: uuid.FromStringOrNil("548bde58-2707-4c28-9474-4f35ba0176cb")},
						APIToken: "test-api-token",
						AppSlug:  "test-app-slug",
					},
				},
			},
		})
	})
}
