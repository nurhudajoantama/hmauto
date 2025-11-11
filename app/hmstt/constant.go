package hmstt

const (
	MQ_CHANNEL_HMSTT = "hmstt_channel"

	PREFIX_HMSTT  = "hmstt"
	PREFIX_SWITCH = "switch"

	MODEM_SWITCH_KEY = "server_1" // pindahin ke database nih biar gampang maintenance

	HTML_TEMPLATE_PATTERN       = "views/hmstt/*.html"
	HTML_TEMPLATE_SWITCH        = "switch.html"
	HTML_TEMPLATE_NOTFOUND_TYPE = "notfoundtipe.html"

	STATE_OFF = "off"
	STATE_ON  = "on"

	ERR_STRING = "ERR"

	KEY_DELIMITER = "."

	INTERNET_CHECK_ADDRESS = "1.1.1.0"    // pindah ke config atau storage nanti
	INTERNET_MODEM_ADDRESS = "10.10.10.1" // pindah ke config atau storage nanti

	INTERVAL_NET_CHECK = 120 // seconds
)

var (
	TYPE_TEMPLATES = map[string]string{
		PREFIX_SWITCH: HTML_TEMPLATE_SWITCH,
	}
)
