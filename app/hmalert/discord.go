package hmalert

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/nurhudajoantama/hmauto/internal/config"
	"github.com/rs/zerolog"
)

type HmalertDiscord struct {
	WebhookUrlInfo    string
	WebhookUrlWarning string
	WebhookUrlError   string

	client *http.Client
}

func NewDiscord(
	webhookUrlInfo config.DiscordWebhook,
	webhookUrlWarning config.DiscordWebhook,
	webhookUrlError config.DiscordWebhook,
) *HmalertDiscord {
	return &HmalertDiscord{
		WebhookUrlInfo:    webhookUrlError.WebhookUrl(),
		WebhookUrlWarning: webhookUrlWarning.WebhookUrl(),
		WebhookUrlError:   webhookUrlError.WebhookUrl(),

		client: &http.Client{},
	}
}

func (d *HmalertDiscord) SendMessage(ctx context.Context, level string, payload DiscordWebhookPayload) error {
	l := zerolog.Ctx(ctx)

	url := d.getUrlByLevel(level)
	if url == "" {
		return nil
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		l.Error().Err(err).Msg("Failed to marshal Discord payload")
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		l.Error().Err(err).Msg("Failed to create Discord request")
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	_, err = d.client.Do(req)
	if err != nil {
		l.Error().Err(err).Msg("Failed to send Discord webhook")
		return err
	}

	return nil
}

func (d *HmalertDiscord) getUrlByLevel(level string) string {
	switch level {
	case LEVEL_INFO:
		return d.WebhookUrlInfo
	case LEVEL_WARNING:
		return d.WebhookUrlWarning
	case LEVEL_ERROR:
		return d.WebhookUrlError
	default:
		return ""
	}
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

// DiscordWebhookPayload adalah struct utama untuk payload webhook Discord.
type DiscordWebhookPayload struct {
	Username    string              `json:"username,omitempty"`
	AvatarURL   string              `json:"avatar_url,omitempty"`
	Content     string              `json:"content,omitempty"`
	Embeds      []DiscordEmbed      `json:"embeds,omitempty"`
	Poll        *DiscordPoll        `json:"poll,omitempty"`
	Attachments []DiscordAttachment `json:"attachments,omitempty"`
}

// DiscordEmbed mewakili satu objek embed.
type DiscordEmbed struct {
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Color       int                    `json:"color,omitempty"`
	Author      *DiscordEmbedAuthor    `json:"author,omitempty"`
	Fields      []DiscordEmbedField    `json:"fields,omitempty"`
	Thumbnail   *DiscordEmbedThumbnail `json:"thumbnail,omitempty"`
	Image       *DiscordEmbedImage     `json:"image,omitempty"`
	Footer      *DiscordEmbedFooter    `json:"footer,omitempty"`
}

// DiscordPoll mewakili objek polling.
type DiscordPoll struct {
	Title            string              `json:"title"`
	Answers          []DiscordPollAnswer `json:"answers"`
	Duration         int                 `json:"duration"`
	AllowMultiselect bool                `json:"allow_multiselect"`
}

// DiscordAttachment mewakili objek attachment.
// (Struktur ini bisa lebih kompleks jika Anda mengirim file,
// tapi untuk JSON ini, strukturnya kosong).
type DiscordAttachment struct {
	// Biasanya kosong untuk payload kirim JSON murni,
	// atau bisa berisi 'id' jika merujuk ke attachment yang ada.
}

// DiscordEmbedAuthor mewakili penulis embed.
type DiscordEmbedAuthor struct {
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

// DiscordEmbedField mewakili satu field dalam embed.
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedThumbnail mewakili thumbnail embed.
type DiscordEmbedThumbnail struct {
	URL string `json:"url,omitempty"`
}

// DiscordEmbedImage mewakili gambar utama embed.
type DiscordEmbedImage struct {
	URL string `json:"url,omitempty"`
}

// DiscordEmbedFooter mewakili footer embed.
type DiscordEmbedFooter struct {
	Text string `json:"text,omitempty"`
}

// DiscordPollAnswer mewakili satu jawaban dalam poll.
type DiscordPollAnswer struct {
	Text string `json:"text"`
}
