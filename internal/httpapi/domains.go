package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/maruf-pfc/trestle/internal/identity"
)

type instanceDomainResponse struct {
	PublicOrigin  string `json:"public_origin"`
	ListenAddress string `json:"listen_address"`
}

func (s *Server) getInstanceDomain(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if !session.User.InstanceOperator {
		forbidden(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()
	origin, err := identity.InstanceOrigin(ctx, s.Pool)
	if err != nil {
		internal(w, "failed to load instance domain", err)
		return
	}
	writeJSON(w, http.StatusOK, instanceDomainResponse{origin, s.Config.BindAddr.String()})
}

func (s *Server) updateInstanceDomain(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if !session.User.InstanceOperator {
		forbidden(w)
		return
	}
	var req struct {
		PublicOrigin   *string `json:"public_origin"`
		PreviousOrigin *string `json:"previous_origin"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.PublicOrigin, req.PreviousOrigin) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()
	origin, err := identity.UpdateInstanceOrigin(ctx, s.Pool, session.User.ID, *req.PreviousOrigin, *req.PublicOrigin)
	switch {
	case errors.Is(err, identity.ErrInvalidPublicOrigin):
		badRequest(w, err.Error())
	case errors.Is(err, identity.ErrForbidden):
		forbidden(w)
	case errors.Is(err, identity.ErrConflict):
		conflict(w, "The public URL changed. Reload the saved URL before saving again.")
	case err != nil:
		internal(w, "failed to save instance domain", err)
	default:
		writeJSON(w, http.StatusOK, instanceDomainResponse{origin, s.Config.BindAddr.String()})
	}
}
