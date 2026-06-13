package types

import (
	"github.com/google/uuid"
	"github.com/tus/tusd/v2/pkg/handler"
)

type Callback = func(
	id string,
	storageProviderRecordId uuid.UUID,
	parentIdOrPathPrefix string,
	basePath string,
	checksum string,
	hook handler.HookEvent,
) handler.HTTPResponse
