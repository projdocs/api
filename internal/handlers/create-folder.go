package handlers

import (
	"github.com/gin-gonic/gin"
)

func CreateFolder(ctx *gin.Context) {

	//// get request details
	//folderID, err := uuid.Parse(ctx.Param("id"))
	//if err != nil {
	//	response.Error(ctx, http.StatusBadRequest, fmt.Sprintf("bad id: %v", err))
	//	return
	//}
	//var body struct {
	//	Name string `json:"name" binding:"required"`
	//}
	//if err := ctx.ShouldBindJSON(&body); err != nil {
	//	response.Error(ctx, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
	//	return
	//}
	//parts := strings.Split(ctx.Request.URL.Path, "/")
	//if len(parts) < 3 {
	//	response.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("unable to handle %s", ctx.Request.URL.Path))
	//	return
	//}
	//var column *string = nil
	//switch parts[2] {
	//case "organizations":
	//	column = new("organization_id")
	//	break
	//case "projects":
	//	column = new("project_id")
	//	break
	//case "clients":
	//	column = new("client_id")
	//	break
	//case "folders":
	//	column = new("folder_id")
	//	break
	//case "members":
	//	column = new("member_id")
	//	break
	//default:
	//	response.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("unable to handle %s", ctx.Request.URL.Path))
	//	return
	//}
	//if column == nil {
	//	response.Error(ctx, http.StatusInternalServerError, fmt.Sprintf("unable to handle %s", ctx.Request.URL.Path))
	//	return
	//}
	//
	//// get user details
	//auth, authd := ctx.Get(middleware.AuthenticationJWTGinContextKey)
	//if !authd {
	//	response.Error(ctx, http.StatusForbidden, "authentication missing")
	//	return
	//}
	//token, ok := auth.(jwt.Token)
	//if !ok {
	//	response.Error(ctx, http.StatusForbidden, "invalid authentication token")
	//	return
	//}
	//uid, err := uuid.Parse(token.Subject())
	//if err != nil {
	//	response.Error(ctx, http.StatusBadRequest, "invalid subject")
	//	return
	//}
	//
	//// save to db
	//if txn, err := db.WithRLS(ctx, db.MustGet(), uid); err != nil {
	//	response.Error(ctx, http.StatusInternalServerError, "transaction failed")
	//} else {
	//	defer txn.Rollback()
	//
	//	// create folder
	//	var folder database.PublicFoldersSelect
	//	err = txn.QueryRowContext(
	//		ctx,
	//		fmt.Sprintf("INSERT INTO public.folders (name, %s) VALUES ($1, $2) RETURNING id", *column),
	//		body.Name,
	//		folderID.String(),
	//	).Scan(
	//		&folder.Id,
	//	)
	//	if err != nil {
	//		logger.Errorf("unable to insert folder: %v", err)
	//		response.Error(ctx, http.StatusInternalServerError, "failed to create folder")
	//		return
	//	}
	//
	//	// create storage folder
	//	//if sp, ok := ResolveStorageProvider(ctx, folder.Id); !ok {
	//	//	return
	//	//} else {
	//	//	s, e := storage.GetProviderFrom(sp)
	//	//	if e != nil {
	//	//		logger.Errorf("unable to get provider from storage: %v", e)
	//	//		response.Error(ctx, http.StatusInternalServerError, "unable to create folder in storage")
	//	//		return
	//	//	}
	//	//
	//	//	s.CreateFolder(ctx, "")
	//	//}
	//
	//	// commit
	//	if err := txn.Commit(); err != nil {
	//		response.Error(ctx, http.StatusInternalServerError, "failed to commit")
	//		return
	//	}
	//
	//	response.Data(ctx, http.StatusCreated, gin.H{"id": folder.Id})
	//}
}
