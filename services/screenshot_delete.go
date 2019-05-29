package services

import (
	"net/http"

	"github.com/bitrise-io/addons-ship-backend/env"
	"github.com/bitrise-io/addons-ship-backend/models"
	"github.com/bitrise-io/api-utils/httpresponse"
	"github.com/pkg/errors"
)

// ScreenshotDeleteResponse ...
type ScreenshotDeleteResponse struct {
	Data *models.Screenshot `json:"data"`
}

// ScreenshotDeleteHandler ...
func ScreenshotDeleteHandler(env *env.AppEnv, w http.ResponseWriter, r *http.Request) error {
	screenshotID, err := GetAuthorizedScreenshotIDFromContext(r.Context())
	if err != nil {
		return errors.WithStack(err)
	}

	if env.ScreenshotService == nil {
		return errors.New("No Screenshot Service defined for handler")
	}

	// aws.DeleteObject

	// // ID, err := uuid.FromString(screenshotID)
	// if err != nil {
	// 	return errors.New("Screenshot ID is invalid")
	// }

	screenshot, err := env.ScreenshotService.Find(&models.Screenshot{Record: models.Record{ID: screenshotID}})
	if err != nil {
		return errors.WithStack(err)
	}

	verrs, err := env.ScreenshotService.Delete(
		screenshot,
	)
	if len(verrs) > 0 {
		return httpresponse.RespondWithUnprocessableEntity(w, verrs)
	}
	if err != nil {
		return errors.Wrap(err, "SQL Error")
	}

	return httpresponse.RespondWithSuccess(w, ScreenshotDeleteResponse{
		Data: screenshot,
	})
}
