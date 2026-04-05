package tg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"omnigram/auth"

	"github.com/gotd/td/tg"
)

func (app *App) GrabChannel(ctx context.Context, username string) error {
	if app.api == nil {
		return errors.New("telegram API client is not initialized yet")
	}

	fmt.Printf("Resolving username: %s\n", username)
	resolved, err := app.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve username: %w", err)
	}
	fmt.Println("resolved: ", resolved)

	if len(resolved.Chats) == 0 {
		return errors.New("no channels found for this username")
	}

	var targetChannel *tg.Channel
	for _, chat := range resolved.Chats {
		fmt.Println("chat: ", chat)
		if ch, ok := chat.(*tg.Channel); ok {
			targetChannel = ch
			break
		}
	}

	if targetChannel == nil {
		return errors.New("resolved peer is not a channel")
	}

	fmt.Printf("Found channel: %s (ID: %d)\n", targetChannel.Title, targetChannel.ID)

	err = app.repo.InsertChannel(targetChannel.ID, targetChannel.Title, username)
	if err != nil {
		return fmt.Errorf("failed to save channel to db: %w", err)
	}

	inputPeer := &tg.InputPeerChannel{
		ChannelID:  targetChannel.ID,
		AccessHash: targetChannel.AccessHash,
	}

	return app.grabHistory(ctx, inputPeer, targetChannel.ID, targetChannel.Title, username)
}

func (app *App) grabHistory(ctx context.Context, inputPeer tg.InputPeerClass, chatID int64, title string, username string) error {
	err := app.repo.InsertChannel(chatID, title, username)
	if err != nil {
		return fmt.Errorf("failed to save channel to db: %w", err)
	}

	limit := 100
	offsetID := 0

	fmt.Printf("Fetching messages for chat %s...\n", title)
	for {
		history, err := app.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:     inputPeer,
			OffsetID: offsetID,
			Limit:    limit,
		})
		if err != nil {
			return fmt.Errorf("failed to get history: %w", err)
		}

		var messages []tg.MessageClass
		switch h := history.(type) {
		case *tg.MessagesMessages:
			messages = h.Messages
		case *tg.MessagesMessagesSlice:
			messages = h.Messages
		case *tg.MessagesChannelMessages:
			messages = h.Messages
		default:
			return fmt.Errorf("unexpected history type: %T", history)
		}

		fmt.Println("Fetched messages: ", len(messages))
		if len(messages) == 0 {
			break
		}

		// The messages are returned from newest to oldest.
		// To paginate further back, the next offset should be the ID of the oldest message in this chunk.
		switch lastMsg := messages[len(messages)-1].(type) {
		case *tg.Message:
			offsetID = lastMsg.ID
		case *tg.MessageService:
			offsetID = lastMsg.ID
		}

		var window []string
		for i := len(messages) - 1; i >= 0; i-- {
			m := messages[i]
			if msg, ok := m.(*tg.Message); ok {
				// Maintain a window of up to 5 messages for context
				window = append(window, msg.Message)
				if len(window) > 5 {
					window = window[1:]
				}

				date := time.Unix(int64(msg.Date), 0)

				contextText := ""
				if len(window) > 1 {
					// Join window for context (excluding current message which is the last one)
					// We'll format it simply as a newline separated list of messages
					// Note: Telegram doesn't give us easy "User Name" in msg.Message
					// but it's okay for embedding context.
					contextText = strings.Join(window, "\n")
				} else {
					contextText = msg.Message
				}

				var embedding []float32
				if app.Embedder != nil && msg.Message != "" {
					// Use contextText for embedding instead of just the message text
					emb, err := app.Embedder.EmbedText(ctx, contextText)
					if err != nil {
						fmt.Printf("Error embedding message %d: %v\n", msg.ID, err)
					} else {
						embedding = emb
					}
				}

				err := app.repo.InsertMessage(msg.ID, chatID, date, embedding, nil, nil, nil, nil)
				if err != nil {
					fmt.Printf("Error saving message %d: %v\n", msg.ID, err)
				}
			}
		}

		fmt.Printf("Fetched %d messages, continuing from offset %d...\n", len(messages), offsetID)
		time.Sleep(1 * time.Second)

		if len(messages) < limit {
			break
		}
	}

	fmt.Printf("Finished fetching messages for chat %s.\n", title)
	return nil
}

