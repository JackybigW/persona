package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nwverse/persona/server/internal/apierr"
	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/db"
	"github.com/nwverse/persona/server/internal/i18n"
)

type healthTellStore interface {
	InsertHealthTell(context.Context, db.HealthTell) (db.HealthTell, bool, error)
	HealthTellsPending(context.Context, int) ([]db.HealthTell, error)
	ReserveHealthTellSession(context.Context, string, string) (string, error)
	SetHealthTellResult(context.Context, string, string, string, string, string, bool) (bool, error)
}

var _ healthTellStore = (*db.DB)(nil)

func (s *Service) healthTellRoutes() {
	s.handle("POST /api/v1/assistant/health/tell", s.healthTellCreate)
}

func (s *Service) healthTellCreate(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if !fullRole(r) && r.Header.Get(RoleHeader) != roleAnswer {
		apierr.Write(w, http.StatusForbidden, i18n.Err("assistant-health_tell.forbidden"))
		return
	}
	st, ok := s.store().(healthTellStore)
	if !ok {
		apierr.Write(w, http.StatusServiceUnavailable, errors.New("Tell Health store unavailable"))
		return
	}
	var in assistantwire.HealthTell
	// Ignore unknown fields: no caller-supplied conversation, space or source can select a destination.
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in); err != nil {
		apierr.Write(w, http.StatusBadRequest, err)
		return
	}
	if !utf8.ValidString(in.Text) {
		apierr.Write(w, http.StatusBadRequest, errors.New("text: invalid UTF-8"))
		return
	}
	in.Text = strings.TrimSpace(strings.Map(func(c rune) rune {
		if unicode.IsControl(c) && c != '\n' && c != '\t' {
			return -1
		}
		return c
	}, in.Text))
	if !captureClientID.MatchString(in.ClientID) || in.Text == "" || len(in.Text) > 8<<10 {
		apierr.Write(w, http.StatusBadRequest, errors.New("client_id: 8-128 letters, digits, dots, dashes, underscores or colons; text: 1-8192 bytes"))
		return
	}
	got, created, err := st.InsertHealthTell(r.Context(), db.HealthTell{ID: "tell_" + randomHex(8), ClientID: in.ClientID, Text: in.Text, Source: "shortcut"})
	if err != nil {
		apierr.Write(w, http.StatusInternalServerError, err)
		return
	}
	code := http.StatusOK
	if created {
		code = http.StatusAccepted
		s.healthTells.kick()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(assistantwire.HealthTellReceipt{ID: got.ID, Status: got.Status, Duplicate: !created})
	s.logf("health tell %s received after %s", got.ID, time.Since(started))
}
