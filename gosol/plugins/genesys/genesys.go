package genesys

import (
	"encoding/json"
	"fmt"
	"github.com/slawomir-pryczek/handler_socket2/config"
	"gosol/plugins/common"
)

const rpc_url = "https://portal.genesysgo.net/api"

type Genesys struct {
	client_id      string
	public_key     string
	signed_message string

	comment string
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

	// refresh token every hour
	if age_ms > 3600*1000 || age_ms == -1 {
		_t := this._getToken(this.client_id)
		if len(_t.token) > 0 {
			return true
		}

		this.comment = _t.error_comment
	}
	return false
}

func (this *Genesys) Status() string {
	return "X"
}

func GetToken() error {

	/*pk, err := solana.PrivateKeyFromBase58("4rZGkEjJ8qcFN7riSZ1V5XcLVoKvsJSPNMVdhsav2BZ9kVbZMxvBxGCAfx5tkc1Ej5Kix1WxNQ2LbhA5fUkwzR4P")
	fmt.Println("Authenticating using PK, Public: ", pk.PublicKey())
	if err != nil {
		return err
	}

	signed_message, err2 := pk.Sign([]byte("Sign in to GenesysGo Shadow Platform."))
	fmt.Println("Signed message: ", signed_message)
	if err2 != nil {
		return err2
	}*/
	/*
		body := "{'message':'" + signed_message.String() + "','signer':'" + pk.PublicKey().String() + "'}"
		body = strings.Replace(body, "'", "\"", 999)
		fmt.Println(body)

		resp, err3 := http.Post(rpc_url+"/signin", "application/json", bytes.NewBuffer([]byte(body)))
		if err3 != nil {
			return err3
		}

		var res map[string]interface{}
		err4 := json.NewDecoder(resp.Body).Decode(&res)
		if err4 != nil {
			return err4
		}

		token := ""
		switch res["token"].(type) {
		case string:
			token = res["token"].(string)
		default:
			return errors.New("Cannot read token (1)")
		}
		if len(token) == 0 {
			return errors.New("Token is empty!")
		}

		client := &http.Client{}
		req, _ := http.NewRequest("POST", rpc_url+"/premium/token/c26fe6cb-a2be-4f61-b1de-84823e68572e", nil)
		req.Header.Add("Authorization", "Bearer "+token+"")

		fmt.Println(res)
		fmt.Println("Auth Token: ", token)

		fmt.Println("---------->>><<<<-----------")
		resp, err5 := (client.Do(req))
		if err5 != nil {
			return err5
		}
		err6 := json.NewDecoder(resp.Body).Decode(&res)
		if err6 != nil {
			return err6
		}
		fmt.Println(res)
	*/
	return nil
}
