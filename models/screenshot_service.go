package models

import "github.com/jinzhu/gorm"

// ScreenshotService ...
type ScreenshotService struct {
	DB *gorm.DB
}

// BatchCreate ...
func (s *ScreenshotService) BatchCreate(screenshots []*Screenshot) ([]*Screenshot, []error, error) {
	tx := s.DB.Begin()
	for _, screenshot := range screenshots {
		result := tx.Create(screenshot)
		verrs := result.GetErrors()
		if len(verrs) > 0 {
			tx.Rollback()
			return nil, verrs, nil
		}
		if result.Error != nil {
			tx.Rollback()
			return nil, nil, result.Error
		}
	}
	return screenshots, nil, tx.Commit().Error
}

// FindAll ...
func (s *ScreenshotService) FindAll(appVersion *AppVersion) ([]Screenshot, error) {
	var screenshots []Screenshot
	err := s.DB.Preload("AppVersion").Preload("AppVersion.App").Where(map[string]interface{}{"app_version_id": appVersion.ID}).
		Find(&screenshots).
		Order("created_at DESC").Error
	if err != nil {
		return nil, err
	}
	return screenshots, nil
}