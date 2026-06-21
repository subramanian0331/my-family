package handlers

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	adminhandler "github.com/subbu/family_tree/handlers/admin"
	authhandler "github.com/subbu/family_tree/handlers/auth"
	familyhandler "github.com/subbu/family_tree/handlers/family"
	gedcomhandler "github.com/subbu/family_tree/handlers/gedcom"
	"github.com/subbu/family_tree/handlers/health"
	invitehandler "github.com/subbu/family_tree/handlers/invite"
	"github.com/subbu/family_tree/handlers/middleware"
	"github.com/subbu/family_tree/handlers/response"
	personhandler "github.com/subbu/family_tree/handlers/person"
	photohandler "github.com/subbu/family_tree/handlers/photo"
	relationshiphandler "github.com/subbu/family_tree/handlers/relationship"
	searchhandler "github.com/subbu/family_tree/handlers/search"
	treehandler "github.com/subbu/family_tree/handlers/tree"
	authservice "github.com/subbu/family_tree/services/auth"
)

type Router struct {
	mux *http.ServeMux
}

type Dependencies struct {
	Auth         *authhandler.Handler
	Family       *familyhandler.Handler
	Invite       *invitehandler.Handler
	Person       *personhandler.Handler
	Relationship *relationshiphandler.Handler
	Search       *searchhandler.Handler
	Photo        *photohandler.Handler
	Tree         *treehandler.Handler
	Gedcom       *gedcomhandler.Handler
	Admin        *adminhandler.Handler
	Health       *health.Handler
	AuthSvc      authservice.Service
}

