package compress

import (
	"github.com/slawomir-pryczek/HSServer/handler_socket2/compress/snappy"
)

type compressionsnappy struct {
	stat *stat
}

func MakeSnappy() *compressionsnappy {
	ret := compressionsnappy{}
	ret.stat = MakeStat("Snappy")
	return &ret
}

func (c *compressionsnappy) compression_thread(c_in chan piece_in, symbol byte) bool {

	block_size := c.block_size()

	thread_id := c.stat.ThreadAdd(symbol)
	buf := make([]byte, 0, block_size+800)
	for {
		data := <-c_in
		c.stat.ThreadSetRunning(thread_id, true)
		b := snappy.Encode(buf, data.data_in)

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

func (c *compressionsnappy) uncompress_simple(in []byte, size int) []byte {

	d_len, err := snappy.DecodedLen(in)
	if err != nil {
		return nil
	}
	buf := make([]byte, 0, d_len)

	ret, err1 := snappy.Decode(buf, in)
	if err1 != nil {
		return nil
	}
	return ret
}

func (c *compressionsnappy) get_status() string {
	return c.stat.GetStatus()
}

func (c compressionsnappy) block_size() int {
	return 120000
}

func (c *compressionsnappy) get_id() string {
	return "mp-snappy"
}
