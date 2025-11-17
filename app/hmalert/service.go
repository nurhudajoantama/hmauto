package hmalert

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type HmalerService struct {
	Discord *HmalertDiscord
	Event   *HmalertEvent
}

func NewService(discord *HmalertDiscord, event *HmalertEvent) *HmalerService {
	return &HmalerService{
		Discord: discord,
		Event:   event,
	}
}

func (s *HmalerService) SendDiscordNotification(ctx context.Context, body alertEvent) error {
	l := zerolog.Ctx(ctx)
	l.Info().Msgf("Sending Discord notification - Level: %s, Message: %s", body.Level, body.Message)

	var payload DiscordWebhookPayload
	payload.Content = "UIIAIUIIIAI"

	embed1 := DiscordEmbed{
		Title:       "Hmalert Notification",
		Description: "Alert " + body.Message,
		Color:       getDiscordColor(body.Level),
		Fields: []DiscordEmbedField{
			{
				Name:   "Type",
				Value:  body.Type,
				Inline: true,
			},
			{
				Name:   "Level",
				Value:  body.Level,
				Inline: true,
			},
			{
				Name:   "Timestamp",
				Value:  time.Unix(body.Timestamp, 0).Format(time.RFC3339),
				Inline: false,
			},
			{
				Name:   "Message",
				Value:  body.Message,
				Inline: false,
			},
		},
	}

	payload.Embeds = []DiscordEmbed{embed1}

	return s.Discord.SendMessage(ctx, body.Level, payload)
}

func (s *HmalerService) PublishAlert(ctx context.Context, tipe, level, message string) error {
	l := zerolog.Ctx(ctx)
	l.Info().Msgf("Publishing alert - Level: %s, Message: %s", level, message)

	body := alertEvent{
		Type:      tipe,
		Level:     level,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	err := s.Event.PublishAlert(ctx, body)
	if err != nil {
		return err
	}
	return nil
}
