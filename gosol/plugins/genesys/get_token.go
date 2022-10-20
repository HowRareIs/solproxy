package genesys

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"net/http"
	"strings"
)

type _gt struct {
	token         string
	error_comment string
}

func _signMessage(privatekey string) (string, string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Error signing message for genesys using PK")
		}
	}()

	pk, err := solana.PrivateKeyFromBase58(privatekey)
	fmt.Println("Signing message using PK, Public address:", pk.PublicKey())
	if err != nil {
		fmt.Println("Error processing Solana Private Key")
		fmt.Println(err.Error())
		return "", ""
	}

	msg := "Sign in to GenesysGo Shadow Platform."
	signed_message, err2 := pk.Sign([]byte(msg))
	if err2 != nil {
		fmt.Println("Error signing message using private key")
		fmt.Println(err2.Error())
		return "", ""
	}
	if signed_message.IsZero() {
		fmt.Println("Error signing message using private key (2)")
		return "", ""
	}

	fmt.Println("Signing message:", msg)
	fmt.Println("Signed message:", signed_message)
	return signed_message.String(), pk.PublicKey().String()
}

func (this *Genesys) _getToken(client_id string) (ret _gt) {
	defer func() {
		if err := recover(); err != nil {
			ret.error_comment = "Error getting genesys token"
		}
	}()

	body := "{'message':'" + this.signed_message + "','signer':'" + this.public_key + "'}"
	body = strings.Replace(body, "'", "\"", 999)

	resp, err3 := http.Post(rpc_url+"/signin", "application/json", bytes.NewBuffer([]byte(body)))
	if err3 != nil {
		ret.error_comment = "Error: " + err3.Error()
		return
	}

	var res map[string]interface{}
	err4 := json.NewDecoder(resp.Body).Decode(&res)
	if err4 != nil {
		ret.token = ""
		ret.error_comment = "Error: " + err4.Error()
		return
	}

	token := ""
	switch res["token"].(type) {
	case string:
		token = res["token"].(string)
	default:
		ret.error_comment = "Error: Cannot read sign-in token (1)"
		return
	}
	if len(token) == 0 {
		ret.error_comment = "Error: Sign-in token is empty"
	}

	client := &http.Client{}
	req, _ := http.NewRequest("POST", rpc_url+"/premium/token/"+this.client_id, nil)
	req.Header.Add("Authorization", "Bearer "+token+"")

	resp, err5 := client.Do(req)
	if err5 != nil {
		ret.error_comment = "Error: " + err5.Error()
	}
	res = make(map[string]interface{})
	err6 := json.NewDecoder(resp.Body).Decode(&res)
	if err6 != nil {
		ret.error_comment = "Error: " + err6.Error()
	}

	ret.error_comment = ""
	ret.token = res["token"].(string)
	x := res["token"].(int)
	fmt.Println(">>>>", x)
	return
}
