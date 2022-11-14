package handler_socket2

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/slawomir-pryczek/HSServer/handler_socket2/byteslabs"
)

type HSParams struct {
	param      map[string][]byte
	porder     []string
	fastreturn []byte

	additional_resp_headers []string

	allocator *byteslabs.Allocator
}

func CreateHSParams() *HSParams {

	ret := HSParams{}
	ret.Cleanup()

	return &ret
}

func CreateHSParamsFromMap(data map[string]string) *HSParams {

	ret := &HSParams{param: make(map[string][]byte), porder: make([]string, 0, len(data))}
	for k, v := range data {
		ret.param[k] = []byte(v)
		ret.porder = append(ret.porder, k)
	}

	return ret
}

func ReadHSParams(message []byte, out_params *HSParams) []byte {

	// read guid from the message
	_mlen := len(message)
	pos := 0

	if pos+2 >= _mlen {
		return nil
	}
	_guid_size := int(binary.LittleEndian.Uint16(message[pos : pos+2]))
	pos += 2

	if pos+_guid_size >= _mlen {
		return nil
	}

	guid := make([]byte, _guid_size)
	copy(guid, message[pos:pos+_guid_size])
	pos += _guid_size

	// Read message parameters
	message = message[pos:]
	pos = 0
	_mlen = len(message)
	if _mlen == 0 {
		return nil
	}

	for pos < len(message) {

		if pos+2 >= _mlen {
			return nil
		}
		_k_s := int(binary.LittleEndian.Uint16(message[pos : pos+2]))
		pos += 2

		if pos+4 >= _mlen {
			return nil
		}
		_v_s := int(binary.LittleEndian.Uint32(message[pos : pos+4]))
		pos += 4

		if pos+_k_s+_v_s > _mlen { // for lop will break, as message could be over now if pos+everything equals mlen
			return nil
		}
		k := message[pos : pos+_k_s]
		pos += _k_s
		v := message[pos : pos+_v_s]
		pos += _v_s

		//vcopy := make([]byte, _v_s)
		//copy(vcopy, v)

		_vstr := string(k)
		out_params.param[_vstr] = v
		out_params.porder = append(out_params.porder, _vstr)
	}

	return guid
}

func (p *HSParams) SetRespHeader(attr, val string) {
	p.additional_resp_headers = append(p.additional_resp_headers, attr+":"+val+"\n")
}

func (p *HSParams) SetParam(attr string, val string) {
	p.param[attr] = []byte(val)
}

func (p *HSParams) GetParam(attr string, def string) string {

	if val, ok := p.param[attr]; ok {
		return string(val)
	}

	return def
}

func (p *HSParams) GetParamsS() map[string]string {

	ret := make(map[string]string, len(p.param))
	for k, v := range p.param {
		ret[k] = string(v)
	}

	return ret

}

// this is not safe because it can use memory shared between requests, so we can't
// keep this after request is over!
func (p *HSParams) GetParamBUnsafe(attr string, def []byte) []byte {

	if val, ok := p.param[attr]; ok {
		return val
	}

	return def
}

func (p *HSParams) GetParamA(attr string, separator string) []string {

	if val, ok := p.param[attr]; ok {
		return strings.Split(string(val), ",")
	}

	return []string{}
}

func (p *HSParams) GetParamIA(attr string) []int {

	ret := make([]int, 0)
	if val, ok := p.param[attr]; ok {
		for _, v := range strings.Split(string(val), ",") {

			if vi, ok := strconv.Atoi(v); ok == nil {
				ret = append(ret, vi)
			}

		}
	}

	return ret
}

func (p *HSParams) GetParamI(attr string, def int) int {

	_ps := p.param[attr]
	if len(_ps) == 0 {
		return def
	}
	if vi, err := strconv.Atoi(string(_ps)); err == nil {
		return vi
	}
	return def
}

func (p *HSParams) getParamInfoHTML() string {

	_req_txt := ""

	_conn_data := p.getParamInfo()
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

	return _req_txt
}

func (p *HSParams) getParamInfo() string {

	ret := ""
	for _, k := range p.porder {

		v := p.param[k]

		limit := len(v)
		add := ""
		if len(v) > 500 {
			limit = 500
			add = "..."
		}

		ret += string(k) + "=" + string(v[0:limit]) + add + "&"
	}

	return ret
}

func (p *HSParams) FastReturnBNocopy(set []byte) {

	p.fastreturn = set
}

func (p *HSParams) FastReturnB(set []byte) {

	// if the data is very short - don't use the allocator!
	len_set := len(set)
	if len_set < 64 || (p.allocator == nil && len_set < 128) {
		ret := make([]byte, len_set, len_set)
		copy(ret, set)
		p.fastreturn = ret
		return
	}

	if p.allocator == nil {
		p.allocator = byteslabs.MakeAllocator()
	}

	b := p.allocator.Allocate(len(set))
	b = b[0:cap(b):cap(b)]
	copy(b, set)
	p.fastreturn = b
}

func (p *HSParams) FastReturnS(set string) {

	if p.allocator == nil {
		p.allocator = byteslabs.MakeAllocator()
	}
	buff := bytes.NewBuffer(p.allocator.Allocate(len(set)))
	buff.WriteString(set)
	p.fastreturn = buff.Bytes()
}

func (p *HSParams) GetAllocator() *byteslabs.Allocator {
	if p.allocator == nil {
		p.allocator = byteslabs.MakeAllocator()
	}
	return p.allocator
}

func (p *HSParams) Allocate(size int) []byte {
	if p.allocator == nil {
		p.allocator = byteslabs.MakeAllocator()
	}
	return p.allocator.Allocate(size)
}

func (p *HSParams) Cleanup() {
	if p.allocator != nil {
		p.allocator.Release()
		p.allocator = nil
	}

	p.fastreturn = nil
	p.param = make(map[string][]byte)
	p.porder = make([]string, 0)
	p.additional_resp_headers = make([]string, 0)
}
