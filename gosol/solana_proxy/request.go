package solana_proxy

import (
	"bytes"
	"fmt"

	"encoding/json"
	"strconv"
	"strings"
)

func (this *SOLClient) GetFirstAvailableBlock() (int, bool) {

	ret := this.RunRequest("getFirstAvailableBlock")
	if ret == nil {
		return 0, false
	}

	r := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&r)

	switch v := r["result"].(type) {
	case json.Number:
		_ret, err := v.Int64()
		if err != nil {
			break
		}
		return int(_ret), true
	default:
		fmt.Println("Error in response for getFirstAvailableBlock: " + string(ret))
	}
	return 0, false
}

func (this *SOLClient) GetVersion() (int, int, string, bool) {

	ret := this.RunRequest("getVersion")
	if ret == nil {
		return 0, 0, "", false
	}

	type out_result struct {
		Solana_core string `json:"solana-core"`
	}
	type out_main struct {
		Jsonrpc string     `json:"jsonrpc"`
		Result  out_result `json:"result"`
	}

	tmp := &out_main{}
	json.Unmarshal(ret, tmp)

	if len(tmp.Result.Solana_core) == 0 {
		return 0, 0, "", false
	}

	tmp_chunks := strings.Split(tmp.Result.Solana_core, ".")
	version_major, _ := strconv.Atoi(tmp_chunks[0])
	version_minor, _ := strconv.Atoi(tmp_chunks[1])
	return version_major, version_minor, tmp.Result.Solana_core, true
}

func (this *SOLClient) GetBlock(block int) ([]byte, bool) {
	ret := []byte("")
	if this.version_major == 1 && this.version_minor <= 6 {
		ret = this.RunRequestP("getConfirmedBlock", fmt.Sprintf("[%d]", block))
	} else {
		ret = this.RunRequestP("getBlock", fmt.Sprintf("[%d]", block))
	}

	if ret == nil {
		return []byte(`{"error":"No response from server 0x01"}`), false
	}

	v := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&v)

	switch v["result"].(type) {
	case nil:
		return ret, false
	}
	return ret, true
}

func (this *SOLClient) GetTransaction(hash string) ([]byte, bool) {
	params := fmt.Sprintf("[\"%s\"]", hash)
	ret := []byte("")
	if this.version_major == 1 && this.version_minor <= 6 {
		ret = this.RunRequestP("getConfirmedTransaction", params)
	} else {
		ret = this.RunRequestP("getTransaction", params)
	}

	if ret == nil {
		return []byte(`{"error":"No response from server 0x01"}`), false
	}

	v := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&v)

	switch v["result"].(type) {
	case nil:
		return ret, false
	}
	return ret, true
}

func (this *SOLClient) SimpleCall(method, pubkey string, commitment string) ([]byte, bool) {
	params := ""
	if len(commitment) > 0 {
		params = fmt.Sprintf("[\"%s\",\"%s\"]", pubkey, commitment)
	} else {
		params = fmt.Sprintf("[\"%s\"]", pubkey)
	}

	ret := this.RunRequestP(method, params)
	if ret == nil {
		return []byte(`{"error":"No response from server 0x01"}`), false
	}
	return ret, true
}

func (this *SOLClient) GetBalance(pubkey string, commitment string) ([]byte, bool) {
	return this.SimpleCall("getBalance", pubkey, commitment)
}

func (this *SOLClient) GetTokenSupply(pubkey string, commitment string) ([]byte, bool) {
	return this.SimpleCall("getTokenSupply", pubkey, commitment)
}
