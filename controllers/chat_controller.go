package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"app/ai/ai_agents"
	"app/helpers"
	"app/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/sashabaranov/go-openai"
)

type ChatController struct{}

func (c *ChatController) RegisterRoutes(router chi.Router) {
	const appPath = "/app/chat"

	router.Route(appPath, func(r chi.Router) {
		r.Use(httprate.LimitByIP(25, time.Minute))

		r.Get("/", helpers.AuthDecorator(c.renderChatList))
		r.Get("/new", helpers.AuthDecorator(c.renderCreatePage))
		r.Get("/{chatID}", helpers.AuthDecorator(c.renderDetailPage))
		r.Get("/{chatID}/edit", helpers.AuthDecorator(c.renderEditPage))
		r.Post("/", helpers.AuthDecorator(c.createChat))
		r.Post("/{chatID}/edit", helpers.AuthDecorator(c.updateChat))
		r.Patch("/{chatID}", helpers.AuthDecorator(c.updateChat))
		r.Delete("/{chatID}", helpers.AuthDecorator(c.deleteChat))
		r.With(httprate.LimitByIP(10, time.Minute)).Post("/{chatID}/messages", helpers.AuthDecorator(c.createMessage))
		r.With(httprate.LimitByIP(10, time.Minute)).Get("/{chatID}/ai-response", helpers.AuthDecorator(c.aiResponse))
	})
}

func (c *ChatController) renderCreatePage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro interno", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	data := map[string]any{
		"User": user,
	}

	RenderTemplate(w, filepath.Join("app", "chat_new.ejs"), data)
}

func (c *ChatController) renderChatList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro interno", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chats, _ := models.GetUserChats(dbCtx, tx, user.ID)
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	var alert map[string]any
	if status == "deleted" {
		alert = map[string]any{"Type": "success", "Message": "Chat removido."}
	}

	data := map[string]any{
		"User":  user,
		"Chats": chats,
		"Alert": alert,
	}

	RenderTemplate(w, filepath.Join("app", "chat_list.ejs"), data)
}

func (c *ChatController) renderDetailPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro interno", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chatID := chi.URLParam(r, "chatID")
	chat, err := models.GetUserChat(dbCtx, tx, chatID)
	if err != nil || chat.UserID != user.ID {
		helpers.RenderUnauthorized(w, r)
		return
	}

	messages, _ := models.GetUserChatMessages(dbCtx, tx, chat.ID)

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	var alert map[string]any
	if status == "updated" {
		alert = map[string]any{"Type": "success", "Message": "Chat atualizado com sucesso!"}
	}

	RenderTemplate(w, filepath.Join("app", "chat_show.ejs"), map[string]any{
		"User":     user,
		"Chat":     chat,
		"Messages": messages,
		"Alert":    alert,
	})
}

func (c *ChatController) renderEditPage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro interno", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chatID := chi.URLParam(r, "chatID")
	chat, err := models.GetUserChat(dbCtx, tx, chatID)
	if err != nil || chat.UserID != user.ID {
		helpers.RenderUnauthorized(w, r)
		return
	}

	RenderTemplate(w, filepath.Join("app", "chat_edit.ejs"), map[string]any{
		"User": user,
		"Chat": chat,
	})
}

func (c *ChatController) createChat(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao criar chat", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = "Novo chat"
	}

	chat := models.UserChat{
		UserID:  user.ID,
		Title:   title,
		Context: strings.TrimSpace(r.FormValue("context")),
	}

	if err := chat.Create(dbCtx, tx); err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao salvar chat", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}

	http.Redirect(w, r, "/app/chat/"+chat.ID, http.StatusSeeOther)
}

