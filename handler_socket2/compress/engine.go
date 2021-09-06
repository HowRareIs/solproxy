package compress

import (
	"encoding/binary"
	"fmt"
)

type Compressor struct {
	compressor      chan piece_in
	compressor_fast chan piece_in
	c               cpr
}

type piece_in struct {
	data_in   []byte
	data_out  []byte
	piece_num int

	ch_out chan<- piece_out
}

type piece_out struct {
	data_out []byte
	piece    int
}

type cpr interface {
	compression_thread(c_in chan piece_in, symbol byte) bool
	block_size() int
	uncompress_simple(in []byte, size int) []byte
	get_status() string
	get_id() string
}

func CreateCompressor(threads int, c cpr) *Compressor {

	ret := Compressor{}
	ret.compressor = make(chan piece_in, 500)
	ret.compressor_fast = make(chan piece_in, 50)
	ret.c = c

	// create normal threads for doing multipart compression
	for i := 0; i < threads; i++ {
		go c.compression_thread(ret.compressor, 'N')
	}

	// create fastpath threads
	threads = threads / 2
	if threads < 6 {
		threads = 6
	}
	for i := 0; i < threads; i++ {
		go c.compression_thread(ret.compressor_fast, 'F')
	}

	return &ret
}

// this function will run fast compression
func (this *Compressor) _fastpath(in []byte, out []byte) []byte {
	offset := 8

	// compress data
	ch_out := make(chan piece_out, 1)
	this.compressor_fast <- piece_in{in, out[offset:], 0, ch_out}

	// get response
	ret := <-ch_out
	if ret.data_out == nil || ret.piece < 0 || float32(len(ret.data_out)) > float32(len(in))*0.8 {
		return nil
	}

	out = out[0 : len(ret.data_out)+offset]
	binary.LittleEndian.PutUint32(out[0:4], uint32(len(ret.data_out)))
	binary.LittleEndian.PutUint32(out[4:8], 0)
	return out
}

func (this *Compressor) Compress(in []byte, out []byte) []byte {

	block_size := this.c.block_size()
	var in_progress = 0
	var in_len = len(in)

	if in_len < block_size {
		return this._fastpath(in, out)
	}

	is_broken := false
	out_size_total := 0
	tasks_no := (in_len / block_size)
	if in_len > tasks_no*block_size {
		tasks_no++
	}

	// function to get data back from workers
	chunks := make([]int, tasks_no+1)
	ch_out := make(chan piece_out, 8)
	get_next_piece := func() {
		ret := <-ch_out
		in_progress--

		out_size_total += len(ret.data_out)
		is_broken = is_broken || ret.piece < 0
		if is_broken {
			return
		}
		chunks[ret.piece] = len(ret.data_out)
	}

	piece_no := 0
	for i := 0; i < in_len; i += block_size {
		end := i + block_size
		if end > in_len {
			end = in_len
		}

		this.compressor <- piece_in{in[i:end], out[i:end], piece_no, ch_out}
		piece_no++
		in_progress++

		if in_progress >= 12 {
			get_next_piece()
			if is_broken {
				break
			}
		}
	}

	for in_progress > 0 {
		get_next_piece()
	}

	if is_broken || float32(out_size_total) > float32(len(in))*0.8 {
		return nil
	}

	// merge the blocks together... skip space needed for allocation table first
	dst_pos := 4 * len(chunks)
	for i := 0; i < tasks_no; i++ {

		chunk_size := chunks[i]
		src_start := block_size * i
		copy(out[dst_pos:dst_pos+chunk_size], out[src_start:src_start+chunk_size])

		dst_pos += chunk_size
	}

	for i := 0; i < len(chunks); i++ {
		pos := i * 4
		binary.LittleEndian.PutUint32(out[pos:pos+4], uint32(chunks[i]))
	}

	//fmt.Println("Yx", len(in), ">", dst_pos, "|", chunks, out[0:28], out[dst_pos-16:dst_pos])
	//fmt.Println(dst_pos, out[0:109])
	return out[0:dst_pos]
}

func (this *Compressor) Uncompress(in []byte) []byte {

	chunks := make([]uint32, 0, 10)
	pos := 0
	for pos < len(in)-4 {
		size := binary.LittleEndian.Uint32(in[pos : pos+4])
		pos += 4

		if size == 0 {
			break
		}
		chunks = append(chunks, size)
	}
	if len(chunks) == 0 {
		return nil
	}

	out := make([]byte, 0, len(in)*2)
	chunk := 0
	for pos < len(in) && chunk < len(chunks) {

		chunk_size := int(chunks[chunk])
		chunk_content := this.c.uncompress_simple(in[pos:pos+chunk_size], chunk_size)

		fmt.Println("UNC c", chunk, len(in[pos:pos+chunk_size]), in[pos:pos+10])

		chunk++
		pos += chunk_size
		out = append(out, chunk_content...)
	}

	return out
}

func (this *Compressor) GetStatus() string {
	return this.c.get_status()
}

func (this *Compressor) GetID() string {
	return this.c.get_id()
}
