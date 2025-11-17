package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/nurhudajoantama/hmauto/internal/config"
	"github.com/rs/zerolog"
)

type DiscordWebhook struct {
	webhookUrl string
	client     *http.Client
}

func NewDiscordWebhook(client *http.Client, discordWebhookConfig config.DiscordWebhook) *DiscordWebhook {
	return &DiscordWebhook{
		webhookUrl: discordWebhookConfig.WebhookUrl(),
		client:     client,
	}
}

func (d *DiscordWebhook) SendMessage(ctx context.Context, payload DiscordWebhookPayload) error {
	l := zerolog.Ctx(ctx)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		l.Error().Err(err).Msg("Failed to marshal DiscordWebhook payload")
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		l.Error().Err(err).Msg("Failed to create DiscordWebhook request")
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	_, err = d.client.Do(req)
	if err != nil {
		l.Error().Err(err).Msg("Failed to send DiscordWebhook webhook")
		return err
	}
	return nil
}

// DiscordWebhookPayload adalah struct utama untuk payload webhook DiscordWebhook.
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
