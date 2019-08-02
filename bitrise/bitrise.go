package bitrise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bitrise-io/api-utils/httpresponse"
	"github.com/pkg/errors"
)

var (
	apiVersion         = "v0.1"
	apiBaseURL         = "https://api.bitrise.io"
	validArtifactTypes = [...]string{"android-apk", "ios-ipa"}
)

// APIInterface ...
type APIInterface interface {
	GetArtifactData(string, string, string) (*ArtifactData, error)
	GetArtifacts(authToken, appSlug, buildSlug string) ([]ArtifactListElementResponseModel, error)
	GetArtifact(authToken, appSlug, buildSlug, artifactSlug string) (*ArtifactShowResponseItemModel, error)
	GetArtifactPublicInstallPageURL(string, string, string, string) (string, error)
	GetAppDetails(authToken, appSlug string) (*AppDetails, error)
	GetProvisioningProfiles(authToken, appSlug string) ([]ProvisioningProfile, error)
	GetCodeSigningIdentities(authToken, appSlug string) ([]CodeSigningIdentity, error)
	GetAndroidKeystoreFiles(authToken, appSlug string) ([]AndroidKeystoreFile, error)
	GetServiceAccountFiles(authToken, appSlug string) ([]GenericProjectFile, error)
	TriggerDENTask(params TaskParams) (*TriggerResponse, error)
	RegisterWebhook(authToken, appSlug, secret, callbackURL string) error
}

// API ...
type API struct {
	*http.Client
	url string
}

// New ...
func New() *API {
	url, ok := os.LookupEnv("BITRISE_API_ROOT_URL")
	if !ok {
		url = apiBaseURL
	}
	url = fmt.Sprintf("%s/%s", url, apiVersion)
	return &API{
		Client: &http.Client{},
		url:    url,
	}
}

func (a *API) doRequest(authToken, method, path string, requestPayload interface{}) (*http.Response, error) {
	var payloadBytes []byte
	if requestPayload != nil {
		var err error
		payloadBytes, err = json.Marshal(requestPayload)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", a.url, path), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	req.Header.Set("Bitrise-Addon-Auth-Token", authToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return resp, nil
}

// GetArtifactData ...
func (a *API) GetArtifactData(authToken, appSlug, buildSlug string) (*ArtifactData, error) {
	responseModel, err := a.listArtifacts(authToken, appSlug, buildSlug, "")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if responseModel.Paging.Next == "" {
		artifactData, err := getInstallableArtifactsFromResponseModel(responseModel)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if artifactData == nil {
			return nil, errors.New("No matching artifact found")
		}
		return artifactData, nil
	}
	next := responseModel.Paging.Next
	for next != "" {
		responseModel, err = a.listArtifacts(authToken, appSlug, buildSlug, next)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		artifactData, err := getInstallableArtifactsFromResponseModel(responseModel)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if artifactData != nil {
			return artifactData, nil
		}
		next = responseModel.Paging.Next
	}
	return nil, errors.New("No matching artifact found")
}

// GetArtifacts ...
func (a *API) GetArtifacts(authToken, appSlug, buildSlug string) ([]ArtifactListElementResponseModel, error) {
	var artifacts []ArtifactListElementResponseModel
	responseModel, err := a.listArtifacts(authToken, appSlug, buildSlug, "")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	artifacts = append(artifacts, responseModel.Data...)

	next := responseModel.Paging.Next
	for next != "" {
		responseModel, err = a.listArtifacts(authToken, appSlug, buildSlug, next)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		artifacts = append(artifacts, responseModel.Data...)
		next = responseModel.Paging.Next
	}
	return artifacts, nil
}

// GetArtifactPublicInstallPageURL ...
func (a *API) GetArtifactPublicInstallPageURL(authToken, appSlug, buildSlug, artifactSlug string) (string, error) {
	resp, err := a.doRequest(authToken, "GET", fmt.Sprintf("/apps/%s/builds/%s/artifacts/%s", appSlug, buildSlug, artifactSlug), nil)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("Failed to fetch artifact data: status: %d", resp.StatusCode)
	}
	var responseModel artifactShowResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return "", errors.WithStack(err)
	}
	return responseModel.Data.PublicInstallPageURL, nil
}

// GetArtifact ...
func (a *API) GetArtifact(authToken, appSlug, buildSlug, artifactSlug string) (*ArtifactShowResponseItemModel, error) {
	resp, err := a.doRequest(authToken, "GET", fmt.Sprintf("/apps/%s/builds/%s/artifacts/%s", appSlug, buildSlug, artifactSlug), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch artifact data: status: %d", resp.StatusCode)
	}
	var responseModel artifactShowResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	return &responseModel.Data, nil
}

// GetAppDetails ...
func (a *API) GetAppDetails(authToken, appSlug string) (*AppDetails, error) {
	resp, err := a.doRequest(authToken, "GET", "/apps/"+appSlug, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch app details: status: %d", resp.StatusCode)
	}
	var responseModel appShowResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	return &responseModel.Data, nil
}

