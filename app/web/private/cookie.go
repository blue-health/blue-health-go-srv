package private

import (
	"net/http"

	"github.com/blue-health/blue-go-toolbox/logger"
	"github.com/blue-health/blue-health-go-srv/app/cookie"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
)

type CookieService struct {
	logger  logger.Logger
	service cookie.Service
}

func NewCookieService(service cookie.Service, lg logger.Logger) *CookieService {
	return &CookieService{
		service: service,
		logger:  lg,
	}
}

func (s *CookieService) Router() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/{identityID}/{id}", s.getCookie)

	return r
}

func (s *CookieService) getCookie(w http.ResponseWriter, r *http.Request) {
	identityID, err := uuid.Parse(chi.URLParam(r, "identityID"))
	if err != nil {
		s.logger.LogResponseMessage(w, r, http.StatusPreconditionFailed, "failed to parse identity id")
		return
	}

	cookieID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.logger.LogResponseMessage(w, r, http.StatusPreconditionFailed, "failed to parse cookie id")
		return
	}

	c, err := s.service.Get(r.Context(), cookie.GetCmd{ID: cookieID, IdentityID: identityID})
	if err != nil {
		s.logger.LogServiceError(w, r, err)
		return
	}

	render.JSON(w, r, c)
}
