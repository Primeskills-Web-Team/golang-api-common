package telegramnotifier

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TelegramNotifier struct {
	bot    *bot.Bot
	ChatID string
	ctx    context.Context
}

func New(botToken string, chatID string) (*TelegramNotifier, error) {
	bot, err := bot.New(botToken)
	if err != nil {
		return nil, err
	}

	return &TelegramNotifier{
		bot:    bot,
		ChatID: chatID,
		ctx:    context.Background(),
	}, nil
}

func (t *TelegramNotifier) SendMessage(message string) error {
	_, err := t.bot.SendMessage(t.ctx, &bot.SendMessageParams{
		ChatID:    t.ChatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})

	return err
}
