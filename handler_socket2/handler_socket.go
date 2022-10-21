package handler_socket2

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/slawomir-pryczek/handler_socket2/byteslabs"
	"github.com/slawomir-pryczek/handler_socket2/compress"
	"github.com/slawomir-pryczek/handler_socket2/config"
	"github.com/slawomir-pryczek/handler_socket2/hscommon"
	"github.com/slawomir-pryczek/handler_socket2/oslimits"
	"github.com/slawomir-pryczek/handler_socket2/stats"
)

const version = "HSServer v4"

var uptime_started int

var compressor_snappy *compress.Compressor = nil
var compressor_flate *compress.Compressor = nil

func init() {
	oslimits.SetOpenFilesLimit(262144)
	uptime_started = int(time.Now().UnixNano()/1000000000) - 1 // so we won't divide by 0

	compression_support := config.Config().Get("COMPRESSION", "mp-flate")
	if strings.Index(compression_support, "mp-flate") > -1 {
		compressor_flate = compress.CreateCompressor(runtime.NumCPU(), compress.MakeFlate())
	}
	if strings.Index(compression_support, "mp-snappy") > -1 {
		compressor_snappy = compress.CreateCompressor(runtime.NumCPU(), compress.MakeSnappy())
	}

	if compressor_flate == nil && compressor_snappy == nil {
		fmt.Println("Multipart compression is disabled, use compression_support=[mp-flate,mp-snappy] to enable")
	} else {
		fmt.Println("Multipart compression is enabled")
	}

	_comp_status := func() (string, string) {

		ret := "<pre>-- Simple Compress\n"
		ret += compress.CompressSimpleStatus()
		ret += "</pre>"
		if compressor_flate == nil && compressor_snappy == nil {
			return "Compression Plugins (multithreaded compression is disabled)", ret
		}

		ret += "<br><pre>-- Multipart Compress (multi threaded)\n"
		ret += "Multipart compression is used to quickly set compressed data using internal framing format\n"
		ret += "It is not compatible with standard compression schemas.\n\n"
		ret += "E-RLow - error, compression ratio too low\tE-Buffer - error, compression buffer too small\n"
		if compressor_flate != nil {
			ret += compressor_flate.GetStatus()
		}
		if compressor_snappy != nil {
			ret += compressor_snappy.GetStatus()
		}
		ret += "</pre>"

		return "Compression Plugins", ret
	}

	StatusPluginRegister(_comp_status)
}

func GetStatus() map[string]string {

	uptime := int(time.Now().UnixNano()/1000000000) - uptime_started

	all_actions := make([]string, 0)
	for _, v := range action_handlers {
		for _, vv := range v.GetActions() {
			all_actions = append(all_actions, vv)
		}
	}

	ret := stats.GetStatus(all_actions, uptime)

	// get HTTP status
	ret["http_threadlist"] = GetStatusHTTP()
	// <<

	// get UDP status
	ret["udp_threadlist"] = GetStatusUDP()
	// <<

	// calculate uptime, plus other small metrics
	ret["_version"] = version
	ret["_bound_to"] = strings.Join(boundTo, ", ")
	ret["uptime"] = hscommon.FormatTime(uptime)
	// <<

	return ret
}

func getStreamBuffer(stream, recv_buffer []byte, b1 []byte, params *HSParams) ([]byte, uint32, bool) {

	size := int64(-1)
	head := byte(0)

	stream_len := len(stream)
	if size == -1 && stream_len >= 5 {
		size = int64(binary.LittleEndian.Uint32(stream[1:5]))
		head = stream[0]
	}

	recv_b_len := len(recv_buffer)
	if size == -1 && recv_b_len >= 5 && stream_len == 0 {
		size = int64(binary.LittleEndian.Uint32(recv_buffer[1:5]))
		head = recv_buffer[0]
	}

	if size == -1 && stream_len+recv_b_len >= 5 {
		d := make([]byte, 0, 16)
		d = append(d, stream...)
		d = append(d, recv_buffer...)

		size = int64(binary.LittleEndian.Uint32(d[1:5]))
		head = d[0]
	}

	if size > -1 {

		if size > 1024*1024*16 {
			size = 1024 * 1024 * 16
		}

		if head == 'B' || head == 'b' {

			if int(size) < cap(b1) {
				b1 = b1[:0]
				b1 = append(b1, stream...)

				return b1, uint32(size), true
			}

			// Use slab allocator, big optimization, it'll save TB's of
			// allocations and cleanups, BUT we need to allocate <= packet SIZE
			// because this buffer is freed at the end of request,
			// so we can't allow anything to persist in this buffer!
			ret := params.Allocate(int(size)) //make([]byte, 0, size)
			ret = append(ret, stream...)
			return ret, uint32(size), true
		}

		return []byte{}, uint32(size), false
	}

	return nil, 0, false
}

