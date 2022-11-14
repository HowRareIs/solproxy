package handler_socket2

import (
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/compress"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
	"runtime"
	"strings"
)

var compressor_snappy *compress.Compressor = nil
var compressor_flate *compress.Compressor = nil

func compression_ex_read_config() {
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
}

func compression_ex_status() string {
	if compressor_flate == nil && compressor_snappy == nil {
		return ""
	}

	ret := ""
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
	return ret
}
