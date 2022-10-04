package status

import (
	"fmt"
	"strings"
)

const (
	LRed    string = "#ffdddd"
	LGreen         = "#ddffdd"
	LYellow        = "#ffffdd"
	LGray          = "#aaaaaa"

	Red    string = "#dd4444"
	Green         = "#44dd44"
	Yellow        = "#dddd44"
	Gray          = "#666666"
)

type Status struct {
	header string
	color  string

	icon       string
	icon_color string

	badge       []string
	badge_color []string
	badge_info  []string

	footer string
}

func Create(is_paused, is_throttled, is_unhealthy bool) (*Status, string) {
	ret := Status{}
	ret.icon = "&#9654;"
	ret.icon_color = Green
	ret.color = LGreen
	tmp := "This node is processing requests normally"
	if is_throttled {
		ret.icon = "&#x29D6;"
		ret.icon_color = Yellow
		ret.color = LYellow
		tmp = "This node is throttled, please wait"
	}
	if is_paused {
		ret.icon = "&#9208;"
		ret.icon_color = Gray
		ret.color = LGray
		tmp = "This node is paused"
	}
	if is_unhealthy {
		ret.icon = "&#9632;"
		ret.icon_color = Red
		ret.color = LRed
		tmp = "This node is not healthy, recent requests failed"
	}
	return &ret, tmp
}
func (this *Status) SetHeader(content string) {
	this.header = content
}
func (this *Status) AddBadge(text, color, info string) {

	if color == Green {
		color = "#449944"
	}

	this.badge = append(this.badge, text)
	this.badge_color = append(this.badge_color, color)
	this.badge_info = append(this.badge_info, info)
}
func (this *Status) Render() string {
	out := make([]string, 0, 50)
	out = append(out, fmt.Sprintf("<div class='node' style='background: %s'>", this.color))
	out = append(out, fmt.Sprintf("<div class='state' style='color: %s'>%s</div>", this.icon_color, this.icon))
	out = append(out, "<pre class='info'>"+this.header+"</pre>")

	out = append(out, "<div class='addl'>")
	for k, badge := range this.badge {
		info := this.badge_info[k]
		color := this.badge_color[k]
		out = append(out, fmt.Sprintf("<span style='background: %s'>%s<div>%s</div></span> ", color, badge, info))
	}
	out = append(out, "</div>")
	out = append(out, "</div>")
	return strings.Join(out, "")
}