// EnrichNextBatch processes a single batch of messages (approx 100) for enrichment.
// Returns (done, error). done is true if there are no more messages to process.
func (app *App) EnrichNextBatch(ctx context.Context, dialogID int64) (bool, error) {
	if app.api == nil {
		return false, errors.New("telegram API client is not initialized yet")
	}

	// 1. Get current task state
	task, err := app.repo.GetEnrichmentTask(dialogID, app.SessionID)
	if err != nil {
		return false, err
	}

	if task == nil {
		// Initialize task if not exists (should have been started by API)
		err = app.repo.UpsertEnrichmentTask(dialogID, app.SessionID, "processing", 0)
		if err != nil {
			return false, err
		}
		task, _ = app.repo.GetEnrichmentTask(dialogID, app.SessionID)
	}

	if task.Status == "completed" {
		return true, nil
	}

	offsetID := task.LastMessageID

	// 1-1 chats, groups, and channels resolution
	var inputPeer tg.InputPeerClass
	chats, users, err := app.GetDialogs(ctx)
	if err != nil {
		return false, err
	}

	// Check chats (Groups/Channels)
	for _, d := range chats {
		switch c := d.(type) {
		case *tg.Chat:
			if c.ID == dialogID {
				inputPeer = &tg.InputPeerChat{ChatID: c.ID}
				app.repo.InsertChannel(c.ID, c.Title, "")
			}
		case *tg.Channel:
			if c.ID == dialogID {
				inputPeer = &tg.InputPeerChannel{ChannelID: c.ID, AccessHash: c.AccessHash}
				username := ""
				if c.Username != "" {
					username = c.Username
				}
				app.repo.InsertChannel(c.ID, c.Title, username)
			}
		}
	}

	// Check users (1-1 Private Chats)
	if inputPeer == nil {
		for _, u := range users {
			if user, ok := u.(*tg.User); ok {
				if user.ID == dialogID {
					inputPeer = &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}
					username := ""
					if user.Username != "" {
						username = user.Username
					}
					fullName := strings.TrimSpace(user.FirstName + " " + user.LastName)
					if fullName == "" {
						fullName = "User " + fmt.Sprint(user.ID)
					}
					app.repo.InsertChannel(user.ID, fullName, username)
				}
			}
		}
	}

	if inputPeer == nil {
		return false, fmt.Errorf("dialog %d not found", dialogID)
	}

	limit := 100 // Process 100 messages per batch

	// Optimization: To get 5-window context, we fetch slightly more than the limit
	// if we're not at the very beginning of the history.
	// However, for simplicity and to follow "one batch = one request", we'll fetch 'limit'.
	history, err := app.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     inputPeer,
		OffsetID: offsetID,
		Limit:    limit,
	})
	if err != nil {
		log.Printf("Enrichment error (session %s, dialog %d): %v", app.SessionID, dialogID, err)
		app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, task.ProcessedMessages, offsetID, "failed", err.Error())
		return false, err
	}

	var messages []tg.MessageClass
	var usersFromBatch []tg.UserClass
	var chatsFromBatch []tg.ChatClass
	var totalMessages int
	switch h := history.(type) {
	case *tg.MessagesMessages:
		messages = h.Messages
		usersFromBatch = h.Users
		chatsFromBatch = h.Chats
		totalMessages = len(h.Messages)
	case *tg.MessagesMessagesSlice:
		messages = h.Messages
		usersFromBatch = h.Users
		chatsFromBatch = h.Chats
		totalMessages = h.Count
	case *tg.MessagesChannelMessages:
		messages = h.Messages
		usersFromBatch = h.Users
		chatsFromBatch = h.Chats
		totalMessages = h.Count
	default:
		return false, fmt.Errorf("unexpected history type: %T", history)
	}

	log.Printf("Fetched %d messages for dialog %d (offset %d)", len(messages), dialogID, offsetID)

	if len(messages) == 0 {
		app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, task.ProcessedMessages, offsetID, "completed", "")
		return true, nil
	}

	// Update total count if needed
	if task.TotalMessages == 0 {
		app.repo.UpsertEnrichmentTask(dialogID, app.SessionID, "processing", totalMessages)
	}

	// 4. Batching & Windowing
	type messageToProcess struct {
		m           *tg.Message
		contextText string
	}
	var toProcess []messageToProcess
	var messageBuffer []*tg.Message
	newOffsetID := offsetID

	for _, mc := range messages {
		if m, ok := mc.(*tg.Message); ok {
			messageBuffer = append(messageBuffer, m)
			newOffsetID = m.ID
		} else if ms, ok := mc.(*tg.MessageService); ok {
			newOffsetID = ms.ID
		}
	}

	peerNames := make(map[string]string)
	for _, u := range usersFromBatch {
		if user, ok := u.(*tg.User); ok {
			name := strings.TrimSpace(user.FirstName + " " + user.LastName)
			if name == "" {
				name = fmt.Sprintf("User%d", user.ID)
			}
			peerNames[fmt.Sprintf("user%d", user.ID)] = name
		}
	}
	for _, c := range chatsFromBatch {
		switch chat := c.(type) {
		case *tg.Chat:
			peerNames[fmt.Sprintf("chat%d", chat.ID)] = chat.Title
		case *tg.Channel:
			peerNames[fmt.Sprintf("channel%d", chat.ID)] = chat.Title
		}
	}

	formatMsg := func(m *tg.Message) string {
		author := "Unknown"
		if m.FromID != nil {
			switch p := m.FromID.(type) {
			case *tg.PeerUser:
				author = peerNames[fmt.Sprintf("user%d", p.UserID)]
			case *tg.PeerChat:
				author = peerNames[fmt.Sprintf("chat%d", p.ChatID)]
			case *tg.PeerChannel:
				author = peerNames[fmt.Sprintf("channel%d", p.ChannelID)]
			}
		}
		if author == "" {
			author = "Author"
		}
		ts := time.Unix(int64(m.Date), 0).Format("2006-01-02 15:04:05")
		return fmt.Sprintf("%s: %s (%s)", author, m.Message, ts)
	}

	for i := 0; i < len(messageBuffer); i++ {
		var past1, past2, past3, past4 *tg.Message
		if i+1 < len(messageBuffer) {
			past1 = messageBuffer[i+1]
		}
		if i+2 < len(messageBuffer) {
			past2 = messageBuffer[i+2]
		}
		if i+3 < len(messageBuffer) {
			past3 = messageBuffer[i+3]
		}
		if i+4 < len(messageBuffer) {
			past4 = messageBuffer[i+4]
		}

		windowTexts := []string{}
		if past4 != nil {
			windowTexts = append(windowTexts, formatMsg(past4))
		}
		if past3 != nil {
			windowTexts = append(windowTexts, formatMsg(past3))
		}
		if past2 != nil {
			windowTexts = append(windowTexts, formatMsg(past2))
		}
		if past1 != nil {
			windowTexts = append(windowTexts, formatMsg(past1))
		}
		windowTexts = append(windowTexts, formatMsg(messageBuffer[i]))

		toProcess = append(toProcess, messageToProcess{
			m:           messageBuffer[i],
			contextText: strings.Join(windowTexts, "\n"),
		})
	}

	var embeddings [][]float32
	if app.Embedder != nil && len(toProcess) > 0 {
		const batchSize = 1 // Reduced from 100 to avoid overloading Ollama
		for batchStart := 0; batchStart < len(toProcess); batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > len(toProcess) {
				batchEnd = len(toProcess)
			}

			chunk := toProcess[batchStart:batchEnd]
			texts := make([]string, len(chunk))
			for i := range chunk {
				texts[i] = chunk[i].contextText
			}

			log.Printf("Enrichment: Batch embedding %d messages (slice %d-%d) for dialog %d", len(texts), batchStart, batchEnd, dialogID)
			embs, err := app.Embedder.EmbedBatch(ctx, texts)
			if err != nil {
				log.Printf("Enrichment: Batch embedding failed (slice %d-%d): %v", err)
				app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, task.ProcessedMessages, newOffsetID, "failed", err.Error())
				return false, err
			}
			embeddings = append(embeddings, embs...)
			// Update progress after each small chunk of embeddings
			app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, task.ProcessedMessages+batchStart, chunk[len(chunk)-1].m.ID, "processing", "")
		}
	}

	processedCount := task.ProcessedMessages
	for i, item := range toProcess {
		var embedding []float32
		if i < len(embeddings) {
			embedding = embeddings[i]
		}

		encryptedData, nonce, err := auth.Encrypt(app.EncryptionKey, []byte(item.m.Message))
		if err != nil {
			app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, processedCount, item.m.ID, "failed", err.Error())
			return false, err
		}

		encryptedContext, nonceContext, err := auth.Encrypt(app.EncryptionKey, []byte(item.contextText))
		if err != nil {
			log.Printf("Context encryption failed: %v", err)
			app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, processedCount, item.m.ID, "failed", err.Error())
			return false, err
		}

		date := time.Unix(int64(item.m.Date), 0)
		if err := app.repo.InsertMessage(item.m.ID, dialogID, date, embedding, encryptedData, nonce, encryptedContext, nonceContext); err != nil {
			app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, processedCount, item.m.ID, "failed", err.Error())
			return false, err
		}
		processedCount++
		app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, processedCount, item.m.ID, "processing", "")
	}

	status := "processing"
	if len(messages) < limit {
		status = "completed"
	}
	err = app.repo.UpdateEnrichmentProgress(dialogID, app.SessionID, processedCount, newOffsetID, status, "")
	return status == "completed", err
}

