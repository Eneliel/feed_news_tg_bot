package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/Eneliel/feed_news_tg_bot/internal/bot"
	"github.com/Eneliel/feed_news_tg_bot/internal/botkit"
	"github.com/Eneliel/feed_news_tg_bot/internal/config"
	"github.com/Eneliel/feed_news_tg_bot/internal/fetcher"
	notifier "github.com/Eneliel/feed_news_tg_bot/internal/norifier"
	"github.com/Eneliel/feed_news_tg_bot/internal/storage"
	"github.com/Eneliel/feed_news_tg_bot/internal/summary"
)

func main() {
	botAPI, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		log.Printf("failed to create bot: %v", err)
		return
	}
	db, err := sqlx.Connect("postgres", config.Get().DatabaseDSN)
	if err != nil {
		log.Printf("failed to connect db: %v", err)
		return
	}
	defer db.Close()
	var (
		articleStorage = storage.NewArticleStorage(db)
		sourceStorage  = storage.NewSourceStorage(db)
		fetcher        = fetcher.New(
			articleStorage,
			sourceStorage,
			config.Get().FetchInterval,
			config.Get().FilterKeywords,
		)
		notifier = notifier.New(
			articleStorage,
			summary.NewOpenAISummarizer(config.Get().OpenAIKey, config.Get().OpenAIPrompt),
			botAPI,
			config.Get().NotificationInterval,
			2*config.Get().FetchInterval,
			config.Get().TelegramChannelID,
		)
	)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	newBot := botkit.New(botAPI)
	newBot.RegisterCmdView("start", bot.ViewCmdStart())

	go func(ctx context.Context) {
		if err := fetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run fetcher: %v", err)
				return
			}

			log.Printf("[INFO] fetcher stopped")
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := notifier.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[ERROR] failed to run notifier: %v", err)
				return
			}

			log.Printf("[INFO] notifier stopped")
		}
	}(ctx)

	if err := newBot.Run(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Printf("[ERROR] failed to start bot: %v", err)
			return
		}

		log.Printf("[INFO] bot stopped")
	}

}