// GetProvisioningProfiles ...
func (a *API) GetProvisioningProfiles(authToken, appSlug string) ([]ProvisioningProfile, error) {
	resp, err := a.doRequest(authToken, "GET", fmt.Sprintf("/apps/%s/provisioning-profiles", appSlug), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch provisioning profiles: status: %d", resp.StatusCode)
	}
	var responseModel provisioningProfileListResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	return responseModel.ProvisioningProfiles, nil
}

// GetCodeSigningIdentities ...
func (a *API) GetCodeSigningIdentities(authToken, appSlug string) ([]CodeSigningIdentity, error) {
	resp, err := a.doRequest(authToken, "GET", fmt.Sprintf("/apps/%s/build-certificates", appSlug), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch build certificates: status: %d", resp.StatusCode)
	}
	var responseModel codeSigningIdentityListResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	return responseModel.CodeSigningIdentities, nil
}

// GetAndroidKeystoreFiles ...
func (a *API) GetAndroidKeystoreFiles(authToken, appSlug string) ([]AndroidKeystoreFile, error) {
	resp, err := a.doRequest(authToken, "GET", fmt.Sprintf("/apps/%s/android-keystore-files", appSlug), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch android keystore files: status: %d", resp.StatusCode)
	}
	var responseModel androidKeystoreFileListResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	return responseModel.AndroidKeystoreFiles, nil
}

// GetServiceAccountFiles ...
func (a *API) GetServiceAccountFiles(authToken, appSlug string) ([]GenericProjectFile, error) {
	resp, err := a.doRequest(authToken, "GET", fmt.Sprintf("/apps/%s/generic-project-files", appSlug), nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch service account files: status: %d", resp.StatusCode)
	}
	var responseModel genericProjectFileListResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	serviceAccountFiles := []GenericProjectFile{}
	for _, genFile := range responseModel.GenericProjectFiles {
		if filepath.Ext(genFile.Filename) == ".json" {
			serviceAccountFiles = append(serviceAccountFiles, genFile)
		}
	}

	return serviceAccountFiles, nil
}

// TriggerDENTask ...
func (a *API) TriggerDENTask(params TaskParams) (*TriggerResponse, error) {
	payloadBytes, err := json.Marshal(params)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to JSON serialize")
	}
	req, err := http.NewRequest("POST", a.url+"/bitrise-den/tasks", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	denAuthHeaderKey, ok := os.LookupEnv("BITRISE_DEN_SERVER_ADMIN_SECRET_HEADER_KEY")
	if !ok {
		return nil, errors.New("No value set for env BITRISE_DEN_SERVER_ADMIN_SECRET_HEADER_KEY")
	}
	denAdminSecret, _ := os.LookupEnv("BITRISE_DEN_SERVER_ADMIN_SECRET")
	if !ok {
		return nil, errors.New("No value set for env BITRISE_DEN_SERVER_ADMIN_SECRET")
	}
	req.Header.Set(denAuthHeaderKey, denAdminSecret)

	resp, err := a.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to trigger DEN task: status: %d", resp.StatusCode)
	}

	var responseModel TriggerResponse
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	return &responseModel, nil
}

// RegisterWebhook ...
func (a *API) RegisterWebhook(authToken, appSlug, secret, callbackURL string) error {
	payloadBytes, err := json.Marshal(map[string]interface{}{
		"events": []string{"build"},
		"secret": secret,
		"url":    callbackURL,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/apps/%s/outgoing-webhooks", a.url, appSlug), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return errors.WithStack(err)
	}
	req.Header.Set("Bitrise-Addon-Auth-Token", authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer httpresponse.BodyCloseWithErrorLog(resp)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Reg webhook response", string(bodyBytes))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.Errorf("Failed to register webhook: status: %d", resp.StatusCode)
	}
	return nil
}

func (a *API) listArtifacts(authToken, appSlug, buildSlug, next string) (*artifactListResponseModel, error) {
	path := fmt.Sprintf("/apps/%s/builds/%s/artifacts", appSlug, buildSlug)
	if next != "" {
		path = fmt.Sprintf("%s?next=%s", path, next)
	}
	resp, err := a.doRequest(authToken, "GET", path, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer httpresponse.BodyCloseWithErrorLog(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Failed to fetch artifact data: status: %d", resp.StatusCode)
	}
	var responseModel artifactListResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&responseModel); err != nil {
		return nil, errors.WithStack(err)
	}
	return &responseModel, nil
}

func getInstallableArtifactsFromResponseModel(respModel *artifactListResponseModel) (*ArtifactData, error) {
	for _, buildArtifact := range respModel.Data {
		if validArtifact(buildArtifact) {
			var artifactMeta ArtifactMeta
			if buildArtifact.ArtifactMeta != nil {
				artifactMeta = *buildArtifact.ArtifactMeta
			}
			return &ArtifactData{
				Meta: artifactMeta,
				Slug: buildArtifact.Slug,
			}, nil
		}
	}
	return nil, nil
}

func validArtifact(artifact ArtifactListElementResponseModel) bool {
	for _, artifactType := range validArtifactTypes {
		if artifact.ArtifactType == nil {
			return false
		}
		if artifactType == *artifact.ArtifactType {
			return true
		}
	}
	return false
}
