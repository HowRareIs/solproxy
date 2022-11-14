package handler_socket2

import (
	"fmt"
	"net"
	"strings"

	"github.com/slawomir-pryczek/HSServer/handler_socket2/compress"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/config"
)

type conninfo struct {
	conn            *net.TCPConn
	remote_distance byte

	comp                  *compress.Compressor
	compression_threshold int
}

func make_conn_ex(conn *net.TCPConn) conninfo {
	remote_addr := strings.Split(conn.RemoteAddr().String(), ":")[0]
	remote_distance := config.Config().GetIPDistance(remote_addr)

	conn_ex := conninfo{}
	conn_ex.conn = conn
	conn_ex.remote_distance = remote_distance
	if remote_distance > 0 {
		if compressor_flate != nil {
			conn_ex.comp = compressor_flate
		}
		conn_ex.compression_threshold = config.Config().GetI("compression_threshold", config.DEFAULT_COMPRESSION_THRESHOLD)
	}

	if conn_ex.compression_threshold == 0 {
		conn_ex.comp = nil
	}
	return conn_ex
}

func handle_conn_ex(data *HSParams, conn_ex *conninfo) {
	if conn_ex.remote_distance == 0 {
		return
	}

	features := strings.ToLower(data.GetParam("features", ""))
	has_snappy := strings.Contains(features, "snappy")

	// always use no compression for LOCAL(0), snappy or flate for LAN(1) and flate for WAN(2)
	if has_snappy && compressor_snappy != nil {
		conn_ex.comp = compressor_snappy
	}
	if (conn_ex.comp == nil || conn_ex.remote_distance > 1) && compressor_flate != nil {
		conn_ex.comp = compressor_flate
	}

	// update threshold
	_thr_from_cfg := config.Config().GetCompressionThreshold()
	conn_ex.compression_threshold = data.GetParamI("compression_threshold", _thr_from_cfg)
	if conn_ex.compression_threshold == 0 {
		conn_ex.comp = nil
	}

	_algo := "-"
	if conn_ex.comp != nil {
		_algo = conn_ex.comp.GetID()
	}

	fmt.Println("\t Conn-Ex <- ", conn_ex.conn.RemoteAddr(),
		" Network distance:", conn_ex.remote_distance,
		" size >", conn_ex.compression_threshold,
		" = ", _algo)
}
