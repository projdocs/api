package gdrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/projdocs/api/internal/storage/providers/gdrive/meta"
	"github.com/tus/tusd/v2/pkg/handler"
)

type Store struct {
	handler.DataStore
	g        *Provider
	parentID string
}

func (s *Store) NewUpload(ctx context.Context, info handler.FileInfo) (handler.Upload, error) {
	mimeType := "application/octet-stream"
	if mt, ok := info.MetaData["filetype"]; ok && mt != "" {
		mimeType = mt
	}

	filename := info.ID
	if fn, ok := info.MetaData["filename"]; ok && fn != "" {
		filename = fn
	}

	fileMeta := map[string]any{
		"name":    filename,
		"parents": []string{s.parentID},
	}
	metaBytes, err := json.Marshal(fileMeta)
	if err != nil {
		return nil, fmt.Errorf("new upload: marshal file meta: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://www.googleapis.com/upload/drive/v3/files?uploadType=resumable&supportsAllDrives=true",
		bytes.NewReader(metaBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("new upload: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("X-Upload-Content-Type", mimeType)
	if info.Size > 0 {
		req.Header.Set("X-Upload-Content-Length", strconv.FormatInt(info.Size, 10))
	}

	res, err := s.g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("new upload: do request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("new upload: bad status: %s", res.Status)
	}

	location := res.Header.Get("Location")
	if location == "" {
		return nil, fmt.Errorf("new upload: no Location header")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("new upload: parse session uri: %w", err)
	}
	uploadID := parsed.Query().Get("upload_id")
	if uploadID == "" {
		return nil, fmt.Errorf("new upload: no upload_id in session uri")
	}
	info.ID = uploadID

	u := &Upload{
		store: s,
		meta: &meta.Upload{
			Info:       info,
			SessionURI: location + "&supportsAllDrives=true",
		},
	}
	u.meta.Store()

	return u, nil
}

func (s *Store) GetUpload(ctx context.Context, id string) (handler.Upload, error) {
	m, ok := meta.Load(id)
	if !ok {
		return nil, handler.ErrNotFound
	}

	// reconstruct session URI from the stored upload_id (info.ID)
	// rather than trusting whatever was serialized, keeping it canonical
	m.SessionURI = "https://www.googleapis.com/upload/drive/v3/files?uploadType=resumable&supportsAllDrives=true&upload_id=" + m.Info.ID

	offset, err := s.querySessionOffset(ctx, m.SessionURI, m.Info.Size)
	if err != nil {
		return nil, fmt.Errorf("get upload: %w", err)
	}
	m.Info.Offset = offset

	return &Upload{meta: m, store: s}, nil
}

// querySessionOffset get existing bytes uploaded
func (s *Store) querySessionOffset(ctx context.Context, sessionURI string, totalSize int64) (int64, error) {
	total := strconv.FormatInt(totalSize, 10)
	if totalSize == 0 {
		total = "*"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, sessionURI, nil)
	if err != nil {
		return 0, fmt.Errorf("query offset: build request: %w", err)
	}
	req.Header.Set("Content-Range", "bytes */"+total)
	req.Header.Set("Content-Length", "0")

	resp, err := s.g.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("query offset: do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return totalSize, nil
	case 308:
		r := resp.Header.Get("Range")
		if r == "" {
			return 0, nil
		}
		upper, err := parseRangeUpper(r)
		if err != nil {
			return 0, fmt.Errorf("query offset: %w", err)
		}
		return upper + 1, nil
	default:
		return 0, fmt.Errorf("query offset: unexpected status %d", resp.StatusCode)
	}
}
