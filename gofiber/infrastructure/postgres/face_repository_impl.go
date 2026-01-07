package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type FaceRepositoryImpl struct {
	db *gorm.DB
}

func NewFaceRepository(db *gorm.DB) repositories.FaceRepository {
	return &FaceRepositoryImpl{db: db}
}

func (r *FaceRepositoryImpl) Create(ctx context.Context, face *models.Face) error {
	return r.db.WithContext(ctx).Create(face).Error
}

func (r *FaceRepositoryImpl) CreateBatch(ctx context.Context, faces []*models.Face) error {
	if len(faces) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(faces, 50).Error
}

func (r *FaceRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Face, error) {
	var face models.Face
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&face).Error
	if err != nil {
		return nil, err
	}
	return &face, nil
}

func (r *FaceRepositoryImpl) GetByPhoto(ctx context.Context, photoID uuid.UUID) ([]models.Face, error) {
	var faces []models.Face
	err := r.db.WithContext(ctx).Where("photo_id = ?", photoID).Find(&faces).Error
	return faces, err
}

func (r *FaceRepositoryImpl) GetByPerson(ctx context.Context, personID uuid.UUID) ([]models.Face, error) {
	var faces []models.Face
	err := r.db.WithContext(ctx).Where("person_id = ?", personID).Find(&faces).Error
	return faces, err
}

func (r *FaceRepositoryImpl) GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Face, int64, error) {
	var faces []models.Face
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Face{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Find(&faces).Error

	return faces, total, err
}

// SearchSimilar finds faces similar to the given embedding using cosine distance
func (r *FaceRepositoryImpl) SearchSimilar(ctx context.Context, userID uuid.UUID, embedding pgvector.Vector, limit int, threshold float64) ([]repositories.FaceSearchResult, error) {
	var results []repositories.FaceSearchResult

	// Use cosine distance for similarity search
	// pgvector uses <=> for cosine distance (1 - cosine similarity)
	// So we convert: similarity = 1 - distance
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT
			f.id, f.user_id, f.photo_id, f.embedding, f.bbox_x, f.bbox_y,
			f.bbox_width, f.bbox_height, f.confidence, f.person_id,
			f.created_at, f.updated_at,
			p.id as photo_id, p.drive_file_id, p.drive_folder_id, p.file_name, p.thumbnail_url,
			p.web_view_url, p.drive_folder_path,
			1 - (f.embedding <=> ?) as similarity
		FROM faces f
		JOIN photos p ON f.photo_id = p.id
		WHERE f.user_id = ?
		AND 1 - (f.embedding <=> ?) >= ?
		ORDER BY f.embedding <=> ?
		LIMIT ?
	`, embedding, userID, embedding, threshold, embedding, limit).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result repositories.FaceSearchResult
		var face models.Face
		var photo models.Photo

		err := rows.Scan(
			&face.ID, &face.UserID, &face.PhotoID, &face.Embedding,
			&face.BboxX, &face.BboxY, &face.BboxWidth, &face.BboxHeight,
			&face.Confidence, &face.PersonID, &face.CreatedAt, &face.UpdatedAt,
			&photo.ID, &photo.DriveFileID, &photo.DriveFolderID, &photo.FileName, &photo.ThumbnailURL,
			&photo.WebViewURL, &photo.DriveFolderPath,
			&result.Similarity,
		)
		if err != nil {
			return nil, err
		}

		result.Face = face
		result.Photo = photo
		results = append(results, result)
	}

	return results, nil
}

// SearchSimilarByFolderPathPrefix finds faces similar to the given embedding filtered by folder path prefix
func (r *FaceRepositoryImpl) SearchSimilarByFolderPathPrefix(ctx context.Context, pathPrefix string, embedding pgvector.Vector, limit int, threshold float64) ([]repositories.FaceSearchResult, error) {
	var results []repositories.FaceSearchResult

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT
			f.id, f.user_id, f.photo_id, f.embedding, f.bbox_x, f.bbox_y,
			f.bbox_width, f.bbox_height, f.confidence, f.person_id,
			f.created_at, f.updated_at,
			p.id as photo_id, p.drive_file_id, p.drive_folder_id, p.file_name, p.thumbnail_url,
			p.web_view_url, p.drive_folder_path,
			1 - (f.embedding <=> ?) as similarity
		FROM faces f
		JOIN photos p ON f.photo_id = p.id
		WHERE p.drive_folder_path LIKE ?
		AND 1 - (f.embedding <=> ?) >= ?
		ORDER BY f.embedding <=> ?
		LIMIT ?
	`, embedding, pathPrefix+"%", embedding, threshold, embedding, limit).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result repositories.FaceSearchResult
		var face models.Face
		var photo models.Photo

		err := rows.Scan(
			&face.ID, &face.UserID, &face.PhotoID, &face.Embedding,
			&face.BboxX, &face.BboxY, &face.BboxWidth, &face.BboxHeight,
			&face.Confidence, &face.PersonID, &face.CreatedAt, &face.UpdatedAt,
			&photo.ID, &photo.DriveFileID, &photo.DriveFolderID, &photo.FileName, &photo.ThumbnailURL,
			&photo.WebViewURL, &photo.DriveFolderPath,
			&result.Similarity,
		)
		if err != nil {
			return nil, err
		}

		result.Face = face
		result.Photo = photo
		results = append(results, result)
	}

	return results, nil
}

