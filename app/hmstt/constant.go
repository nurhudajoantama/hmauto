package hmstt

const (
	MQ_CHANNEL_HMSTT = "hmstt_channel"

	PREFIX_HMSTT  = "hmstt"
	PREFIX_SWITCH = "switch"

	HTML_TEMPLATE_PATTERN       = "views/hmstt/*.html"
	HTML_TEMPLATE_SWITCH        = "switch.html"
	HTML_TEMPLATE_NOTFOUND_TYPE = "notfoundtipe.html"

	STATE_OFF = "off"
	STATE_ON  = "on"

	STATE_SWITCH_1 = "1"
	STATE_SWITCH_2 = "2"
	STATE_SWITCH_3 = "3"
	STATE_SWITCH_4 = "4"

	ERR_STRING = "ERR"

	KEY_DELIMITER = "."
)

var (
	SWITCH_STATES = []string{STATE_SWITCH_1, STATE_SWITCH_2, STATE_SWITCH_3, STATE_SWITCH_4}

	TYPE_TEMPLATES = map[string]string{
		PREFIX_SWITCH: HTML_TEMPLATE_SWITCH,
	}
)
