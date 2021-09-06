package compress

import (
	"bytes"
	"compress/flate"
	"io"
)

type compressionflate struct {
	stat *stat
}

func MakeFlate() *compressionflate {
	ret := compressionflate{}
	ret.stat = MakeStat("Flate")
	return &ret
}

func (c *compressionflate) compression_thread(c_in chan piece_in, symbol byte) bool {

	block_size := c.block_size()

	buf := make([]byte, 0, block_size+800)
	buffer := bytes.NewBuffer(buf)
	writer, err := flate.NewWriter(buffer, flate.BestSpeed)
	if err != nil {
		return false
	}

	thread_id := c.stat.ThreadAdd(symbol)
	for {
		buffer = bytes.NewBuffer(buf[0:0:cap(buf)])
		writer.Reset(buffer)

		data := <-c_in
		c.stat.ThreadSetRunning(thread_id, true)

		io.Copy(writer, bytes.NewReader(data.data_in))
		writer.Close()

		b := buffer.Bytes()
		if len(b) >= len(data.data_in) && len(b) > block_size/3 {
			data.ch_out <- piece_out{nil, -1}
			c.stat.reportCompressionRatioTooLow()
			continue
		}
		if len(b) > len(data.data_out) {
			data.ch_out <- piece_out{nil, -1}
			c.stat.reportBufferTooSmall()
			continue
		}

		stat_in := len(data.data_in)
		stat_out := len(b)
		copy(data.data_out[0:len(b)], b)
		data.ch_out <- piece_out{data.data_out[0:len(b)], data.piece_num}

		c.stat.doStats(thread_id, data.piece_num, stat_in, stat_out)
	}
	return true
}

func (c *compressionflate) uncompress_simple(in []byte, size int) []byte {

	reader := flate.NewReader(bytes.NewReader(in))
	buf := make([]byte, 0, len(in)*3)
	buffer := bytes.NewBuffer(buf)
	_, err := io.Copy(buffer, reader)
	if err != nil {
		return nil
	}

	return buffer.Bytes()
}

func (c *compressionflate) get_status() string {
	return c.stat.GetStatus()
}

func (c compressionflate) block_size() int {
	return 60000
}

func (c *compressionflate) get_id() string {
	return "mp-flate"
}
