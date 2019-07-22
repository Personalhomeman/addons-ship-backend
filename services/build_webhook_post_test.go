package services_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/bitrise-io/addons-ship-backend/bitrise"
	"github.com/bitrise-io/addons-ship-backend/env"
	"github.com/bitrise-io/addons-ship-backend/models"
	"github.com/bitrise-io/addons-ship-backend/services"
	ctxpkg "github.com/bitrise-io/api-utils/context"
	"github.com/bitrise-io/api-utils/httpresponse"
	"github.com/c2fo/testify/require"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

func Test_BuildWebhookHandler(t *testing.T) {
	httpMethod := "POST"
	url := "/webhook"
	handler := services.BuildWebhookHandler

	behavesAsServiceCravingHandler(t, httpMethod, url, handler, []string{"AppService", "AppSettingsService", "AppVersionService", "BitriseAPI"}, ControllerTestCase{
		contextElements: map[ctxpkg.RequestContextKey]interface{}{
			services.ContextKeyAuthorizedAppID: uuid.NewV4(),
		},
		requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
		env: &env.AppEnv{
			AppService:         &testAppService{},
			AppVersionService:  &testAppVersionService{},
			AppSettingsService: &testAppSettingsService{},
			BitriseAPI:         &testBitriseAPI{},
		},
	})

	behavesAsContextCravingHandler(t, httpMethod, url, handler, []ctxpkg.RequestContextKey{services.ContextKeyAuthorizedAppID}, ControllerTestCase{
		contextElements: map[ctxpkg.RequestContextKey]interface{}{
			services.ContextKeyAuthorizedAppID: uuid.NewV4(),
		},
		requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
		env: &env.AppEnv{
			AppService:         &testAppService{},
			AppVersionService:  &testAppVersionService{},
			AppSettingsService: &testAppSettingsService{},
			BitriseAPI:         &testBitriseAPI{},
		},
	})

	t.Run("when build event type is started", func(t *testing.T) {
		performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
			contextElements: map[ctxpkg.RequestContextKey]interface{}{
				services.ContextKeyAuthorizedAppID: uuid.NewV4(),
			},
			requestHeaders:     map[string]string{"Bitrise-Event-Type": "build/started"},
			expectedStatusCode: http.StatusOK,
		})
	})

	t.Run("when build event type is finished", func(t *testing.T) {
		t.Run("ok - minimal", func(t *testing.T) {
			performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
				contextElements: map[ctxpkg.RequestContextKey]interface{}{
					services.ContextKeyAuthorizedAppID: uuid.NewV4(),
				},
				requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
				env: &env.AppEnv{
					AppService: &testAppService{
						findFn: func(app *models.App) (*models.App, error) {
							return app, nil
						},
					},
					AppSettingsService: &testAppSettingsService{
						findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
							return appSettings, nil
						},
					},
					AppVersionService: &testAppVersionService{},
					BitriseAPI:        &testBitriseAPI{},
				},
				requestBody:        `{}`,
				expectedStatusCode: http.StatusOK,
			})
		})

		t.Run("when request body contains invalid JSON", func(t *testing.T) {
			performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
				contextElements: map[ctxpkg.RequestContextKey]interface{}{
					services.ContextKeyAuthorizedAppID: uuid.NewV4(),
				},
				requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
				env: &env.AppEnv{
					AppService: &testAppService{
						findFn: func(app *models.App) (*models.App, error) {
							return app, nil
						},
					},
					AppSettingsService: &testAppSettingsService{
						findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
							return appSettings, nil
						},
					},
					AppVersionService: &testAppVersionService{},
					BitriseAPI:        &testBitriseAPI{},
				},
				requestBody:        `invalid JSON`,
				expectedStatusCode: http.StatusBadRequest,
				expectedResponse:   httpresponse.StandardErrorRespModel{Message: "Invalid request body, JSON decode failed"},
			})
		})

		t.Run("when db error happens at finding app", func(t *testing.T) {
			performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
				contextElements: map[ctxpkg.RequestContextKey]interface{}{
					services.ContextKeyAuthorizedAppID: uuid.NewV4(),
				},
				requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
				env: &env.AppEnv{
					AppService: &testAppService{
						findFn: func(app *models.App) (*models.App, error) {
							return nil, errors.New("SOME-SQL-ERROR")
						},
					},
					AppSettingsService: &testAppSettingsService{
						findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
							return appSettings, nil
						},
					},
					AppVersionService: &testAppVersionService{},
					BitriseAPI:        &testBitriseAPI{},
				},
				requestBody:         `{}`,
				expectedInternalErr: "SQL Error: SOME-SQL-ERROR",
			})
		})

		t.Run("when app settings not found in database", func(t *testing.T) {
			performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
				contextElements: map[ctxpkg.RequestContextKey]interface{}{
					services.ContextKeyAuthorizedAppID: uuid.NewV4(),
				},
				requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
				env: &env.AppEnv{
					AppService: &testAppService{
						findFn: func(app *models.App) (*models.App, error) {
							return app, nil
						},
					},
					AppSettingsService: &testAppSettingsService{
						findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
							return nil, gorm.ErrRecordNotFound
						},
					},
					AppVersionService: &testAppVersionService{},
					BitriseAPI:        &testBitriseAPI{},
				},
				requestBody:        `{}`,
				expectedStatusCode: http.StatusNotFound,
				expectedResponse:   httpresponse.StandardErrorRespModel{Message: "Not Found"},
			})
		})

		t.Run("when platform is ios", func(t *testing.T) {
			t.Run("ok - more complex - when ios workflow whitelist is 'all'", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "ios", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								require.Equal(t, "12", appVersion.BuildNumber)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								require.Equal(t, []string{"iPhone", "iPod Touch", "iPad", "Unknown"}, artifactData.SupportedDeviceTypes)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								require.Equal(t, "test-api-token", apiToken)
								require.Equal(t, "test-app-slug", appSlug)
								require.Equal(t, "test-build-slug", buildSlug)
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-ios-artifact.ipa",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												Version:          "1.0",
												DeviceFamilyList: []int{1, 2, 12},
											},
											ProvisioningInfo: bitrise.ProvisioningInfo{DistributionType: "app-store"},
										},
									},
								}, nil
							},
						},
					},
					requestBody:        `{"build_slug":"test-build-slug","build_number":12}`,
					expectedStatusCode: http.StatusOK,
				})
			})

			t.Run("ok - more complex - when triggered workflow is whitelisted for iOS", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "ios-wf,ios-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "ios", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								require.Equal(t, "test-api-token", apiToken)
								require.Equal(t, "test-app-slug", appSlug)
								require.Equal(t, "test-build-slug", buildSlug)
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-ios-artifact.ipa",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												Version: "1.0",
											},
											ProvisioningInfo: bitrise.ProvisioningInfo{DistributionType: "app-store"},
										},
									},
								}, nil
							},
						},
					},
					requestBody:        `{"build_slug":"test-build-slug","build_triggered_workflow":"ios-wf"}`,
					expectedStatusCode: http.StatusOK,
				})
			})

			t.Run("when error happens at finding app settings in database", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return nil, errors.New("SOME-SQL-ERROR")
							},
						},
						AppVersionService: &testAppVersionService{},
						BitriseAPI:        &testBitriseAPI{},
					},
					requestBody:         `{}`,
					expectedInternalErr: "SQL Error: SOME-SQL-ERROR",
				})
			})

			t.Run("when getting artifacts from API retrieves error", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "ios-wf,ios-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "ios", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return nil, errors.New("SOME-BITRISE-API-ERROR")
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug","build_triggered_workflow":"ios-wf"}`,
					expectedInternalErr: "SOME-BITRISE-API-ERROR",
				})
			})

			t.Run("when no matching artifact found", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "ios-wf,ios-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "ios", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug","build_triggered_workflow":"ios-wf"}`,
					expectedInternalErr: "No artifact found",
				})
			})

			t.Run("when selected artifact has no artifact meta", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "ios-wf,ios-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "ios", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title:        "my-ios-xcarchive.zip",
										ArtifactMeta: nil,
									},
								}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug","build_triggered_workflow":"ios-wf"}`,
					expectedInternalErr: "No artifact meta data found for artifact",
				})
			})

			t.Run("when selected artifact has no app info", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "ios-wf,ios-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "ios", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-ios-artifact.ipa",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo:          bitrise.AppInfo{},
											ProvisioningInfo: bitrise.ProvisioningInfo{DistributionType: "app-store"},
										},
									},
								}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug","build_triggered_workflow":"ios-wf"}`,
					expectedInternalErr: "No artifact app info found for artifact",
				})
			})

			t.Run("when selected artifact has no provision info", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "ios-wf,ios-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "ios", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-xcarchive.zip",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												Version: "1.0",
											},
											ProvisioningInfo: bitrise.ProvisioningInfo{},
										},
									},
								}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug","build_triggered_workflow":"ios-wf"}`,
					expectedInternalErr: "No artifact provisioning info found for artifact",
				})
			})

			t.Run("when validation error is retrieved when creating new ios version", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									IosWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								return nil, []error{errors.New("SOME-VALIDATION-ERROR")}, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-ios-artifact.ipa",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												Version: "1.0",
											},
											ProvisioningInfo: bitrise.ProvisioningInfo{DistributionType: "app-store"},
										},
									},
								}, nil
							},
						},
					},
					requestBody:        `{"build_slug":"test-build-slug"}`,
					expectedStatusCode: http.StatusUnprocessableEntity,
					expectedResponse: httpresponse.ValidationErrorRespModel{
						Message: "Unprocessable Entity",
						Errors:  []string{"SOME-VALIDATION-ERROR"},
					},
				})
			})

			t.Run("when db error is retrieved when creating new ios version", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{IosWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								return nil, nil, errors.New("SOME-SQL-ERROR")
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-ios-artifact.ipa",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												Version: "1.0",
											},
											ProvisioningInfo: bitrise.ProvisioningInfo{DistributionType: "app-store"},
										},
									},
								}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug"}`,
					expectedInternalErr: "SQL Error: SOME-SQL-ERROR",
				})
			})
		})

		t.Run("when platform is ios", func(t *testing.T) {
			t.Run("ok - more complex - when android workflow whitelist is 'all'", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{AndroidWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "android", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								appInfo, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, models.ArtifactInfo{Version: "1.0", MinimumSDK: "1.23", PackageName: "myPackage"}, appInfo)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-android-artifact.aab",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												VersionName:       "1.0",
												MinimumSDKVersion: "1.23",
												PackageName:       "myPackage",
											},
										},
									},
								}, nil
							},
						},
					},
					requestBody:        `{"build_slug":"test-build-slug"}`,
					expectedStatusCode: http.StatusOK,
				})
			})

			t.Run("ok - more complex - when triggered workflow is whitelisted for Android", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									AndroidWorkflow: "android-wf,android-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "android", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								appInfo, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, models.ArtifactInfo{Version: "1.0", MinimumSDK: "1.23", PackageName: "myPackage"}, appInfo)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-android-artifact.aab",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												VersionName:       "1.0",
												MinimumSDKVersion: "1.23",
												PackageName:       "myPackage",
											},
										},
									},
								}, nil
							},
						},
					},
					requestBody:        `{"build_slug":"test-build-slug","build_triggered_workflow":"android-wf"}`,
					expectedStatusCode: http.StatusOK,
				})
			})

			t.Run("when getting artifacts from API retrieves error", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									AndroidWorkflow: "android-wf,android-wf2",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "android", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return nil, errors.New("SOME-BITRISE-API-ERROR")
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug","build_triggered_workflow":"android-wf"}`,
					expectedInternalErr: "SOME-BITRISE-API-ERROR",
				})
			})

			t.Run("when no matching artifact found", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									AndroidWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "android", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug"}`,
					expectedInternalErr: "No artifact found",
				})
			})

			t.Run("when selected artifact has no artifact meta", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									AndroidWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "android", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title:        "my-android.aab",
										ArtifactMeta: nil,
									},
								}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug"}`,
					expectedInternalErr: "No artifact meta data found for artifact",
				})
			})

			t.Run("when selected artifact has no app info", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									AndroidWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								require.Equal(t, "android", appVersion.Platform)
								require.Equal(t, "test-build-slug", appVersion.BuildSlug)
								require.NotEqual(t, time.Time{}, appVersion.LastUpdate)
								artifactData, err := appVersion.ArtifactInfo()
								require.NoError(t, err)
								require.Equal(t, "1.0", artifactData.Version)
								return appVersion, nil, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-android.aab",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{},
										},
									},
								}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug"}`,
					expectedInternalErr: "No artifact app info found for artifact",
				})
			})

			t.Run("when validation error is retrieved when creating new android version", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									AndroidWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								return nil, []error{errors.New("SOME-VALIDATION-ERROR")}, nil
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-android-artifact.aab",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												VersionName:       "1.0",
												MinimumSDKVersion: "1.23",
												PackageName:       "myPackage",
											},
										},
									},
								}, nil
							},
						},
					},
					requestBody:        `{"build_slug":"test-build-slug"}`,
					expectedStatusCode: http.StatusUnprocessableEntity,
					expectedResponse: httpresponse.ValidationErrorRespModel{
						Message: "Unprocessable Entity",
						Errors:  []string{"SOME-VALIDATION-ERROR"},
					},
				})
			})

			t.Run("when db error is retrieved when creating new android version", func(t *testing.T) {
				performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
					contextElements: map[ctxpkg.RequestContextKey]interface{}{
						services.ContextKeyAuthorizedAppID: uuid.NewV4(),
					},
					requestHeaders: map[string]string{"Bitrise-Event-Type": "build/finished"},
					env: &env.AppEnv{
						AppService: &testAppService{
							findFn: func(app *models.App) (*models.App, error) {
								return app, nil
							},
						},
						AppSettingsService: &testAppSettingsService{
							findFn: func(appSettings *models.AppSettings) (*models.AppSettings, error) {
								return &models.AppSettings{
									AndroidWorkflow: "all",
									App: &models.App{
										APIToken: "test-api-token",
										AppSlug:  "test-app-slug",
									},
								}, nil
							},
						},
						AppVersionService: &testAppVersionService{
							createFn: func(appVersion *models.AppVersion) (*models.AppVersion, []error, error) {
								return nil, nil, errors.New("SOME-SQL-ERROR")
							},
						},
						BitriseAPI: &testBitriseAPI{
							getArtifactsFn: func(apiToken, appSlug, buildSlug string) ([]bitrise.ArtifactListElementResponseModel, error) {
								return []bitrise.ArtifactListElementResponseModel{
									bitrise.ArtifactListElementResponseModel{
										Title: "my-android-artifact.aab",
										ArtifactMeta: &bitrise.ArtifactMeta{
											AppInfo: bitrise.AppInfo{
												VersionName:       "1.0",
												MinimumSDKVersion: "1.23",
												PackageName:       "myPackage",
											},
										},
									},
								}, nil
							},
						},
					},
					requestBody:         `{"build_slug":"test-build-slug"}`,
					expectedInternalErr: "SQL Error: SOME-SQL-ERROR",
				})
			})
		})
	})

	t.Run("when build event type is invalid", func(t *testing.T) {
		performControllerTest(t, httpMethod, url, handler, ControllerTestCase{
			contextElements: map[ctxpkg.RequestContextKey]interface{}{
				services.ContextKeyAuthorizedAppID: uuid.NewV4(),
			},
			requestHeaders:      map[string]string{"Bitrise-Event-Type": "invalid build event type"},
			expectedInternalErr: "Invalid build event",
		})
	})
}