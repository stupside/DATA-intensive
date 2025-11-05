package models

import (
	"gorm.io/gorm"
)

// User represents a registered device/user
type User struct {
	gorm.Model
	Certificate *Certificate `gorm:"foreignKey:UserID"`
}

// Certificate stores the public key of a user's certificate
type Certificate struct {
	gorm.Model
	UserID    uint   `gorm:"not null;uniqueIndex"`
	PublicKey []byte `gorm:"not null"`
}

// Share represents a collection of shared files
type Share struct {
	gorm.Model
	Key      string     `gorm:"not null;uniqueIndex"`
	Contents []*Content `gorm:"many2many:share_contents;"`

	// FK to User who created the share
	UserID uint `gorm:"not null"`
}

// Content represents metadata about a file in a share
type Content struct {
	gorm.Model
	Size uint64 `gorm:"not null"`
	Path string `gorm:"not null"`
}