func NewRouter(deps Dependencies) *Router {
	mux := http.NewServeMux()
	withUser := func(handler http.HandlerFunc) http.Handler {
		return middleware.WithUser(deps.AuthSvc, handler)
	}

	mux.Handle("GET /api/health", deps.Health)
	mux.HandleFunc("GET /api/auth/status", deps.Auth.Status)
	mux.HandleFunc("GET /api/auth/google", deps.Auth.Login)
	mux.HandleFunc("GET /api/auth/google/callback", deps.Auth.Callback)
	mux.Handle("GET /api/auth/me", withUser(deps.Auth.Me))
	mux.HandleFunc("POST /api/auth/exchange", deps.Auth.Exchange)

	mux.Handle("GET /api/families", withUser(deps.Family.List))
	mux.Handle("POST /api/families", withUser(deps.Family.Create))
	mux.Handle("GET /api/families/{familyID}", withUser(withFamilyID(deps.Family.Get)))
	mux.Handle("PATCH /api/families/{familyID}", withUser(withFamilyID(deps.Family.Update)))
	mux.Handle("DELETE /api/families/{familyID}", withUser(withFamilyID(deps.Family.Delete)))
	mux.Handle("GET /api/families/{familyID}/members", withUser(withFamilyID(deps.Family.ListMembers)))
	mux.Handle("GET /api/families/{familyID}/invites", withUser(withFamilyID(deps.Family.ListInvites)))
	mux.Handle("POST /api/families/{familyID}/invites", withUser(withFamilyID(deps.Invite.Create)))
	mux.Handle("GET /api/families/{familyID}/tree", withUser(withFamilyID(deps.Tree.Get)))
	mux.Handle("GET /api/families/{familyID}/people", withUser(withFamilyID(deps.Person.List)))
	mux.Handle("POST /api/families/{familyID}/people", withUser(withFamilyID(deps.Person.Create)))
	mux.Handle("POST /api/families/{familyID}/people/bulk", withUser(withFamilyID(deps.Person.BulkCreate)))
	mux.Handle("POST /api/families/{familyID}/people/{personID}/add", withUser(withFamilyAndPersonID(deps.Person.AddToFamily)))
	mux.Handle("PATCH /api/families/{familyID}/people/{personID}/family-label", withUser(withFamilyAndPersonID(deps.Person.SetFamilyMarriageLabel)))
	mux.Handle("DELETE /api/families/{familyID}/people/{personID}", withUser(withFamilyAndPersonID(deps.Person.RemoveFromFamily)))
	mux.Handle("GET /api/families/{familyID}/search", withUser(withFamilyID(deps.Search.Search)))
	mux.Handle("GET /api/people/search", withUser(deps.Search.SearchGlobal))
	mux.Handle("GET /api/families/{familyID}/relationships", withUser(withFamilyID(deps.Relationship.List)))
	mux.Handle("POST /api/families/{familyID}/relationships", withUser(withFamilyID(deps.Relationship.Create)))
	mux.Handle("GET /api/families/{familyID}/gedcom/export", withUser(withFamilyID(deps.Gedcom.Export)))
	mux.Handle("POST /api/families/{familyID}/gedcom/preview", withUser(withFamilyID(deps.Gedcom.PreviewImport)))
	mux.Handle("POST /api/families/{familyID}/gedcom/import", withUser(withFamilyID(deps.Gedcom.CommitImport)))

	mux.Handle("GET /api/people/{personID}", withUser(withPersonID(deps.Person.Get)))
	mux.Handle("GET /api/people/{personID}/families", withUser(withPersonID(deps.Person.ListFamilies)))
	mux.Handle("PATCH /api/people/{personID}", withUser(withPersonID(deps.Person.Update)))
	mux.Handle("DELETE /api/people/{personID}", withUser(withPersonID(deps.Person.Delete)))
	mux.Handle("GET /api/people/{personID}/patronymic-suggestion", withUser(withPersonID(deps.Person.SuggestPatronymic)))
	mux.Handle("POST /api/people/{personID}/photos", withUser(withPersonID(deps.Photo.Upload)))

	mux.Handle("GET /api/photos/{photoID}", withUser(withPhotoID(deps.Photo.Get)))
	mux.Handle("DELETE /api/photos/{photoID}", withUser(withPhotoID(deps.Photo.Delete)))
	mux.Handle("PATCH /api/relationships/{relationshipID}", withUser(withRelationshipID(deps.Relationship.Update)))
	mux.Handle("DELETE /api/relationships/{relationshipID}", withUser(withRelationshipID(deps.Relationship.Delete)))

	mux.Handle("GET /api/invites/pending", withUser(deps.Invite.ListPending))
	mux.Handle("POST /api/invites/accept", withUser(deps.Invite.Accept))
	mux.Handle("DELETE /api/invites/{inviteID}", withUser(withInviteID(deps.Invite.Revoke)))

	mux.Handle("GET /api/admin/users", withUser(deps.Admin.ListUsers))
	mux.Handle("GET /api/admin/families", withUser(deps.Admin.ListFamilies))

	return &Router{mux: mux}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/api/") {
		r.mux.ServeHTTP(w, req)
		return
	}
	http.NotFound(w, req)
}

type uuidHandler func(http.ResponseWriter, *http.Request, uuid.UUID)

func withFamilyID(handler uuidHandler) http.HandlerFunc {
	return withPathID("familyID", handler)
}

func withPersonID(handler uuidHandler) http.HandlerFunc {
	return withPathID("personID", handler)
}

func withPhotoID(handler uuidHandler) http.HandlerFunc {
	return withPathID("photoID", handler)
}

func withRelationshipID(handler uuidHandler) http.HandlerFunc {
	return withPathID("relationshipID", handler)
}

func withInviteID(handler uuidHandler) http.HandlerFunc {
	return withPathID("inviteID", handler)
}

type familyPersonHandler func(http.ResponseWriter, *http.Request, uuid.UUID, uuid.UUID)

func withFamilyAndPersonID(handler familyPersonHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		familyID, err := uuid.Parse(r.PathValue("familyID"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid family id")
			return
		}
		personID, err := uuid.Parse(r.PathValue("personID"))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid person id")
			return
		}
		handler(w, r, familyID, personID)
	}
}

func withPathID(name string, handler uuidHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := uuid.Parse(r.PathValue(name))
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}
		handler(w, r, id)
	}
}