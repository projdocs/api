package meta

import (
	"sync"
	"time"

	"github.com/tus/tusd/v2/pkg/handler"
)

type Upload struct {
	Info       handler.FileInfo `json:"info"`
	FileID     string           `json:"file_id"`
	SessionURI string           `json:"session_uri"`
}

type entry struct {
	upload    *Upload
	expiresAt time.Time
}

var uploads sync.Map

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			uploads.Range(func(k, v any) bool {
				if v.(entry).expiresAt.Before(now) {
					uploads.Delete(k)
				}
				return true
			})
		}
	}()
}

func (meta *Upload) Store() {
	uploads.Store(meta.Info.ID, entry{
		upload:    meta,
		expiresAt: time.Now().Add(time.Hour),
	})
}

func Load(id string) (*Upload, bool) {
	v, ok := uploads.Load(id)
	if !ok {
		return nil, false
	}
	e := v.(entry)
	if time.Now().After(e.expiresAt) {
		uploads.Delete(id)
		return nil, false
	}
	return e.upload, true
}

func Delete(id string) {
	uploads.Delete(id)
}
