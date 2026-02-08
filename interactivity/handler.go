package interactivity

import (
	"bytes"
	"encoding/json"
	"flight-tracker-slack/shared"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
)

var InteractionList []shared.Interaction = []shared.Interaction{
	TrackInteraction,
	UntrackInteraction,
}

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
	err = r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	payloadJSON := r.PostForm.Get("payload")
	println("Received interaction payload:", payloadJSON)

	var payload slack.InteractionCallback
	err = json.Unmarshal([]byte(payloadJSON), &payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error unmarshaling interaction payload:", err)
		return
	}

	switch payload.Type {
	case slack.InteractionTypeBlockActions:
		// separate actionId by "-"
		args := strings.Split(payload.ActionCallback.BlockActions[0].ActionID, "-")
		for _, interaction := range InteractionList {
			if args[0] == interaction.Prefix {
				go interaction.Execute(payload, config)
				break
			}
		}
		return
	default:
		// nothing yet
		return
	}
}
