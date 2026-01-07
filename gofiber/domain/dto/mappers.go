package dto

import (
	"gofiber-template/domain/models"
)

func UserToUserResponse(user *models.User) *UserResponse {
	if user == nil {
		return nil
	}

	// Mask API key for security (show only last 4 chars)
	maskedAPIKey := ""
	if user.GeminiAPIKey != "" {
		if len(user.GeminiAPIKey) > 4 {
			maskedAPIKey = "****" + user.GeminiAPIKey[len(user.GeminiAPIKey)-4:]
		} else {
			maskedAPIKey = "****"
		}
	}

	return &UserResponse{
		ID:           user.ID,
		Email:        user.Email,
		Username:     user.Username,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Avatar:       user.Avatar,
		Role:         user.Role,
		IsActive:     user.IsActive,
		Provider:     user.Provider,
		LastLogin:    user.LastLogin,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		GeminiAPIKey: maskedAPIKey,
		GeminiModel:  user.GeminiModel,
	}
}

func CreateUserRequestToUser(req *CreateUserRequest) *models.User {
	return &models.User{
		Email:     req.Email,
		Username:  req.Username,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}
}

func UpdateUserRequestToUser(req *UpdateUserRequest) *models.User {
	return &models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Avatar:    req.Avatar,
	}
}

func TaskToTaskResponse(task *models.Task, user *models.User) *TaskResponse {
	if task == nil {
		return nil
	}
	taskResp := &TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		Priority:    task.Priority,
		DueDate:     task.DueDate,
		UserID:      task.UserID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
	if user != nil {
		taskResp.User = *UserToUserResponse(user)
	}
	return taskResp
}

func CreateTaskRequestToTask(req *CreateTaskRequest) *models.Task {
	return &models.Task{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
	}
}

func UpdateTaskRequestToTask(req *UpdateTaskRequest) *models.Task {
	return &models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
	}
}

func JobToJobResponse(job *models.Job) *JobResponse {
	if job == nil {
		return nil
	}
	return &JobResponse{
		ID:        job.ID,
		Name:      job.Name,
		CronExpr:  job.CronExpr,
		Payload:   job.Payload,
		Status:    job.Status,
		LastRun:   job.LastRun,
		NextRun:   job.NextRun,
		IsActive:  job.IsActive,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
}

func CreateJobRequestToJob(req *CreateJobRequest) *models.Job {
	return &models.Job{
		Name:     req.Name,
		CronExpr: req.CronExpr,
		Payload:  req.Payload,
	}
}

func UpdateJobRequestToJob(req *UpdateJobRequest) *models.Job {
	return &models.Job{
		Name:     req.Name,
		CronExpr: req.CronExpr,
		Payload:  req.Payload,
		IsActive: req.IsActive,
	}
}

func FileToFileResponse(file *models.File) *FileResponse {
	if file == nil {
		return nil
	}
	return &FileResponse{
		ID:        file.ID,
		FileName:  file.FileName,
		FileSize:  file.FileSize,
		MimeType:  file.MimeType,
		URL:       file.URL,
		CDNPath:   file.CDNPath,
		UserID:    file.UserID,
		CreatedAt: file.CreatedAt,
		UpdatedAt: file.UpdatedAt,
	}
}

func PhotoToPhotoResponse(photo *models.Photo) *PhotoResponse {
	if photo == nil {
		return nil
	}
	return &PhotoResponse{
		ID:              photo.ID,
		SharedFolderID:  photo.SharedFolderID,
		DriveFileID:     photo.DriveFileID,
		FileName:        photo.FileName,
		MimeType:        photo.MimeType,
		ThumbnailURL:    photo.ThumbnailURL,
		WebViewURL:      photo.WebViewURL,
		DriveFolderPath: photo.DriveFolderPath,
		FaceStatus:      string(photo.FaceStatus),
		FaceCount:       photo.FaceCount,
		CreatedAt:       photo.CreatedAt,
	}
}

func PhotosToPhotoResponses(photos []models.Photo) []PhotoResponse {
	responses := make([]PhotoResponse, len(photos))
	for i, photo := range photos {
		responses[i] = *PhotoToPhotoResponse(&photo)
	}
	return responses
}