func processHeader(headerData []byte) (bool, int, string) {

	var is_compressed bool = false
	var guid string = ""
	var content_length int = -1
	var err error

	if headerData[0] == '!' {
		is_compressed = true
		headerData = headerData[1:]
	}

	has_guid := bytes.IndexByte(headerData, '|')
	if has_guid > -1 {
		guid = string(headerData[has_guid+1:])
		headerData = headerData[0:has_guid]
	}

	content_length, err = strconv.Atoi(string(headerData))
	if err != nil {
		return false, -1, ""
	}

	return is_compressed, content_length, guid
}

func serveSocket(conn *net.TCPConn, handler handlerFunc) {

	// add new connection struct
	newconn := stats.MakeConnection(conn.RemoteAddr().String())

	// better error support!
	defer func(c *stats.Connection) {
		if e := recover(); e != nil {

			var to_log = ""

			to_log += "###################\n"
			to_log += "#     ERROR!!     #\n"
			to_log += "###################\n"
			to_log += "\n"
			to_log += fmt.Sprintf("Request: %v\n\n", *c) + "\n"
			to_log += "Program panic:\n"

			trace := make([]byte, 4096)
			count := runtime.Stack(trace, true)

			if err, ok := e.(error); ok {
				to_log += err.Error()
			}

			if err, ok := e.(string); ok {
				to_log += "Raw err info: " + err
			}

			to_log += "\n\n"
			to_log += fmt.Sprintf("Stack of %d bytes:\n %s\n", count, trace)
			to_log += "\n"

			to_log += "###################"

			fmt.Println(to_log)

			for _, v := range strings.Split(to_log, "\n") {
				fmt.Fprintln(os.Stderr, v)
			}

			var _ = ioutil.WriteFile("panic"+time.Now().Format("20060201_150405")+".txt", []byte(to_log), 0644)

			os.Exit(1)
		}
	}(newconn)
	// <<

	conn.SetKeepAlive(true)
	conn.SetNoDelay(true)

	message := []byte{}
	data_stream := []byte{}
	buffer := make([]byte, 1024*16)
	sharedmem := make([]byte, 1024*32)
	t_start := int64(0)

	// get configuration and some connection specific data, like compression we should use
	conn_ex := make_conn_ex(conn)
	fmt.Println("In ServeSocket <- ", conn.RemoteAddr(), "Network distance:", conn_ex.remote_distance, "Compression threshold:", conn_ex.compression_threshold)

	params := CreateHSParams()
	for {
		n := 0

		// we don't have new header yet, read next data packet
		if len(data_stream) < 5 {

			nr, err := conn.Read(buffer)
			if err != nil {

				if err != io.EOF {
					newconn.Close(fmt.Sprintf("Read error: %s", err), true)
					break // we have an error - break always!
				}

				// EOF, break the connection, ONLY if we don't have more data to process!
				if nr == 0 {
					break
				}
			}

			n = nr
		}

		if t_start == 0 {
			t_start = newconn.StateReading()
		}

		// we need at least 5 bytes of data to pass this step, here we'll read
		// the message length to pre-allocate its buffer
		message_len := uint32(0)
		if tmp_buff, tmp_size, header_ok := getStreamBuffer(data_stream, buffer[:n], sharedmem, params); tmp_buff != nil {
			data_stream = tmp_buff
			message_len = tmp_size

			// check the header
			if !header_ok {
				newconn.Close("Header type is incorrect", true)
				break
			}

		} else {
			data_stream = append(data_stream, buffer[:n]...)
			continue
		}

		data_stream = append(data_stream, buffer[:n]...)

		// read the remaining data for current packet!
		for uint32(len(data_stream)) < message_len {

			n, err := conn.Read(buffer)
			if err != nil {
				if err != io.EOF {
					newconn.Close(fmt.Sprintf("Read2 error: %s", err), true)

					data_stream = nil
					break // we have an error - break always!
				}

				// EOF break the connection, ONLY if we don't have more data to process!
				if n == 0 {
					data_stream = nil
					break
				}
			}

			data_stream = append(data_stream, buffer[:n]...)
		}

		if data_stream == nil {
			break
		}

		// read message and truncate the length
		is_compressed := data_stream[0] == 'B'
		message = data_stream[5:message_len]
		data_stream = data_stream[message_len:]
		// <<

		bytes_rec_uncompressed := 0
		if is_compressed {

			r := flate.NewReader(bytes.NewReader(message))
			buf := new(bytes.Buffer)
			buf.ReadFrom(r)
			bytes_rec_uncompressed = buf.Len()
			message = buf.Bytes()
			r.Close()

			if message == nil {
				newconn.Close("Cannot decompres", true)
				break
			}

		} else {

			bytes_rec_uncompressed = int(message_len)
		}

		// We have MESSAGE decoded, start serving the request!
		guid := ReadHSParams(message, params)
		if guid == nil {

			newconn.Close("Cannot decode header", true)
			break
		}

		action := params.GetParam("action", "?")
		_pinfo := params.getParamInfo()
		newconn.StateServing(action, _pinfo)

		if config.CfgIsDebug() {
			fmt.Println("Rec:", bytes_rec_uncompressed, "b GUID:", string(guid))
		}

		// remove the message, clean the stream if possible!
		message = []byte{}
		if len(data_stream) == 0 {
			data_stream = []byte{}
		}

		tmp := ""
		if action == "conn-ex" {
			// special case, conn-ex handler is always available
			handle_conn_ex(params, &conn_ex)
		} else {
			tmp = handler(params)
		}

		response := []byte(nil)
		if len(tmp) > 2000 && params.fastreturn == nil {

			// if the return string is long, just use fast mode!
			if len(tmp) > cap(buffer) {
				params.FastReturnS(tmp)
				response = params.fastreturn
			}

			// if the string is short - re-use read buffer to return
			if len(tmp) <= cap(buffer) {
				xb := bytes.NewBuffer(buffer[:0])
				xb.WriteString(tmp)
				response = xb.Bytes()
			}
		}

		if response == nil {
			if params.fastreturn != nil {
				response = params.fastreturn
			} else {
				response = []byte(tmp)
			}
		}
		response_len := uint64(len(response))

		// ##############################################################################
		// <<< Stats code
		skip_response_sent := params.GetParamBUnsafe("__skipsendback", nil) != nil
		took := newconn.StateWriting(uint64(bytes_rec_uncompressed), uint64(message_len), response_len)
		// <<< Stats code ends

		_sent_bytes := int(response_len)
		if !skip_response_sent {
			_sent_bytes = sendBack(conn_ex, params, response, int(took), guid, sharedmem[:0])
		}

		// ##############################################################################
		// <<< Stats code
		newconn.StateKeepalive(uint64(_sent_bytes), took, skip_response_sent)
		// <<< Stats code ends

		t_start = 0
		params.Cleanup()
	}

	conn.Close()
	newconn.Close("Connection Closed OK", false)

}

