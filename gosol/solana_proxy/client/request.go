package client

/* This file adds support for requests on higher level, without error processing */

import (
	"bytes"
	"fmt"
	"time"

	"encoding/json"
	"strconv"
	"strings"
)

func (this *SOLClient) _intcall(method string) (int, ResponseType) {
	ret, r_type := this.RequestBasic(method)
	if ret == nil {
		return 0, r_type
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
		return int(_ret), r_type
	default:
		fmt.Println("Error in response for " + method + ": " + string(ret))
	}
	return 0, R_ERROR
}

func (this *SOLClient) GetFirstAvailableBlock() (int, ResponseType) {
	ret, r_type := this._intcall("getFirstAvailableBlock")
	if r_type == R_OK {
		_ts := time.Now().UnixMilli()
		this.mu.Lock()
		this.available_block_first = ret
		this.available_block_first_ts = _ts
		this.mu.Unlock()
	}
	return ret, r_type
}

func (this *SOLClient) GetLastAvailableBlock() (int, ResponseType) {
	ret, r_type := this._intcall("getBlockHeight")
	if r_type == R_OK {
		_ts := time.Now().UnixMilli()
		this.mu.Lock()
		this.available_block_last = ret
		this.available_block_last_ts = _ts
		this.mu.Unlock()
	}
	return ret, r_type
}

func (this *SOLClient) GetVersion() (int, int, string, ResponseType) {

	ret, r_type := this.RequestBasic("getVersion")
	if ret == nil {
		return 0, 0, "", r_type
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
		fmt.Println("Error in response for GetVersion: can't find solana core")
		return 0, 0, "", R_ERROR
	}

	tmp_chunks := strings.Split(tmp.Result.Solana_core, ".")
	version_major, _ := strconv.Atoi(tmp_chunks[0])
	version_minor, _ := strconv.Atoi(tmp_chunks[1])

	_ts := time.Now().UnixMilli()
	this.mu.Lock()
	this.version = tmp.Result.Solana_core
	this.version_ts = _ts
	this.mu.Unlock()
	return version_major, version_minor, tmp.Result.Solana_core, R_OK
}

func (this *SOLClient) GetBlock(block int) ([]byte, ResponseType) {
	ret := []byte("")
	r_type := R_OK
	if this.version_major == 1 && this.version_minor <= 6 {
		ret, r_type = this.RequestBasic("getConfirmedBlock", fmt.Sprintf("[%d]", block))
	} else {
		ret, r_type = this.RequestBasic("getBlock", fmt.Sprintf("[%d]", block))
	}
	if ret == nil {
		return ret, r_type
	}

	v := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&v)

	switch v["result"].(type) {
	case nil:
		fmt.Sprintf("Warning: Cannot get block %d using endpoint %s. Probably the node doesn't have data.\n", block, this.endpoint)
		return ret, R_ERROR
	}
	return ret, R_OK
}

func (this *SOLClient) GetTransaction(hash string) ([]byte, ResponseType) {
	params := fmt.Sprintf("[\"%s\"]", hash)
	ret := []byte("")
	r_type := ResponseType(R_OK)
	if this.version_major == 1 && this.version_minor <= 6 {
		ret, r_type = this.RequestBasic("getConfirmedTransaction", params)
	} else {
		ret, r_type = this.RequestBasic("getTransaction", params)
	}
	if ret == nil {
		return ret, r_type
	}

	// genesys patch, redo transaction if it fails for given transaction id
	if len(ret) < 200 &&
		bytes.Index(ret, ([]byte)("\"result\":null")) > -1 &&
		strings.Index(this.endpoint, "genesysgo") > -1 {
		ret, r_type = this.RequestBasic("getTransaction", params)
		if ret == nil {
			return ret, r_type
		}
	}

	v := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewReader(ret))
	dec.UseNumber()
	dec.Decode(&v)

	switch v["result"].(type) {
	case nil:
		return ret, R_ERROR
	}
	return ret, R_OK
}

func (this *SOLClient) SimpleCall(method, pubkey string, commitment string) ([]byte, ResponseType) {
	params := ""
	if len(commitment) > 0 {
		params = fmt.Sprintf("[\"%s\",\"%s\"]", pubkey, commitment)
	} else {
		params = fmt.Sprintf("[\"%s\"]", pubkey)
	}

	return this.RequestBasic(method, params)
}

func (this *SOLClient) GetBalance(pubkey string, commitment string) ([]byte, ResponseType) {
	return this.SimpleCall("getBalance", pubkey, commitment)
}

func (this *SOLClient) GetTokenSupply(pubkey string, commitment string) ([]byte, ResponseType) {
	return this.SimpleCall("getTokenSupply", pubkey, commitment)
}
