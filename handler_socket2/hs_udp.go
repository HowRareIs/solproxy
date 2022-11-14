package handler_socket2

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
	"net"
	"sync"
	"time"
)

type udpRequest struct {
	action     string
	req        string
	start_time int64
	end_time   int64
	status     string

	request_no           int
	sub_requests_pending int
}

var udpStatMutex sync.Mutex
var udpStats map[string]*udpRequest

func init() {
	udpStats = make(map[string]*udpRequest)
	init_cleaners(3, 5, func(cleanup []string) []string {

		ret := make([]string, 0)

		// clean stats
		udpStatMutex.Lock()
		for _, v := range cleanup {
			if tmp, ok := udpStats[v]; ok {
				if tmp.status == "F" || tmp.status == "X" {
					delete(udpStats, v)
				} else {
					ret = append(ret, v)
				}
			}
		}
		udpStatMutex.Unlock()

		return ret
	})
}

func udpStatBeginRequest(rec_id string, request_no int) {
	udpStatMutex.Lock()
	udpStats[rec_id] = &udpRequest{start_time: time.Now().UnixNano(), request_no: request_no, status: "O"}
	udpStatMutex.Unlock()
}

func udpStatRequest(rec_id, action string, request string) {
	udpStatMutex.Lock()
	if info, exists := udpStats[rec_id]; exists {
		info.action = action
		info.req = request
		info.sub_requests_pending++
		info.status = "P"
	}
	udpStatMutex.Unlock()
}

func udpStatFinishRequest(rec_id string, is_ok bool) {

	status := "F"
	if !is_ok {
		status = "X"
	}

	udpStatMutex.Lock()
	if info, exists := udpStats[rec_id]; exists {
		info.sub_requests_pending--
		fmt.Println("TASK PENDING FOR CLEANUP: ", udpStats[rec_id], is_ok)

		if !is_ok || info.sub_requests_pending <= 0 {
			info.status = status
			info.end_time = time.Now().UnixNano()

			cleaners_insert(rec_id)
		}
	}
	udpStatMutex.Unlock()
}

func GetStatusUDP() string {

	action_count := make(map[string]int)
	time_now := time.Now().UnixNano()
	ret := ""

	udpStatMutex.Lock()
	for k, v := range udpStats {

		_key := "<b>[" + v.status + "]</b> " + v.action
		action_count[_key]++
		if action_count[_key] >= 5 {
			continue
		}

		tmp := "<div class='thread_list'>"
		status := v.status
		if status == "P" {
			status = fmt.Sprintf("%s/%d", status, v.sub_requests_pending)
		}

		_took := float64(time_now-v.start_time) / float64(1000000)
		if status == "F" {
			_took = float64(v.end_time-v.start_time) / float64(1000000)
		}

		tmp += fmt.Sprintf("<span>Num. %d - %s</span> - <b>[%s]</b> %.3fms - %s\n", v.request_no, k, status, _took, v.req)
		tmp += "</div>\n"
		ret += tmp
	}
	udpStatMutex.Unlock()

	for action, count := range action_count {
		count -= 5
		if count > 0 {
			ret += fmt.Sprintf("<div class='thread_list'><span> Aggr x%d:</span> %s</div>", count, action)
		}
	}

	return ret
}

func startServiceUDP(bindTo string, handler handlerFunc) {

	udpAddr, err := net.ResolveUDPAddr("udp", bindTo)
	if err != nil {
		fmt.Printf("Error resolving address: %s, %s\n", bindTo, err)
		return
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Printf("Error listening on UDP address: %s, %s\n", bindTo, err)
		return
	}

	fmt.Printf("UDP Service started : %s\n", bindTo)
	boundMutex.Lock()
	boundTo = append(boundTo, "udp:"+bindTo)
	boundMutex.Unlock()

	req_no := 0
	source_buffer := make(map[string][]byte)
	func() {

		buffer := make([]byte, 65536)
		for {
			n, udparr, err := listener.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			key := udparr.String()
			if _, ok := source_buffer[key]; !ok {

				_tmp := make([]byte, n)
				copy(_tmp, buffer[:n])
				source_buffer[key] = _tmp

			} else {
				source_buffer[key] = append(source_buffer[key], buffer[:n]...)
			}

			req_no++
			message := source_buffer[key]

			// version 2 UDP protocol!
			v2_protocol := len(message) > 0 && message[0] == 'B' || message[0] == 'b'
			if v2_protocol {

				for {
					move_forward, msg_body, is_compressed := processRequestDataV2(key, message, handler)

					// error in message, delete all data!
					if move_forward == -1 {
						udpStatBeginRequest(key, req_no)
						message = message[:0]
						udpStatFinishRequest(key, false)
						break
					}

					// packet processed correctly, process the message!
					if move_forward > 0 {

						udpStatBeginRequest(key, req_no)
						go func(key string, msg_body []byte, is_compressed bool) {
							is_ok := runRequestV2(key, msg_body, is_compressed, handler)
							udpStatFinishRequest(key, is_ok)
						}(key, msg_body, is_compressed)

						message = message[move_forward:]
					}

					// we need more data, if
					// 1. engine needs it
					// 2. data is empty
					if move_forward == 0 || len(message) == 0 {
						break
					}
				}

				if len(message) == 0 {
					delete(source_buffer, key)
				} else {
					source_buffer[key] = message
				}
			}

			if v2_protocol {
				continue
			}

			for {

				move_forward, msg_body, is_compressed := processRequestData(key, message, handler)

				// error in message, delete all data!
				if move_forward == -1 {
					message = message[:0]
					udpStatFinishRequest(key, false)
					break
				}

				// packet processed correctly, process the message!
				if move_forward > 0 {

					go func(key string, msg_body []byte, is_compressed bool) {
						runRequest(key, msg_body, is_compressed, handler)
						udpStatFinishRequest(key, true)
					}(key, msg_body, is_compressed)

					message = message[move_forward:]
					if len(message) == 0 {
						break
					}
				}

				// we need more data!
				if move_forward == 0 {
					break
				}

			}

			if len(message) == 0 {
				delete(source_buffer, key)
			} else {
				source_buffer[key] = message
			}
		}
	}()
}

