package gdrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/projdocs/api/internal/storage/providers/gdrive/meta"
	"github.com/tus/tusd/v2/pkg/handler"
	"google.golang.org/api/drive/v3"
)

type Upload struct {
	handler.Upload
	meta  *meta.Upload
	store *Store
}

func (u *Upload) GetInfo(_ context.Context) (handler.FileInfo, error) {
	return u.meta.Info, nil
}

func (u *Upload) WriteChunk(ctx context.Context, offset int64, src io.Reader) (int64, error) {
	data, err := io.ReadAll(src)
	if err != nil {
		return 0, fmt.Errorf("read chunk: %w", err)
	}
	if len(data) == 0 {
		return 0, nil
	}

	end := offset + int64(len(data)) - 1
	var contentRange string
	if u.meta.Info.Size > 0 {
		contentRange = fmt.Sprintf("bytes %d-%d/%d", offset, end, u.meta.Info.Size)
	} else {
		contentRange = fmt.Sprintf("bytes %d-%d/*", offset, end)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.meta.SessionURI, bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("build chunk request: %w", err)
	}
	req.Header.Set("Content-Range", contentRange)
	req.Header.Set("Content-Length", strconv.Itoa(len(data)))

	resp, err := u.store.g.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("upload chunk: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated: // upload complete

		// extract the file ID from the response body
		var f drive.File
		if err := json.NewDecoder(resp.Body).Decode(&f); err == nil {
			u.meta.FileID = f.Id
			u.meta.Info.MetaData["fileId"] = f.Id
		}
		u.meta.Info.Offset = u.meta.Info.Size

	case 308: // resume Incomplete

		// drive returns the Range header indicating bytes received so far
		if r := resp.Header.Get("Range"); r != "" {
			upper, parseErr := parseRangeUpper(r)
			if parseErr == nil {
				u.meta.Info.Offset = upper + 1
			}
		}

	default:
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("chunk upload status %d: %s", resp.StatusCode, body)
	}

	u.meta.Store()

	return int64(len(data)), nil
}

func (u *Upload) GetReader(ctx context.Context) (io.ReadCloser, error) {
	if u.meta.FileID == "" {
		return nil, fmt.Errorf("get reader: upload not complete")
	}
	resp, err := u.store.g.client.Files.Get(u.meta.FileID).
		SupportsAllDrives(true).
		Context(ctx).
		Download()
	if err != nil {
		return nil, fmt.Errorf("get reader: %w", err)
	}
	return resp.Body, nil
}

func (u *Upload) FinishUpload(ctx context.Context) error {
	u.meta.Store()
	return nil
}

// parseRangeUpper extracts the upper bound from a "bytes=0-N" or "bytes 0-N" header.
func parseRangeUpper(r string) (int64, error) {
	r = strings.TrimPrefix(r, "bytes=")
	r = strings.TrimPrefix(r, "bytes ")
	parts := strings.SplitN(r, "-", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("drivestore: malformed Range header: %q", r)
	}
	return strconv.ParseInt(parts[1], 10, 64)
}
