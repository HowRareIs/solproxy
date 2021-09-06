package handle_echo

import (
	"encoding/json"
	"strings"

	"github.com/slawomir-pryczek/handler_socket2"
)

type HandleEcho struct {
}

func (this *HandleEcho) Initialize() {
	handler_socket2.StatusPluginRegister(func() (string, string) {
		return "Echo", "Echo plugin is enabled"
	})
}

func (this *HandleEcho) Info() string {
	return "This plugin will send back received data"
}

func (this *HandleEcho) GetActions() []string {
	return []string{"echo"}
}

func (this *HandleEcho) HandleAction(action string, data *handler_socket2.HSParams) string {

	ret := map[string]string{}

	in := data.GetParam("data", "")
	repeat := data.GetParamI("repeat", 1)
	if repeat < 1 {
		repeat = 1
		ret["warning"] = "Repeat must be at least 1"
	}
	if repeat > 500 {
		repeat = 500
		ret["warning"] = "Repeat must be at most 500"
	}

	if repeat > 1 {
		ret["data"] = strings.Repeat(in, repeat)
	} else {
		ret["data"] = in
	}

	_tmp, _ := json.Marshal(ret)
	data.FastReturnBNocopy(_tmp)
	return ""
}
