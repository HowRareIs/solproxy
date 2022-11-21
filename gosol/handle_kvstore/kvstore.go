package handle_kvstore

import (
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2"
)

type Handle_kvstore struct {
}

func init() {
	for i := 0; i < len(pools); i++ {
		pools[i] = &pool{data: make(map[string]item)}
	}
}

func (this *Handle_kvstore) Initialize() {
	handler_socket2.StatusPluginRegister(func() (string, string) {
		ret := "KV Storage Plugin is Enabled\n"
		for i := 0; i < len(pools); i++ {
			pools[i].mu.RLock()
			ret += fmt.Sprintf("Pool #%d, keys: %d\n", i, len(pools[i].data))
			pools[i].mu.RUnlock()
		}
		return "KV Storage", "<pre>" + ret + "</pre>"
	})
}

func (this *Handle_kvstore) Info() string {
	return "Basic KV storage plugin"
}

func (this *Handle_kvstore) GetActions() []string {
	return []string{"keyGet", "keySet"}
}

func (this *Handle_kvstore) HandleAction(action string, data *handler_socket2.HSParams) string {

	k := data.GetParam("k", "")
	if len(k) == 0 {
		ret := "?k=key name (required).<br>"
		ret += "Set options:<br>"
		ret += "v=value (keep empty to delete the key)<br>"
		ret += "ttl=time to live (seconds)<br>"
	}

	if action == "keySet" {
		KeySet(k, []byte(data.GetParam("v", "")), data.GetParamI("ttl", 3600), false)
		return "{\"result\":\"ok\"}"
	}
	if action == "keyGet" {
		i := keyGet(k, []byte{})
		if i.is_sensitive {
			return "Cannot retrieve sensitive data"
		}
		data.FastReturnBNocopy(i.data)
		return ""
	}

	return "No function?!"
}
