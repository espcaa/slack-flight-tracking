package interactivity

import (
	"bytes"
	"encoding/json"
	"flight-tracker-slack/shared"
	"io"
	"net/http"

	"github.com/slack-go/slack"
)

func HandleInteraction(w http.ResponseWriter, r *http.Request, config shared.Config) {
	// verify the request

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	sv, err := slack.NewSecretsVerifier(r.Header, config.SigningSecret)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if _, err := sv.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := sv.Ensure(); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	// now parse the payload

	var payload slack.InteractionCallback
	err = json.Unmarshal(body, &payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch payload.Type {
	case slack.InteractionTypeBlockActions:
		// handle block actions

	case slack.InteractionTypeViewSubmission:
		// for now, this shouldn't happen
		return
	default:
		// nothing yet
		return
	}
}
