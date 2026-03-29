package tg

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"telegai/ai"
	"telegai/db"
)

type termAuth struct {
	phone string
}

func (a termAuth) Phone(_ context.Context) (string, error) {
	if a.phone != "" {
		return a.phone, nil
	}
	fmt.Print("Enter Phone: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (a termAuth) Password(_ context.Context) (string, error) {
	fmt.Print("Enter 2FA Password: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (a termAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter Code: ")
	code, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (a termAuth) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up is not allowed")
}

func (a termAuth) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

type App struct {
	client   *telegram.Client
	api      *tg.Client
	repo     *db.Repository
	Gemini   *ai.GeminiClient
	Embedder ai.Embedder
}

func NewApp(apiID int, apiHash, sessionFile string, repo *db.Repository, gemini *ai.GeminiClient, embedder ai.Embedder) *App {
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{
			Path: sessionFile,
		},
	})
	
	return &App{
		client:   client,
		repo:     repo,
		Gemini:   gemini,
		Embedder: embedder,
	}
}

func (app *App) Start(ctx context.Context, phone string) error {
	fmt.Println("Starting Telegram client...")
	return app.client.Run(ctx, func(ctx context.Context) error {
		flow := auth.NewFlow(
			termAuth{phone: phone},
			auth.SendCodeOptions{},
		)
		
		fmt.Println("Authenticating...")
		if err := app.client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}

		fmt.Println("Telegram client successfully authenticated and running.")
		app.api = tg.NewClient(app.client)
		
		<-ctx.Done()
		return ctx.Err()
	})
}
