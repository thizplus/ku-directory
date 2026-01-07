package models

import (
	"time"
	"github.com/google/uuid"
)

type Job struct {
	ID        uuid.UUID  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name      string     `gorm:"not null"`
	CronExpr  string     `gorm:"not null"`
	Payload   string     `gorm:"type:jsonb"`
	Status    string     `gorm:"default:'active'"`
	LastRun   *time.Time
	NextRun   *time.Time
	IsActive  bool       `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Job) TableName() string {
	return "jobs"
}