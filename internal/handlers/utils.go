package handlers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/projdocs/api/internal/db"
	"github.com/projdocs/api/internal/types/response"
	"github.com/projdocs/projdocs/packages/go/database"
)

func ResolveStorageProviderFromID(c *gin.Context, id uuid.UUID) (*database.PublicStorageProvidersSelect, bool) {
	var sp database.PublicStorageProvidersSelect
	err := db.MustGet().QueryRowContext(c, `
		SELECT sp.__is_migration_locked, sp.created_at, sp.data, sp.id, sp.is_valid, sp.type
		FROM public.storage_providers sp
		WHERE sp.id = $1
		LIMIT 1
	`, id).Scan(
		&sp.IsMigrationLocked,
		&sp.CreatedAt,
		&sp.Data,
		&sp.Id,
		&sp.IsValid,
		&sp.Type,
	)
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(c, http.StatusNotFound, "storage provider not found")
		return nil, false
	} else if err != nil {
		log.Printf("error getting storage provider: %v", err)
		response.Error(c, http.StatusInternalServerError, "error getting storage provider")
		return nil, false
	}

	if !sp.IsValid {
		response.Error(c, http.StatusBadRequest, "storage provider not properly configured")
		return nil, false
	}

	return &sp, true
}

func ResolveDefaultStorageProvider(c *gin.Context) (*database.PublicStorageProvidersSelect, bool) {
	var sp database.PublicStorageProvidersSelect
	err := db.MustGet().QueryRowContext(c, `
		SELECT sp.__is_migration_locked, sp.created_at, sp.data, sp.id, sp.is_valid, sp.type
		FROM public.storage_providers sp
		WHERE sp.type = 'BUILT_IN'::public.settings_storage_type
		LIMIT 1
	`).Scan(
		&sp.IsMigrationLocked,
		&sp.CreatedAt,
		&sp.Data,
		&sp.Id,
		&sp.IsValid,
		&sp.Type,
	)
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(c, http.StatusNotFound, "default storage provider not found")
		return nil, false
	} else if err != nil {
		log.Printf("error getting default storage provider: %v", err)
		response.Error(c, http.StatusInternalServerError, "error getting default storage provider")
		return nil, false
	}

	if !sp.IsValid {
		response.Error(c, http.StatusBadRequest, "default storage provider unexpectedly invalid")
		return nil, false
	}

	return &sp, true
}

func ResolveStorageProviderFromOrganization(c *gin.Context, organizationID uuid.UUID) (*database.PublicStorageProvidersSelect, bool) {
	var sp database.PublicStorageProvidersSelect
	err := db.MustGet().QueryRowContext(c, `
		SELECT sp.__is_migration_locked, sp.created_at, sp.data, sp.id, sp.is_valid, sp.type
		FROM public.storage_providers sp
		WHERE sp.id = (
			SELECT o.storage_providers_id
			FROM public.organizations o
			WHERE o.id = $1
			LIMIT 1
		)
		LIMIT 1
	`, organizationID).Scan(
		&sp.IsMigrationLocked,
		&sp.CreatedAt,
		&sp.Data,
		&sp.Id,
		&sp.IsValid,
		&sp.Type,
	)
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(c, http.StatusNotFound, "storage provider not found")
		return nil, false
	} else if err != nil {
		log.Printf("error getting storage provider: %v", err)
		response.Error(c, http.StatusInternalServerError, "error getting storage provider")
		return nil, false
	}

	if !sp.IsValid {
		response.Error(c, http.StatusBadRequest, "storage provider not properly configured for this organization")
		return nil, false
	}

	return &sp, true
}

func ResolveStorageProviderFromFolder(c *gin.Context, folderID uuid.UUID) (*database.PublicStorageProvidersSelect, bool) {
	var orgID uuid.UUID
	err := db.MustGet().QueryRowContext(c, `SELECT private.get_folder_organization_id($1)`, folderID.String()).Scan(&orgID)
	if err != nil {
		log.Printf("error getting folder's organization ID: %v", err)
		response.Error(c, http.StatusInternalServerError, "error getting folder's organization ID")
		return nil, false
	}

	return ResolveStorageProviderFromOrganization(c, orgID)
}
