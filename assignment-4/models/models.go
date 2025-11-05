package models

import (
	"gorm.io/gorm"
)

// Device represents a registered device
type Device struct {
	gorm.Model
	Certificate *Certificate `gorm:"foreignKey:DeviceID"`
}

// Certificate stores the public key of a device's certificate
type Certificate struct {
	gorm.Model
	DeviceID  uint64 `gorm:"not null;uniqueIndex"`
	PublicKey []byte `gorm:"not null"`
}

// Connection tracks all mTLS authentication attempts (successful and failed)
type Connection struct {
	gorm.Model
	DeviceID  uint64  `gorm:"not null;index"`
	IPAddress string  `gorm:"not null"`
	Success   bool    `gorm:"not null"`
	Device    *Device `gorm:"foreignKey:DeviceID"`
}

// Share represents a collection of shared files
type Share struct {
	gorm.Model
	Files  []*File `gorm:"many2many:share_files;"`
	Secret string  `gorm:"column:share_secret;not null;uniqueIndex"`

	// FK to Device who created the share
	DeviceID uint64 `gorm:"not null"`
}

// File represents metadata about a file in a share
type File struct {
	gorm.Model
	Size uint64 `gorm:"not null"`
	Path string `gorm:"not null"`
}