func sendBack(conn_ex conninfo, params *HSParams, data []byte, took int, guid []byte, sharedmem []byte) int {

	conn := conn_ex.conn

	buffer := bytes.NewBuffer(sharedmem)
	/*h := sha1.New()
	h.Write([]byte(data))
	fingerprint := fmt.Sprintf("%x", h.Sum(nil))
	resp += "Fingerprint: " + fingerprint + "\r\n"
	*/

	buffer.WriteString(fmt.Sprintf("Request 200 OK\r\nServer: %s\r\nContent-Type: text/html\r\nTook:%dµs\r\nGUID:%s\r\n",
		version, took, guid))
	for _, v := range params.additional_resp_headers {
		buffer.WriteString(v)
	}

	if conn_ex.compression_threshold > 0 && len(data) > conn_ex.compression_threshold {

		// standard gzip compression, if we don't have multipart option enabled
		if conn_ex.comp == nil {
			compressed := compress.CompressSimple(data, params.GetAllocator())
			buffer.WriteString("Content-Length: " + strconv.Itoa(len(compressed)) + "\r\n")
			buffer.WriteString("Content-Encoding: gzip / " + strconv.Itoa(len(data)) + "\r\n\r\n")

			blen := buffer.Len()
			dlen := len(compressed)
			if (blen + dlen + 10) < buffer.Cap() {
				buffer.Write(compressed)
				conn.Write(buffer.Bytes())
			} else {
				conn.Write(buffer.Bytes())
				conn.Write(compressed)
			}
			return dlen
		}

		// multipart compression using framework goes here
		out_buffer := params.Allocate(len(data))
		out_buffer = compressor_flate.Compress(data, out_buffer[0:cap(out_buffer)])
		if out_buffer != nil {
			compressor_id := conn_ex.comp.GetID()
			buffer.WriteString("Content-Length: " + strconv.Itoa(len(out_buffer)) + "\r\n")
			buffer.WriteString("Content-Encoding: " + compressor_id + " / " + strconv.Itoa(len(data)) + "\r\n\r\n")
			conn.Write(buffer.Bytes())
			conn.Write(out_buffer)
			return len(out_buffer)
		}
	}

	buffer.WriteString("Content-Length: " + strconv.Itoa(len(data)) + "\r\n\r\n")

	blen := buffer.Len()
	dlen := len(data)
	if (blen + dlen + 10) < buffer.Cap() {
		buffer.Write(data)
		conn.Write(buffer.Bytes())
	} else {
		conn.Write(buffer.Bytes())
		conn.Write([]byte(data))
	}

	return dlen

}

