package models

import (
	"time"
	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Title       string     `gorm:"not null"`
	Description string
	Status      string     `gorm:"default:'pending'"`
	Priority    int        `gorm:"default:1"`
	DueDate     *time.Time
	UserID      uuid.UUID  `gorm:"not null"`
	User        User       `gorm:"foreignKey:UserID"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (Task) TableName() string {
	return "tasks"
}