package byteslabs2

import (
	"fmt"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
	"strconv"
	"strings"
)

func (this *BSManager) GetStatus() (string, string) {

	ret := "<pre>"
	ret += fmt.Sprintf("Slab Allocator. Slab Size: %d x %d Slabs = %d items per Page. %d Pages.\n", this.conf_slab_size, this.conf_slab_count, this.conf_slab_size*this.conf_slab_count, len(this.mem_chunks))
	ret += "Alloc Full - We're allocating >1 SLABS, happens at page begin\n"
	ret += "Alloc Full Small - We're allocating 1 SLAB, happens at page's end to prevent fragmentation\n"
	ret += "Tail - We can put the data into already allocated SLAB's tail\n"
	ret += "OOM - Page is out of memory, we're using system allocator instead\n"
	ret += "Routed - Allocations was routed to least used slab page, allocations per routing\n"
	ret += "Routed Alloc - Allocations in routed slabs\n"
	ret += ". - Slab is EMPTY; F - Slab is FULL\n"
	ret += "</pre>"

	ret += "<style>"
	ret += ".salloc td:nth-child(5) span {color: #771111}\n"
	ret += ".salloc td:nth-child(7) span {color: #666633}\n"
	ret += ".salloc thead td:nth-child(5) {background:#ffaaaa}\n"
	ret += ".salloc thead td:nth-child(7) {background:#dddd88}\n"
	ret += "</style>"

	tg := hscommon.NewTableGen("#", "Alloc Full", "Full Small", "Tail",
		"<span>OOM</span>", "Slabs", "<span>Routed</span>", "Routed Alloc")
	tg.SetClass("tab salloc")

	for k, chunk := range this.mem_chunks {
		chunk.mu.Lock()

		_slabs := ""
		for _, used := range chunk.slab_used {
			if used {
				_slabs += "F"
			} else {
				_slabs += "."
			}
		}

		percent_oom := int(chunk.stat_alloc_full + chunk.stat_alloc_full_small + chunk.stat_alloc_tail)
		if percent_oom > 0 {
			percent_oom = int((float64(chunk.stat_oom) / float64(percent_oom)) * 1000)
		}
		oom := fmt.Sprintf("<span>%d</span> (%d.%d%%)", chunk.stat_oom, percent_oom/10, percent_oom%10)

		apr := 0
		if chunk.stat_routed > 0 {
			apr = chunk.stat_routed_alloc * 10 / chunk.stat_routed
		}
		routed := fmt.Sprintf("<span>%d</span> (%d.%d apr)", chunk.stat_routed, apr/10, apr%10)

		tg.AddRow(strconv.Itoa(k), strconv.Itoa(chunk.stat_alloc_full), strconv.Itoa(chunk.stat_alloc_full_small),
			strconv.Itoa(chunk.stat_alloc_tail), oom, "<pre>"+_slabs+"</pre>", routed, strconv.Itoa(chunk.stat_routed_alloc))

		chunk.mu.Unlock()
	}

	return "Slab Allocator \\ QCompress", ret + tg.Render()
}

func (this *BSManager) GetStatusStr() string {
	ret := make([]string, 0, 40)
	ret = append(ret, "=====")
	total_failed := 0
	total_f, total_fs, total_t := 0, 0, 0
	for k, v := range this.mem_chunks {
		ret = append(ret, fmt.Sprint(k, "Full:", v.stat_alloc_full, "Full Small:", v.stat_alloc_full_small,
			"Tail:", v.stat_alloc_tail, "OOM:", v.stat_oom, "Routed:", v.stat_routed,
			"Slab taken:", v.used_slab_count))

		total_failed += v.stat_oom - v.stat_routed
		total_f += v.stat_alloc_full
		total_fs += v.stat_alloc_full_small
		total_t += v.stat_alloc_tail
	}

	ret = append(ret, fmt.Sprintf("=Totals ... Full: %d Full Small: %d Tail: %d \n", total_f, total_fs, total_t))
	ret = append(ret, fmt.Sprintf("=Items Total: %d, Failed: %d\n",
		total_f+total_fs+total_t+total_failed, total_failed))
	ret = append(ret, fmt.Sprint("Real Total: ", ttT_total))
	return strings.Join(ret, "\n")
}
