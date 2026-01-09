package serviceimpl

import (
	"context"

	"github.com/google/uuid"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
)

type ActivityLogServiceImpl struct {
	activityLogRepo repositories.ActivityLogRepository
}

func NewActivityLogService(activityLogRepo repositories.ActivityLogRepository) services.ActivityLogService {
	return &ActivityLogServiceImpl{
		activityLogRepo: activityLogRepo,
	}
}

func (s *ActivityLogServiceImpl) GetByFolder(ctx context.Context, folderID uuid.UUID, page, limit int) ([]models.ActivityLog, int64, error) {
	// แปลง page เป็น offset
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.activityLogRepo.GetByFolder(ctx, folderID, offset, limit)
}

func (s *ActivityLogServiceImpl) GetByFolderAndType(ctx context.Context, folderID uuid.UUID, activityType models.ActivityType, page, limit int) ([]models.ActivityLog, int64, error) {
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	return s.activityLogRepo.GetByFolderAndType(ctx, folderID, activityType, offset, limit)
}

func (s *ActivityLogServiceImpl) GetRecent(ctx context.Context, limit int) ([]models.ActivityLog, error) {
	return s.activityLogRepo.GetRecent(ctx, limit)
}

func (s *ActivityLogServiceImpl) Cleanup(ctx context.Context, days int) (int64, error) {
	return s.activityLogRepo.DeleteOlderThan(ctx, days)
}