func (app *App) grabFromChat(ctx context.Context, chatClass tg.ChatClass) error {
	switch chat := chatClass.(type) {
	case *tg.Chat:
		fmt.Printf("Found Group: %s (ID: %d)\n", chat.Title, chat.ID)
		peer := &tg.InputPeerChat{ChatID: chat.ID}
		return app.grabHistory(ctx, peer, chat.ID, chat.Title, "")
	case *tg.Channel:
		fmt.Printf("Found Supergroup/Channel: %s (ID: %d)\n", chat.Title, chat.ID)
		peer := &tg.InputPeerChannel{
			ChannelID:  chat.ID,
			AccessHash: chat.AccessHash,
		}

		username := ""
		if chat.Username != "" {
			username = chat.Username
		}

		return app.grabHistory(ctx, peer, chat.ID, chat.Title, username)
	default:
		return fmt.Errorf("unsupported chat format: %T", chatClass)
	}
}

func (app *App) GrabByInvite(ctx context.Context, hash string) error {
	if app.api == nil {
		return errors.New("telegram API client is not initialized yet")
	}

	fmt.Printf("Checking invite hash: %s\n", hash)
	invite, err := app.api.MessagesCheckChatInvite(ctx, hash)
	if err != nil {
		return fmt.Errorf("failed to check invite: %w", err)
	}

	switch inv := invite.(type) {
	case *tg.ChatInviteAlready:
		peer := inv.Chat
		return app.grabFromChat(ctx, peer)
	case *tg.ChatInvite:
		fmt.Printf("Importing invite for: %s\n", inv.Title)
		updates, err := app.api.MessagesImportChatInvite(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to import invite: %w", err)
		}

		upd, ok := updates.(*tg.Updates)
		if !ok || len(upd.Chats) == 0 {
			return errors.New("failed to find chat in updates after joining")
		}

		return app.grabFromChat(ctx, upd.Chats[0])
	default:
		return fmt.Errorf("unsupported invite type: %T", invite)
	}
}

