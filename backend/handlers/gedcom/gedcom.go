package gedcom

import (
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/subbu/family_tree/handlers/access"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	familyservice "github.com/subbu/family_tree/services/family"
	gedcomservice "github.com/subbu/family_tree/services/gedcom"
)

type Handler struct {
	gedcom   gedcomservice.Service
	families familyservice.Service
}

func NewHandler(gedcom gedcomservice.Service, families familyservice.Service) *Handler {
	return &Handler{gedcom: gedcom, families: families}
}

func (h *Handler) Export(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireView(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	data, err := h.gedcom.ExportFamily(r.Context(), familyID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=family.ged")
	_, _ = w.Write(data)
}

func (h *Handler) PreviewImport(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file too large")
		return
	}

	preview, err := h.gedcom.PreviewImport(r.Context(), familyID, readerFromBytes(body))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, preview)
}

func (h *Handler) CommitImport(w http.ResponseWriter, r *http.Request, familyID uuid.UUID) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if _, ok := access.RequireEdit(w, r, h.families, familyID, user.ID); !ok {
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file too large")
		return
	}

	if err := h.gedcom.CommitImport(r.Context(), familyID, user.ID, readerFromBytes(body)); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"status": "imported"})
}

type bytesReader struct {
	data []byte
	pos  int
}

func readerFromBytes(data []byte) io.Reader {
	return &bytesReader{data: data}
}

func (b *bytesReader) Read(p []byte) (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n := copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}