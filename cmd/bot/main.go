// Command bot is the entry point for tt-bot, a Telegram bot that manages
// qBittorrent downloads. It wires together configuration, the qBittorrent
// HTTP client, the Telegram bot handler, and the completion poller, then
// drives Telegram long-polling until a SIGINT or SIGTERM signal is received.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/home/tt-bot/internal/bot"
	"github.com/home/tt-bot/internal/config"
	"github.com/home/tt-bot/internal/poller"
	"github.com/home/tt-bot/internal/qbt"
)

// telegramNotifier implements poller.Notifier by sending a Telegram message
// to the given chat whenever a torrent download completes.
type telegramNotifier struct {
	bot *tgbotapi.BotAPI
}

// NotifyCompletion sends a completion message for torrent t to chatID.
func (n *telegramNotifier) NotifyCompletion(_ context.Context, chatID int64, t qbt.Torrent) error {
	text := fmt.Sprintf("✅ Download complete!\n\n%s", t.Name)
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := n.bot.Send(msg)
	return err
}

func main() {
	// 1. Load configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Create and authenticate the Telegram bot.
	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("failed to create Telegram bot: %v", err)
	}
	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	// 3. Create the qBittorrent HTTP client and authenticate.
	qbtClient := qbt.NewHTTPClient(cfg.QBTBaseURL, cfg.QBTUsername, cfg.QBTPassword)
	if err := qbtClient.Login(context.Background()); err != nil {
		log.Fatalf("qBittorrent login failed: %v", err)
	}
	log.Printf("Connected to qBittorrent at %s", cfg.QBTBaseURL)

	// 4. Set up a root context with graceful cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. Create the bot handler (passes ctx for cleanup goroutine shutdown).
	auth := bot.NewAuthorizer(cfg.AllowedUsers)
	handler := bot.New(botAPI, qbtClient, auth, cfg.TelegramToken, ctx)

	// 6. Create the completion notifier.
	notifier := &telegramNotifier{bot: botAPI}

	// 7. Start the completion poller in the background.
	p := poller.New(qbtClient, notifier, cfg.PollInterval, cfg.AllowedUsers)
	go p.Run(ctx)

	// 8. Install OS signal handler for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received %v, shutting down...", sig)
		cancel()
	}()

	// 9. Begin Telegram long-polling.
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := botAPI.GetUpdatesChan(u)

	log.Println("Bot started, listening for updates...")
	for {
		select {
		case update := <-updates:
			handler.HandleUpdate(ctx, update)
		case <-ctx.Done():
			log.Println("Shutting down...")
			return
		}
	}
}