func (app *App) GetDialogs(ctx context.Context) ([]tg.ChatClass, []tg.UserClass, error) {
	if app.api == nil {
		return nil, nil, errors.New("telegram API client is not initialized yet")
	}
	dialogs, err := app.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetDate: 0,
		OffsetID:   0,
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100, // Fetch the recent 100 dialogs
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		return d.Chats, d.Users, nil
	case *tg.MessagesDialogsSlice:
		return d.Chats, d.Users, nil
	case *tg.MessagesDialogsNotModified:
		return nil, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported dialogs type: %T", dialogs)
	}
}

func (app *App) GrabByID(ctx context.Context, chatID int64) error {
	chats, users, err := app.GetDialogs(ctx)
	if err != nil {
		return err
	}

	for _, chatClass := range chats {
		switch chat := chatClass.(type) {
		case *tg.Chat:
			if chat.ID == chatID {
				return app.grabFromChat(ctx, chatClass)
			}
		case *tg.Channel:
			if chat.ID == chatID {
				return app.grabFromChat(ctx, chatClass)
			}
		}
	}

	for _, u := range users {
		if user, ok := u.(*tg.User); ok {
			if user.ID == chatID {
				inputPeer := &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}
				username := ""
				if user.Username != "" {
					username = user.Username
				}
				fullName := strings.TrimSpace(user.FirstName + " " + user.LastName)
				if fullName == "" {
					fullName = "User " + fmt.Sprint(user.ID)
				}
				return app.grabHistory(ctx, inputPeer, user.ID, fullName, username)
			}
		}
	}

	return fmt.Errorf("chat/user with ID %d not found in your dialogs", chatID)
}
