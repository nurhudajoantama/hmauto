package hmalert

import (
	"context"
	"fmt"
	"time"

	"github.com/nurhudajoantama/hmauto/internal/discord"
	"github.com/rs/zerolog"
)

type HmalerService struct {
	DiscordInfo    *discord.DiscordWebhook
	DiscordWarning *discord.DiscordWebhook
	DiscordError   *discord.DiscordWebhook

	Event *HmalertEvent
}

func NewService(discordInfo *discord.DiscordWebhook, discordWarning *discord.DiscordWebhook, discordError *discord.DiscordWebhook, event *HmalertEvent) *HmalerService {
	return &HmalerService{
		DiscordInfo:    discordInfo,
		DiscordWarning: discordWarning,
		DiscordError:   discordError,
		Event:          event,
	}
}

func (s *HmalerService) SendDiscordNotification(ctx context.Context, body alertEvent) error {
	l := zerolog.Ctx(ctx)
	l.Info().Msgf("Sending Discord notification - Level: %s, Message: %s", body.Level, body.Message)

	var payload discord.DiscordWebhookPayload

	timestamp := time.Unix(body.Timestamp, 0)

	embed1 := discord.DiscordEmbed{
		Title:       "Hmalert Notification",
		Description: "",
		Color:       getDiscordColor(body.Level),
		Fields: []discord.DiscordEmbedField{
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
				Name:   "Time",
				Value:  fmt.Sprintf("%s %s", timestamp.Format("2006-01-02"), timestamp.Format("15:04:05")),
				Inline: false,
			},
			{
				Name:   "Message",
				Value:  body.Message,
				Inline: false,
			},
		},
	}

	payload.Embeds = []discord.DiscordEmbed{embed1}

	var err error

	switch body.Level {
	case LEVEL_INFO:
		err = s.DiscordInfo.SendMessage(ctx, payload)
	case LEVEL_WARNING:
		err = s.DiscordWarning.SendMessage(ctx, payload)
	case LEVEL_ERROR:
		err = s.DiscordError.SendMessage(ctx, payload)
	default:
		l.Warn().Msgf("Unknown Discord message level: %s", body.Level)
	}

	return err
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

func getDiscordColor(level string) int {
	switch level {
	case LEVEL_INFO:
		return 0x00FF00 // Green
	case LEVEL_WARNING:
		return 0xFFFF00 // Yellow
	case LEVEL_ERROR:
		return 0xFF0000 // Red
	default:
		return 0x808080 // Grey
	}
}