/*
func main() {

	//http://www.ajanicij.info/content/websocket-tutorial-go
	//http://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.4
	//http://greenbytes.de/tech/webdav/draft-ietf-httpbis-p1-messaging-16.html#rfc.section.3.3

	fmt.Print("Handler socket service / Mini HTTP implementation")

}
*/

/* PHP functions for GO */
func explode(delimiter string, str string) []string {
	return strings.Split(str, delimiter)

}

func implode(glue string, str []string) string {
	return strings.Join(str, glue)
}

func trim(str string) string {
	return strings.Trim(str, "\r\n\t ")
}

func in_array(arr []string, needle string) bool {
	for _, v := range arr {
		if v == needle {
			return true
		}
	}

	return false
}

type TimeSpan struct {
	req_time int64
}

func NewTimeSpan() *TimeSpan {

	ts := TimeSpan{}
	ts.req_time = time.Now().UnixNano()
	return &ts

}

func (ts *TimeSpan) Get() string {

	t := float64((time.Now().UnixNano() - ts.req_time)) / float64(1000000)

	return fmt.Sprintf("%.3fms", t)
}

func (ts *TimeSpan) GetRaw() float64 {

	return float64((time.Now().UnixNano() - ts.req_time)) / float64(1000000)
}

// Here goes support for standard handler_socket functionality!
func handlerServerStatus(data *HSParams) string {

	// let's get plugin status!
	status_additional := ""
	for _, sp := range statusPlugins {
		header, content := sp()
		status_additional += "<div class='container'><h1>" + header + "</h1>" + content + "</div>"
	}

	// byteslabs status
	header, content := byteslabs.GetStatus()
	status_additional += "<div class='container'><h1>" + header + "</h1>" + content + "</div>"
	// <<

	if data.GetParam("plugin_only", "") != "" {
		return status_additional
	}

	path := ""
	if _path, err := os.Readlink("/proc/self/exe"); err == nil {
		path = filepath.Dir(_path) + "/"
	} else {
		fmt.Println("Warning: Can't find exe file path")
	}

	_template, err := ioutil.ReadFile(path + "server-status.html")
	if err != nil && len(path) > 0 {
		_template, err = ioutil.ReadFile("server-status.html")
	}

	if err != nil {
		_wd, _ := os.Getwd()
		out := "Error reading server-status.html"
		out += "<br>Directory: " + "server-status.html"
		if len(path) > 0 {
			out += "<br>Alt-Directory: " + (path + "server-status.html")
		}
		out += "<br>CD: " + _wd
		out += "<br><br>--<br>" + (err.Error())
		return out
	}
	template := string(_template)
	template_vars := GetStatus()

	// additional status
	template_vars["status_additional"] = status_additional
	// <<

	for attr, val := range template_vars {
		template = strings.Replace(template, "##"+attr+"##", val, -1)
	}

	return template
}
