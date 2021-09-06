package handler_socket2

import (
	"bytes"
	"compress/flate"
	"fmt"
	"net/url"
	"strings"
)

func runRequest(key string, message_body []byte, is_compressed bool, handler handlerFunc) {
	// compression support!

	if CfgIsDebug() {
		fmt.Println("FROM UDP: ", string(message_body))
	}

	curr_msg := ""
	if is_compressed {

		r := flate.NewReader(bytes.NewReader(message_body))

		buf := new(bytes.Buffer)
		buf.ReadFrom(r)
		curr_msg = buf.String()
		r.Close()

	} else {
		curr_msg = string(message_body)
	}
	// <<

	// connection debugging
	_req_txt := ""
	_conn_data, _ := url.QueryUnescape(curr_msg)
	if len(_conn_data) > 60 {

		_pos := 0
		_conn_data_wbr := ""
		for _pos < len(_conn_data) {
			_end := _pos + 80
			if _end > len(_conn_data) {
				_end = len(_conn_data)
			}
			_conn_data_wbr += _conn_data[_pos:_end] + "<wbr>"
			_pos += 80
		}

		_req_txt = "<span class='tooltip'>[...] " + _conn_data[0:60] + "<div>" + _conn_data_wbr + "</div></span>"
	} else {
		_req_txt = "<span>" + _conn_data + "</span>"
	}
	// <<

	// process packet - parameters
	params, _ := url.ParseQuery(curr_msg)
	fmt.Print(params)

	params2 := make(map[string]string)
	for _k, _v := range params {
		params2[_k] = implode(",", _v)
	}
	// <<

	udpStatRequest(key, params2["action"], _req_txt)

	data := ""
	action := ""
	action_specified := false
	for {

		action, action_specified = params2["action"]
		if !action_specified || action == "" {
			action = "default"
		}

		hsparams := CreateHSParamsFromMap(params2)
		data = handler(hsparams)
		if hsparams.fastreturn != nil {
			data = string(hsparams.fastreturn)
		}

		//fmt.Println(data)
		if !strings.HasPrefix(data, "X-Forward:") {

			hsparams.Cleanup()
			break
		}

		var redir_vals url.Values
		var err error
		if redir_vals, err = url.ParseQuery(data[10:]); err != nil {

			data = "X-Forward, wrong redirect" + data

			hsparams.Cleanup()
			break
		}

		params2 = make(map[string]string)
		for rvk, _ := range redir_vals {
			params2[strings.TrimLeft(rvk, "?")] = redir_vals.Get(rvk)
		}

		hsparams.Cleanup()
		fmt.Println(params2)
	}
}

// return number of bytes we need to move forward
// < -1 error - flush the buffer
// 0 - needs more data to process the request
// > 1 message processed correctly - move forward
func processRequestData(key string, message []byte, handler handlerFunc) (int, []byte, bool) {

	var terminator = []byte("\r\n\r\n")
	const terminator_len = len("\r\n\r\n")

	lenf := bytes.Index(message, terminator)
	if lenf == -1 {
		return 0, nil, false
	}

	// request is terminated - we can start to process it!
	bytes_rec_uncompressed := 0
	is_compressed, required_size, guid := processHeader(message[0:lenf])
	fmt.Println("GUID: ", guid)

	if required_size < 0 {
		return -1, nil, false
	}

	// we also need to account for length header
	required_size += lenf + terminator_len

	// still need more data to arrive?
	if len(message) < required_size {
		return 0, nil, false
	}

	bytes_rec_uncompressed++
	return required_size, message[lenf+terminator_len : required_size], is_compressed
}
