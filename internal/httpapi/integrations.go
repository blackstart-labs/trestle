package httpapi

import (
	"context"
	"errors"
	"fmt"
	"html"
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
		Secret  string `json:"secret"`
		ChatID  string `json:"chat_id"`
		TopicID string `json:"topic_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	i := projects.Integration{
		Provider: r.PathValue("provider"),
		Secret:   strings.TrimSpace(req.Secret),
		ChatID:   strings.TrimSpace(req.ChatID),
		TopicID:  strings.TrimSpace(req.TopicID),
	}
	if err := integrations.ValidateWithTopic(i.Provider, i.Secret, i.ChatID, i.TopicID); err != nil {
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
		err = integrations.SendWithTopic(r.Context(), i.Provider, i.Secret, i.ChatID, i.TopicID, "Trestle test notification for "+p.Name+" ("+p.Key+").")
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

func formatTaskNotification(provider string, p projects.Project, t projects.Task, action, link string) string {
	var header string
	if t.MilestoneTitle != nil && strings.TrimSpace(*t.MilestoneTitle) != "" {
		header = fmt.Sprintf("📋 %s - %s", p.Name, strings.TrimSpace(*t.MilestoneTitle))
	} else {
		header = fmt.Sprintf("📋 %s", p.Name)
	}

	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = t.HumanKey
	}
	if action == "deleted" {
		title = fmt.Sprintf("%s (Deleted)", title)
	}
	titleLine := fmt.Sprintf("📌 Title: %s", title)

	statusName := ""
	if t.StatusName != nil && strings.TrimSpace(*t.StatusName) != "" {
		statusName = strings.TrimSpace(*t.StatusName)
	} else {
		switch strings.ToLower(t.Status) {
		case "started":
			statusName = "In Progress"
		case "completed":
			statusName = "Done"
		case "unstarted":
			statusName = "Todo"
		case "backlog":
			statusName = "Backlog"
		case "cancelled":
			statusName = "Cancelled"
		default:
			if len(t.Status) > 0 {
				statusName = strings.ToUpper(t.Status[:1]) + strings.ToLower(t.Status[1:])
			} else {
				statusName = "Todo"
			}
		}
	}
	statusLine := fmt.Sprintf("🏷 Status: %s", statusName)

	priority := strings.TrimSpace(t.Priority)
	if priority == "" || strings.EqualFold(priority, "none") {
		priority = "None"
	} else {
		priority = strings.ToUpper(priority[:1]) + strings.ToLower(priority[1:])
	}
	priorityLine := fmt.Sprintf("⚡ Priority: %s", priority)

	var assigneeNames []string
	for _, a := range t.Assignees {
		name := strings.TrimSpace(a.DisplayName)
		if name != "" {
			assigneeNames = append(assigneeNames, name)
		}
	}
	var assignedLine string
	if len(assigneeNames) > 0 {
		assignedLine = fmt.Sprintf("👤 Assigned To: %s", strings.Join(assigneeNames, ", "))
	} else {
		assignedLine = "👤 Assigned To: Unassigned"
	}

	var descBlock string
	desc := strings.TrimSpace(t.Description)
	if desc != "" {
		descBlock = fmt.Sprintf("\n\n📝 Description:\n%s", desc)
	}

	switch provider {
	case "telegram":
		escHeader := html.EscapeString(header)
		escTitle := html.EscapeString(titleLine)
		escStatus := html.EscapeString(statusLine)
		escPriority := html.EscapeString(priorityLine)
		escAssigned := html.EscapeString(assignedLine)
		var escDesc string
		if desc != "" {
			escDesc = fmt.Sprintf("\n\n📝 Description:\n%s", html.EscapeString(desc))
		}
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s%s\n\n👉 <a href=\"%s\">Open in Trestle</a>",
			escHeader, escTitle, escStatus, escPriority, escAssigned, escDesc, link)
	case "slack":
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s%s\n\n👉 <%s|Open in Trestle>",
			header, titleLine, statusLine, priorityLine, assignedLine, descBlock, link)
	default:
		// Discord and standard Markdown
		return fmt.Sprintf("%s\n\n%s\n%s\n%s\n%s%s\n\n👉 [Open in Trestle](%s)",
			header, titleLine, statusLine, priorityLine, assignedLine, descBlock, link)
	}
}

// shortcut: best-effort delivery has no queue or retry; add an outbox when guaranteed delivery is required.
func (s *Server) notifyTask(r *http.Request, ws, project uuid.UUID, action string, t projects.Task) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	p, err := projects.GetProject(ctx, s.Pool, ws, project)
	if err != nil || p.Archived {
		return
	}
	list, err := projects.ListIntegrations(ctx, s.Pool, ws, project)
	if err != nil || len(list) == 0 {
		return
	}

	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	if host == "" {
		host = "127.0.0.1:3000"
	}
	taskURL := fmt.Sprintf("%s://%s/#task/%s/%s/%s", scheme, host, ws, project, t.ID)

	var wg sync.WaitGroup
	for _, i := range list {
		wg.Add(1)
		go func(item projects.Integration) {
			defer wg.Done()
			msgText := formatTaskNotification(item.Provider, p, t, action, taskURL)
			message := ""
			if err := integrations.SendWithTopic(ctx, item.Provider, item.Secret, item.ChatID, item.TopicID, msgText); err != nil {
				message = err.Error()
			}
			recordCtx, done := context.WithTimeout(context.WithoutCancel(r.Context()), time.Second)
			err := projects.RecordIntegrationAttempt(recordCtx, s.Pool, ws, project, item, message)
			done()
			if err != nil {
				slog.Warn("could not record project notification result")
			}
		}(i)
	}
	wg.Wait()
}
