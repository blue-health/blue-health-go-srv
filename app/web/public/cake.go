package public

import (
	"net/http"

	"github.com/blue-health/blue-go-toolbox/authn"
	"github.com/blue-health/blue-go-toolbox/logger"
	"github.com/blue-health/blue-health-go-srv/app/cake"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type CakeService struct {
	logger      logger.Logger
	service     cake.Service
	authnPolicy *authn.Policy
}

func NewCakeService(authnPolicy *authn.Policy, service cake.Service, lg logger.Logger) *CakeService {
	return &CakeService{
		authnPolicy: authnPolicy,
		service:     service,
		logger:      lg,
	}
}

func (s *CakeService) Router() *chi.Mux {
	r := chi.NewRouter()

	r.With(s.authnPolicy.Parser, authn.Enforce).Group(func(r chi.Router) {
		r.Get("/{id}", s.getCake)
	})

	return r
}

func (s *CakeService) getCake(w http.ResponseWriter, r *http.Request) {
	identityID, ok := authn.GetIdentityID(r.Context())
	if !ok {
		s.logger.LogResponse(w, r, http.StatusPreconditionFailed)
		return
	}

	cookieID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.logger.LogResponseMessage(w, r, http.StatusPreconditionFailed, "failed to parse cake id")
		return
	}

	c, err := s.service.Get(r.Context(), cake.GetCmd{ID: cookieID, IdentityID: identityID})
	if err != nil {
		s.logger.LogServiceError(w, r, err)
		return
	}

	render.JSON(w, r, c)
}