func (c *ChatController) updateChat(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao atualizar chat", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chatID := chi.URLParam(r, "chatID")
	chat, err := models.GetUserChat(dbCtx, tx, chatID)
	if err != nil || chat.UserID != user.ID {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chat.Title = strings.TrimSpace(r.FormValue("title"))
	chat.Context = strings.TrimSpace(r.FormValue("context"))

	if err := chat.Update(dbCtx, tx); err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao salvar edição", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}

	http.Redirect(w, r, "/app/chat/"+chat.ID+"?status=updated", http.StatusSeeOther)
}

func (c *ChatController) deleteChat(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao apagar chat", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chatID := chi.URLParam(r, "chatID")
	chat, err := models.GetUserChat(dbCtx, tx, chatID)
	if err != nil || chat.UserID != user.ID {
		helpers.RenderUnauthorized(w, r)
		return
	}

	_ = chat.Delete(dbCtx, tx)

	helpers.Redirect(w, r, "/app/chat?status=deleted")
	return
}

func (c *ChatController) createMessage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chatID := chi.URLParam(r, "chatID")
	chat, err := models.GetUserChat(dbCtx, tx, chatID)
	if err != nil || chat.UserID != user.ID {
		helpers.RenderUnauthorized(w, r)
		return
	}

	messageText := strings.TrimSpace(r.FormValue("message"))
	if messageText == "" {
		return
	}

	userMessage := models.UserChatMessage{
		ChatID:  chat.ID,
		Role:    "user",
		Message: messageText,
		Emotion: "neutro",
	}

	if err := userMessage.Create(dbCtx, tx); err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao salvar", Message: err.Error(), Path: r.URL.Path})
		return
	}

	// Render user message immediately + indicator for AI
	data := map[string]any{
		"UserMessage": userMessage,
		"ChatID":      chat.ID,
	}

	RenderTemplate(w, filepath.Join("partials", "chat_user_message_step.ejs"), data)
}

func (c *ChatController) aiResponse(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), LlmTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, LlmTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	chatID := chi.URLParam(r, "chatID")
	chat, err := models.GetUserChat(dbCtx, tx, chatID)
	if err != nil || chat.UserID != user.ID {
		helpers.RenderUnauthorized(w, r)
		return
	}

	messages, _ := models.GetUserChatMessages(dbCtx, tx, chat.ID)
	if len(messages) == 0 || messages[len(messages)-1].Role != "user" {
		return // Nothing to respond to
	}

	// lastUserMessage := messages[len(messages)-1].Message

	agentPrompt := `Você é o assistente de IA. Responda de forma útil e natural.
IMPORTANTE: Responda APENAS com um JSON válido:
{"reply": "sua resposta aqui", "user_emotion": "emoção dele", "assistant_emotion": "sua emoção"}`

	agent := ai_agents.NewAgent(agentPrompt, nil)

	contextParts := []string{"Chat: " + chat.Title}
	if chat.Context != "" {
		contextParts = append(contextParts, "Contexto: "+chat.Context)
	}
	systemContext := strings.Join(contextParts, "\n")

	conversation := []openai.ChatCompletionMessage{{Role: "system", Content: systemContext}}
	// Add a bit of history (last 4 messages)
	history := messages
	if len(history) > 5 {
		history = history[len(history)-5:]
	}
	for _, m := range history {
		role := "user"
		if m.Role == "assistant" {
			role = "assistant"
		}
		conversation = append(conversation, openai.ChatCompletionMessage{Role: role, Content: m.Message})
	}

	// AI Logic
	type chatResponse struct {
		Reply            string `json:"reply"`
		UserEmotion      string `json:"user_emotion"`
		AssistantEmotion string `json:"assistant_emotion"`
	}
	aiResult := chatResponse{Reply: "Entendi!", UserEmotion: "neutro", AssistantEmotion: "atencioso"}

	answer, err := agent.Answer(ctx, conversation)
	if err == nil && answer != "" {
		answer = strings.TrimSpace(answer)
		jsonStart := strings.Index(answer, "{")
		jsonEnd := strings.LastIndex(answer, "}")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			jsonStr := answer[jsonStart : jsonEnd+1]
			_ = json.Unmarshal([]byte(jsonStr), &aiResult)
		} else {
			aiResult.Reply = answer
		}
	}

	sanitizedReply := helpers.SanitizeAssistantMessage(aiResult.Reply)
	assistantMessage := models.UserChatMessage{
		ChatID:  chat.ID,
		Role:    "assistant",
		Message: sanitizedReply,
		Emotion: aiResult.AssistantEmotion,
	}
	_ = assistantMessage.Create(dbCtx, tx)

	// Update last user message emotion
	if len(messages) > 0 {
		userMsg := messages[len(messages)-1]
		userMsg.Emotion = aiResult.UserEmotion
		_ = userMsg.Update(dbCtx, tx)
	}

	RenderTemplate(w, filepath.Join("partials", "chat_ai_message_step.ejs"), map[string]any{
		"Message": assistantMessage,
	})
}
