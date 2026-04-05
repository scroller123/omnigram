package tg

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"omnigram/ai"
	"omnigram/db"
)

type App struct {
	client        *telegram.Client
	api           *tg.Client
	repo          *db.Repository
	Gemini        *ai.GeminiClient
	Embedder      ai.Embedder
	SessionID     string
	EncryptionKey []byte

	apiID   int
	apiHash string
}

func NewApp(apiID int, apiHash string, storage telegram.SessionStorage, repo *db.Repository, gemini *ai.GeminiClient, embedder ai.Embedder, sessionID string, encKey []byte) *App {
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: storage,
	})

	return &App{
		client:        client,
		repo:          repo,
		Gemini:        gemini,
		Embedder:      embedder,
		SessionID:     sessionID,
		EncryptionKey: encKey,
		apiID:         apiID,
		apiHash:       apiHash,
	}
}

func (app *App) Start(ctx context.Context) error {
	fmt.Println("Starting Telegram client...")
	return app.client.Run(ctx, func(ctx context.Context) error {
		fmt.Println("Telegram client connection established.")
		app.api = tg.NewClient(app.client)

		// Check initial auth status
		status, _ := app.client.Auth().Status(ctx)
		if status.Authorized {
			fmt.Println("Telegram client is already authenticated.")
		} else {
			fmt.Println("Telegram client not authenticated. Waiting for login via API...")
		}

		<-ctx.Done()
		return ctx.Err()
	})
}

func (app *App) IsAuthenticated(ctx context.Context) (bool, error) {
	status, err := app.client.Auth().Status(ctx)
	if err != nil {
		return false, err
	}
	return status.Authorized, nil
}

func (app *App) SendCode(ctx context.Context, phone string) (string, error) {
	res, err := app.client.Auth().SendCode(ctx, phone, auth.SendCodeOptions{})
	if err != nil {
		return "", err
	}

	// res is of type tg.AuthSentCodeClass, we need to cast it to get the hash
	switch s := res.(type) {
	case *tg.AuthSentCode:
		return s.PhoneCodeHash, nil
	default:
		return "", fmt.Errorf("unexpected sent code type: %T", res)
	}
}

func (app *App) SignIn(ctx context.Context, phone, code, phoneCodeHash string) error {
	_, err := app.client.Auth().SignIn(ctx, phone, code, phoneCodeHash)
	if err != nil {
		return err
	}
	return nil
}

func (app *App) Password(ctx context.Context, password string) error {
	_, err := app.client.Auth().Password(ctx, password)
	if err != nil {
		return err
	}
	return nil
}

func (app *App) GetMe(ctx context.Context) (*tg.User, error) {
	user, err := app.client.Self(ctx)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (app *App) Logout(ctx context.Context) error {
	_, err := app.api.AuthLogOut(ctx)
	return err
}
