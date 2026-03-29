package tg

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

				err := app.repo.InsertMessage(msg.ID, chatID, msg.Message, contextText, date, embedding)
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

func (app *App) GetDialogs(ctx context.Context) ([]tg.ChatClass, error) {
	if app.api == nil {
		return nil, errors.New("telegram API client is not initialized yet")
	}
	dialogs, err := app.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		OffsetDate: 0,
		OffsetID:   0,
		OffsetPeer: &tg.InputPeerEmpty{},
		Limit:      100, // Fetch the recent 100 dialogs
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get dialogs: %w", err)
	}

	switch d := dialogs.(type) {
	case *tg.MessagesDialogs:
		return d.Chats, nil
	case *tg.MessagesDialogsSlice:
		return d.Chats, nil
	default:
		return nil, fmt.Errorf("unsupported dialogs type: %T", dialogs)
	}
}

func (app *App) GrabByID(ctx context.Context, chatID int64) error {
	chats, err := app.GetDialogs(ctx)
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

	return fmt.Errorf("chat ID %d not found in recent dialogs", chatID)
}
