package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/maruf-pfc/trestle/internal/integrations"
	"github.com/maruf-pfc/trestle/internal/projects"
)

func (s *Server) integrationScope(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) ([]uuid.UUID, projects.Project, bool) {
	ids, _, ok := s.scope(w, r, session, true, "project_id")
	if !ok {
		return nil, projects.Project{}, false
	}
	p, err := projects.GetProject(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		storeError(w, err, "failed to get integration project", "")
		return nil, p, false
	}
	return ids, p, true
}

func (s *Server) listIntegrations(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.integrationScope(w, r, session)
	if !ok {
		return
	}
	list, err := projects.ListIntegrations(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		internal(w, "failed to list integrations", err)
		return
	}
	if list == nil {
		list = []projects.Integration{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"integrations": list})
}

func (s *Server) saveIntegration(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.integrationScope(w, r, session)
	if !ok {
		return
	}
	var req struct {
		Secret string `json:"secret"`
		ChatID string `json:"chat_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	i := projects.Integration{Provider: r.PathValue("provider"), Secret: strings.TrimSpace(req.Secret), ChatID: strings.TrimSpace(req.ChatID)}
	if err := integrations.Validate(i.Provider, i.Secret, i.ChatID); err != nil {
		badRequest(w, err.Error())
		return
	}
	if err := projects.SaveIntegration(r.Context(), s.Pool, ids[0], ids[1], i); err != nil {
		if errors.Is(err, projects.ErrNotFound) {
			notFound(w)
		} else {
			internal(w, "failed to save integration", errors.New("integration storage failed"))
		}
		return
	}
	writeJSON(w, http.StatusOK, i)
}

func (s *Server) deleteIntegration(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.integrationScope(w, r, session)
	if !ok {
		return
	}
	if !integrations.ValidProvider(r.PathValue("provider")) {
		badRequest(w, "Unknown integration provider")
		return
	}
	if err := projects.DeleteIntegration(r.Context(), s.Pool, ids[0], ids[1], r.PathValue("provider")); err != nil {
		internal(w, "failed to disconnect integration", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testIntegration(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, p, ok := s.integrationScope(w, r, session)
	if !ok {
		return
	}
	if p.Archived {
		conflict(w, "Archived projects do not send notifications")
		return
	}
	list, err := projects.ListIntegrations(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		internal(w, "failed to load integration", err)
		return
	}
	for _, i := range list {
		if i.Provider != r.PathValue("provider") {
			continue
		}
		if i.LastAttemptAt != nil && time.Since(*i.LastAttemptAt) < 5*time.Second {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "Wait five seconds before sending another test")
			return
		}
		err = integrations.Send(r.Context(), i.Provider, i.Secret, i.ChatID, "Trestle test notification for "+p.Name+" ("+p.Key+").")
		message := ""
		if err != nil {
			message = err.Error()
		}
		if recordErr := projects.RecordIntegrationAttempt(r.Context(), s.Pool, ids[0], ids[1], i, message); recordErr != nil {
			internal(w, "failed to record integration test", errors.New("integration status storage failed"))
			return
		}
		if err != nil {
			writeError(w, http.StatusBadGateway, "delivery_failed", message)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
		return
	}
	notFound(w)
}

// shortcut: best-effort delivery has no queue or retry; add an outbox when guaranteed delivery is required.
func (s *Server) notifyTask(r *http.Request, ws, project uuid.UUID, action, key, title, status string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	p, err := projects.GetProject(ctx, s.Pool, ws, project)
	if err != nil || p.Archived {
		return
	}
	list, err := projects.ListIntegrations(ctx, s.Pool, ws, project)
	if err != nil {
		slog.Warn("could not load project notifications")
		return
	}
	text := fmt.Sprintf("%s (%s)\nTask %s: %s", p.Name, p.Key, action, key)
	if title != "" {
		text += " — " + title
	}
	if status != "" {
		text += "\nStatus category: " + status
	}
	var wg sync.WaitGroup
	for _, i := range list {
		wg.Add(1)
		go func() {
			defer wg.Done()
			message := ""
			if err := integrations.Send(ctx, i.Provider, i.Secret, i.ChatID, text); err != nil {
				message = err.Error()
			}
			recordCtx, done := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second)
			err := projects.RecordIntegrationAttempt(recordCtx, s.Pool, ws, project, i, message)
			done()
			if err != nil {
				slog.Warn("could not record project notification result")
			}
		}()
	}
	wg.Wait()
}
