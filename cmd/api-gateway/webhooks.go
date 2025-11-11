package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aetherium/aetherium/pkg/flows"
	"github.com/aetherium/aetherium/pkg/integrations/discord"
	"github.com/aetherium/aetherium/pkg/integrations/slack"
	"github.com/go-chi/chi/v5"
)

// WebhookHandlers manages webhook handlers for different integrations
type WebhookHandlers struct {
	slackHandler   *slack.CommandHandler
	discordHandler *discord.CommandHandler
	flowExecutor   *flows.FlowExecutor
}

// NewWebhookHandlers creates a new webhook handlers instance
func NewWebhookHandlers(slackHandler *slack.CommandHandler, discordHandler *discord.CommandHandler, flowExecutor *flows.FlowExecutor) *WebhookHandlers {
	return &WebhookHandlers{
		slackHandler:   slackHandler,
		discordHandler: discordHandler,
		flowExecutor:   flowExecutor,
	}
}

// HandleSlackWebhook handles Slack slash commands and interactions
func (h *WebhookHandlers) HandleSlackWebhook(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "Failed to parse form", err)
		return
	}

	// Check if it's a slash command or interactive message
	payload := r.FormValue("payload")
	if payload != "" {
		// Interactive message callback
		h.handleSlackInteraction(w, r, payload)
		return
	}

	// Slash command
	command := r.FormValue("command")
	text := r.FormValue("text")
	userID := r.FormValue("user_id")
	channelID := r.FormValue("channel_id")

	if command == "" {
		respondError(w, http.StatusBadRequest, "Missing command", nil)
		return
	}

	// Handle command
	response, err := h.slackHandler.HandleCommand(r.Context(), command, text, userID, channelID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to handle command", err)
		return
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSlackInteraction handles interactive message callbacks from Slack
func (h *WebhookHandlers) handleSlackInteraction(w http.ResponseWriter, r *http.Request, payload string) {
	var interaction map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &interaction); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid interaction payload", err)
		return
	}

	// Handle button clicks
	if actionType, ok := interaction["type"].(string); ok && actionType == "block_actions" {
		actions, ok := interaction["actions"].([]interface{})
		if !ok || len(actions) == 0 {
			respondJSON(w, http.StatusOK, map[string]string{"status": "no_actions"})
			return
		}

		action := actions[0].(map[string]interface{})
		actionID := action["action_id"].(string)
		value := action["value"].(string)

		// Handle flow run button
		if actionID == "run_flow" {
			user := interaction["user"].(map[string]interface{})
			userID := user["id"].(string)

			channel := interaction["channel"].(map[string]interface{})
			channelID := channel["id"].(string)

			// Execute flow
			execution, err := h.flowExecutor.ExecuteFlow(r.Context(), value, map[string]interface{}{}, userID, channelID)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to execute flow", err)
				return
			}

			response := map[string]interface{}{
				"response_type": "in_channel",
				"text":          fmt.Sprintf("✅ Flow `%s` started with execution ID: `%s`", value, execution.ID.String()),
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleDiscordWebhook handles Discord interactions (slash commands and components)
func (h *WebhookHandlers) HandleDiscordWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify Discord signature
	timestamp := r.Header.Get("X-Signature-Timestamp")
	signature := r.Header.Get("X-Signature-Ed25519")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to read body", err)
		return
	}

	// Parse interaction
	var interaction map[string]interface{}
	if err := json.Unmarshal(body, &interaction); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid interaction payload", err)
		return
	}

	// Check interaction type
	interactionType := int(interaction["type"].(float64))

	// Handle PING (verification)
	if interactionType == 1 {
		respondJSON(w, http.StatusOK, map[string]interface{}{"type": 1})
		return
	}

	// Handle application command
	if interactionType == 2 {
		response, err := h.discordHandler.HandleCommand(r.Context(), interaction)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to handle command", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Handle message component (buttons, etc.)
	if interactionType == 3 {
		// TODO: Handle component interactions
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"type": 4,
			"data": map[string]interface{}{
				"content": "Component interaction received",
			},
		})
		return
	}

	respondError(w, http.StatusBadRequest, "Unknown interaction type", nil)
}

// SetupWebhookRoutes adds webhook routes to the API gateway
func SetupWebhookRoutes(r chi.Router, handlers *WebhookHandlers) {
	// Slack webhooks
	r.Post("/webhooks/slack", handlers.HandleSlackWebhook)
	r.Post("/slack/commands", handlers.HandleSlackWebhook)     // Alternative endpoint
	r.Post("/slack/interactions", handlers.HandleSlackWebhook) // Interactive messages

	// Discord webhooks
	r.Post("/webhooks/discord", handlers.HandleDiscordWebhook)
	r.Post("/discord/interactions", handlers.HandleDiscordWebhook) // Alternative endpoint
}