func (r *FaceRepositoryImpl) Update(ctx context.Context, id uuid.UUID, face *models.Face) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(face).Error
}

func (r *FaceRepositoryImpl) UpdatePersonID(ctx context.Context, id uuid.UUID, personID *uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.Face{}).Where("id = ?", id).Update("person_id", personID).Error
}

func (r *FaceRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Face{}).Error
}

func (r *FaceRepositoryImpl) DeleteByPhoto(ctx context.Context, photoID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("photo_id = ?", photoID).Delete(&models.Face{}).Error
}

func (r *FaceRepositoryImpl) Count(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Face{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// GetBySharedFolders returns faces from multiple shared folders with pagination
func (r *FaceRepositoryImpl) GetBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, offset, limit int) ([]models.Face, int64, error) {
	var faces []models.Face
	var total int64

	if len(folderIDs) == 0 {
		return faces, 0, nil
	}

	if err := r.db.WithContext(ctx).Model(&models.Face{}).Where("shared_folder_id IN ?", folderIDs).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Where("shared_folder_id IN ?", folderIDs).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&faces).Error

	return faces, total, err
}

// CountBySharedFolders returns total face count for multiple shared folders
func (r *FaceRepositoryImpl) CountBySharedFolders(ctx context.Context, folderIDs []uuid.UUID) (int64, error) {
	var count int64
	if len(folderIDs) == 0 {
		return 0, nil
	}
	err := r.db.WithContext(ctx).Model(&models.Face{}).Where("shared_folder_id IN ?", folderIDs).Count(&count).Error
	return count, err
}

// SearchSimilarBySharedFolders finds faces similar to the given embedding filtered by shared folder IDs
func (r *FaceRepositoryImpl) SearchSimilarBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, embedding pgvector.Vector, limit int, threshold float64) ([]repositories.FaceSearchResult, error) {
	var results []repositories.FaceSearchResult

	if len(folderIDs) == 0 {
		return results, nil
	}

	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT
			f.id, f.shared_folder_id, f.user_id, f.photo_id, f.embedding, f.bbox_x, f.bbox_y,
			f.bbox_width, f.bbox_height, f.confidence, f.person_id,
			f.created_at, f.updated_at,
			p.id as photo_id, p.shared_folder_id, p.drive_file_id, p.drive_folder_id, p.file_name, p.thumbnail_url,
			p.web_view_url, p.drive_folder_path,
			1 - (f.embedding <=> ?) as similarity
		FROM faces f
		JOIN photos p ON f.photo_id = p.id
		WHERE f.shared_folder_id IN ?
		AND 1 - (f.embedding <=> ?) >= ?
		ORDER BY f.embedding <=> ?
		LIMIT ?
	`, embedding, folderIDs, embedding, threshold, embedding, limit).Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var result repositories.FaceSearchResult
		var face models.Face
		var photo models.Photo

		err := rows.Scan(
			&face.ID, &face.SharedFolderID, &face.UserID, &face.PhotoID, &face.Embedding,
			&face.BboxX, &face.BboxY, &face.BboxWidth, &face.BboxHeight,
			&face.Confidence, &face.PersonID, &face.CreatedAt, &face.UpdatedAt,
			&photo.ID, &photo.SharedFolderID, &photo.DriveFileID, &photo.DriveFolderID, &photo.FileName, &photo.ThumbnailURL,
			&photo.WebViewURL, &photo.DriveFolderPath,
			&result.Similarity,
		)
		if err != nil {
			return nil, err
		}

		result.Face = face
		result.Photo = photo
		results = append(results, result)
	}

	return results, nil
}
