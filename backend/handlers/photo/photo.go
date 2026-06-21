package photo

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	familyservice "github.com/subbu/family_tree/services/family"
	photoservice "github.com/subbu/family_tree/services/photo"
)

const maxUploadSize = 5 << 20

type Handler struct {
	photos   photoservice.Service
	families familyservice.Service
}

func NewHandler(photos photoservice.Service, families familyservice.Service) *Handler {
	return &Handler{photos: photos, families: families}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request, personID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(r.URL.Query().Get("family_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "family_id query param required")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		response.Error(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "photo field required")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	photo, err := h.photos.Upload(r.Context(), photoservice.UploadInput{
		PersonID:   personID,
		Filename:   header.Filename,
		MimeType:   mimeType,
		SizeBytes:  header.Size,
		UploadedBy: user.ID,
		Reader:     file,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, photo)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request, photoID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_ = user

	reader, photo, err := h.photos.Open(r.Context(), photoID)
	if err != nil {
		response.Error(w, http.StatusNotFound, err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", photo.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", photo.SizeBytes))
	_, _ = io.Copy(w, reader)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request, photoID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	familyID, err := uuid.Parse(r.URL.Query().Get("family_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "family_id query param required")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	if err := h.photos.Delete(r.Context(), photoID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}