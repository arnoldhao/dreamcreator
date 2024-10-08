package storage

import (
	"context"

	"gorm.io/gorm"
)

type Ollama struct {
	gorm.Model
	Key    string `gorm:"primaryKey;type:varchar(36);not null;unique"`
	Schema string `json:"schema" gorm:"type:varchar(10);not null" default:"http"`
	Domain string `json:"domain" gorm:"type:varchar(100);not null" default:"localhost"`
	Port   string `json:"port" gorm:"type:varchar(10);not null" default:"11434"`
}

func (o *Ollama) Create(ctx context.Context) error {
	o.Key = "default"
	return GetGlobalPersistentStorage().Create(ctx, o)
}

func (o *Ollama) Read(ctx context.Context) error {
	o.Key = "default"
	return GetGlobalPersistentStorage().First(ctx, o)
}

func (o *Ollama) Update(ctx context.Context) error {
	o.Key = "default"
	return GetGlobalPersistentStorage().Update(ctx, o, o)
}