// return number of bytes we need to move forward
// < -1 error - flush the buffer
// 0 - needs more data to process the request
// > 1 message processed correctly - move forward
func processRequestDataV2(key string, message []byte, handler handlerFunc) (int, []byte, bool) {

	if len(message) < 5 {
		return 0, nil, false
	}

	size := int(binary.LittleEndian.Uint32(message[1:5]))
	is_compressed := message[0] == 'B'

	if size < 0 {
		return -1, nil, false
	}

	if len(message) >= size {
		return size, message[5:size], is_compressed
	}

	return 0, nil, false
}

func runRequestV2(key string, message_body []byte, is_compressed bool, handler handlerFunc) bool {

	// compression support!
	if is_compressed {

		r := flate.NewReader(bytes.NewReader(message_body))
		buf := new(bytes.Buffer)
		buf.ReadFrom(r)
		message_body = buf.Bytes()
		r.Close()

	}
	// <<

	if config.CfgIsDebug() {
		fmt.Println("FROM UDP: ", string(message_body))
	}

	hsparams := CreateHSParams()
	guid := ReadHSParams(message_body, hsparams)
	if guid == nil {
		hsparams.Cleanup()
		return false
	}

	udpStatRequest(key, hsparams.GetParam("action", "?"), hsparams.getParamInfoHTML())
	data := handler(hsparams)
	if config.CfgIsDebug() {
		fmt.Println(string(guid), ">>", data)
	}
	hsparams.Cleanup()

	return true
}

type stat_cleaner struct {
	ids map[string]bool
	mu  sync.Mutex
}

var sc_mutex sync.Mutex
var item_pos int
var stat_cleaners []stat_cleaner
var numCleaners int

func init_cleaners(num_pieces, second_per_piece int, callback func([]string) []string) {

	numCleaners = num_pieces
	stat_cleaners = make([]stat_cleaner, num_pieces)
	for k, _ := range stat_cleaners {
		stat_cleaners[k] = stat_cleaner{ids: make(map[string]bool)}
	}

	go func() {
		time_last := -1
		for _ = range time.Tick(200 * time.Millisecond) {
			now := int(time.Now().UnixNano() / 1000000000)
			if now == time_last {
				continue
			}

			sc_mutex.Lock()
			item_pos = (now / second_per_piece) % numCleaners
			sc_mutex.Unlock()
			time_last = now

			_cbdata := cleaners_get()
			go func(_cbdata []string) {
				ret := callback(_cbdata)
				/*fmt.Println("====", item_pos)
				fmt.Println(_cbdata)*/

				cleaners_insert(ret...)

			}(_cbdata)
		}

	}()
}

func cleaners_get() []string {
	sc_mutex.Lock()
	pos := item_pos
	sc_mutex.Unlock()

	tmp := &stat_cleaners[(pos+1)%numCleaners]

	tmp.mu.Lock()
	ret := tmp.ids
	tmp.ids = make(map[string]bool)
	tmp.mu.Unlock()

	ret2 := make([]string, 0, len(ret))
	for k, _ := range ret {
		ret2 = append(ret2, k)
	}

	return ret2
}

func cleaners_insert(ids ...string) {
	sc_mutex.Lock()
	pos := item_pos
	sc_mutex.Unlock()

	tmp := &stat_cleaners[pos]
	tmp.mu.Lock()
	for _, id := range ids {
		tmp.ids[id] = true
	}
	tmp.mu.Unlock()
}
