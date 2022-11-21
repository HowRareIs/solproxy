package genesys

import (
	"encoding/json"
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
	"gosol/handle_kvstore"
	"gosol/plugins/common"
	"strings"
	"time"
)

const rpc_url = "https://portal.genesysgo.net/api"

type Genesys struct {
	client_id      string
	public_key     string
	signed_message string

	comment         string
	last_updated_ts int64
}

func Init(config_attr string) *common.Plugin {
	ret := &Genesys{}
	id, _ := config.Config().GetSubattrString(config_attr, "client_id")
	if len(id) == 0 {
		return nil
	}
	ret.client_id = id

	pk, _ := config.Config().GetSubattrString(config_attr, "pk")
	if len(pk) > 0 {
		fmt.Println("Solproxy genesys plugin, PK mode")
		ret.signed_message, ret.public_key = _signMessage(pk)

		if len(ret.signed_message) > 0 {
			fmt.Println("Warning: You should copy genesys plugin config from below for production, to not store unencrypted PK")
			fmt.Println("------------------------------------------------------------------")
			cfg := make(map[string]string)
			cfg["client_id"] = ret.client_id
			cfg["public_key"] = ret.public_key
			cfg["msg"] = ret.signed_message
			cfg_json, _ := json.Marshal(cfg)
			fmt.Println(string(cfg_json))
			fmt.Println("------------------------------------------------------------------")
		}
		return common.PluginFactory(ret)
	}

	msg, _ := config.Config().GetSubattrString(config_attr, "msg")
	pubkey, _ := config.Config().GetSubattrString(config_attr, "public_key")
	if len(msg) > 0 && len(pubkey) > 0 {
		fmt.Println("Solproxy genesys plugin, presigned message mode")
		ret.signed_message = msg
		ret.public_key = pubkey
		return common.PluginFactory(ret)
	}

	return nil
}

func (this *Genesys) Run(age_ms int) bool {

	// refresh token every 2 hours
	if age_ms > 3600*2000 || age_ms == -1 {
		_t := this._getToken(this.client_id)
		if len(_t.token) > 0 {
			this.comment = "Received token " + hscommon.StrMidChars(_t.token, 5)
			this.last_updated_ts = time.Now().Unix()
			handle_kvstore.KeySet(fmt.Sprintf("genesys-%s", this.client_id), []byte(_t.token), 0, true)
			return true
		}

		this.comment = _t.error_comment
	}
	return false
}

func (this *Genesys) Status() string {

	ret := make([]string, 0, 20)
	ret = append(ret, fmt.Sprintf("Genesys plugin for Client ID: %s", this.client_id))
	ret = append(ret, fmt.Sprintf(" Signed message: %s", hscommon.StrMidChars(this.signed_message, 3)))
	ret = append(ret, fmt.Sprintf(" Comment: %s", this.comment))

	_age_s := "Never"
	if this.last_updated_ts != 0 {
		_age := time.Now().Unix() - this.last_updated_ts
		_age_s = hscommon.FormatTime(int(_age)) + " ago"
	}
	ret = append(ret, fmt.Sprintf(" Last Successfull Update: %s", _age_s))
	return strings.Join(ret, "\n")
}
