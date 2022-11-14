package handle_profiler

import (
	"fmt"
	"runtime"

	"github.com/slawomir-pryczek/HSServer/handler_socket2"
	"github.com/slawomir-pryczek/HSServer/handler_socket2/hscommon"
)

type HandleProfiler struct {
}

func (this *HandleProfiler) Initialize() {

	handler_socket2.StatusPluginRegister(func() (string, string) {

		tmp := this.HandleAction("profiler", &handler_socket2.HSParams{})
		return "Memory profile", tmp

	})

}

func (this *HandleProfiler) Info() string {
	return "This plugin will display basic memory profiling information"
}

func (this *HandleProfiler) GetActions() []string {
	return []string{"profiler"}
}

func (this *HandleProfiler) HandleAction(action string, data *handler_socket2.HSParams) string {

	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	if data.GetParam("simple", "") == "1" {

		ret := ""
		ret += "memStats.TotalAlloc: " + hscommon.FormatBytes(memStats.TotalAlloc) + "\n"
		ret += "memStats.Mallocs: " + fmt.Sprintf("%d", memStats.Mallocs) + "\n"
		ret += "memStats.NumGC: " + fmt.Sprintf("%d", memStats.NumGC) + "\n"
		return ret

	}

	tg := hscommon.NewTableGen("Parameter", "Value", "Help")
	tg.SetClass("tab")

	tg.AddRow("General statistics ==== ")
	tg.AddRow("memStats.Alloc", hscommon.FormatBytes(memStats.Alloc), "Bytes allocated and still in use")
	tg.AddRow("memStats.TotalAlloc", hscommon.FormatBytes(memStats.TotalAlloc), "Bytes allocated in total (even if freed)")
	tg.AddRow("memStats.Sys", hscommon.FormatBytes(memStats.Sys), "Bytes obtained from system")
	tg.AddRow("memStats.Lookups", fmt.Sprintf("%d", memStats.Lookups), "Number of pointer lookups")
	tg.AddRow("memStats.Mallocs", fmt.Sprintf("%d", memStats.Mallocs), "Number of mallocs")
	tg.AddRow("memStats.Frees", fmt.Sprintf("%d", memStats.Frees), "Number of frees")

	tg.AddRow("Heap statistics ==== ")
	tg.AddRow("memStats.HeapAlloc", hscommon.FormatBytes(memStats.HeapAlloc), "Bytes allocated and still in use")
	tg.AddRow("memStats.HeapSys", hscommon.FormatBytes(memStats.HeapSys), "Bytes obtained from system")
	tg.AddRow("memStats.HeapIdle", hscommon.FormatBytes(memStats.HeapIdle), "Bytes in idle spans")
	tg.AddRow("memStats.HeapInuse", hscommon.FormatBytes(memStats.HeapInuse), "Bytes in non-idle span")
	tg.AddRow("memStats.HeapReleased", hscommon.FormatBytes(memStats.HeapReleased), "Bytes released to the OS")
	tg.AddRow("memStats.HeapObjects", fmt.Sprintf("%d", memStats.HeapObjects), "Total number of allocated objects")

	tg.AddRow("Stack & System allocation ==== ")
	tg.AddRow("memStats.StackInuse", hscommon.FormatBytes(memStats.StackInuse), "Bytes used by stack allocator")
	tg.AddRow("memStats.StackSys", hscommon.FormatBytes(memStats.StackSys), "Bytes obtained from OS by stack allocator")
	tg.AddRow("memStats.MSpanInuse", fmt.Sprintf("%d", memStats.MSpanInuse), "MSpan structures in use")
	tg.AddRow("memStats.MSpanSys", fmt.Sprintf("%d", memStats.MSpanSys), "MSpan structures obtained from OS")
	tg.AddRow("memStats.MCacheInuse", fmt.Sprintf("%d", memStats.MCacheInuse), "MCache structures in use")
	tg.AddRow("memStats.MCacheSys", fmt.Sprintf("%d", memStats.MCacheSys), "MCache structures obtained from OS")
	tg.AddRow("memStats.BuckHashSys", fmt.Sprintf("%d", memStats.BuckHashSys), "Profiling bucket hash table")
	tg.AddRow("memStats.OtherSys", fmt.Sprintf("%d", memStats.OtherSys), "Other system allocations")

	tg.AddRow("Garbage Collection ==== ")
	tg.AddRow("memStats.NumGC", fmt.Sprintf("%d", memStats.NumGC), "Number of garbage colledtions")
	tg.AddRow("memStats.PauseTotalNs", fmt.Sprintf("%d", memStats.PauseTotalNs), "Total time waited in GC")
	tg.AddRow("memStats.NextGC", fmt.Sprintf("%d", memStats.NextGC), "Next collection will happen when HeapAlloc ≥ this amount")
	tg.AddRow("memStats.LastGC", fmt.Sprintf("%d", memStats.LastGC), "End time of last collection (nanoseconds since 1970)")

	tg.AddRow("Server statistics ==== ")
	tg.AddRow("runtime.NumGoroutine", fmt.Sprintf("%d", runtime.NumGoroutine()), "Number of goroutines that currently exist")
	//tg.AddRow("runtime.InUseBytes", fmt.Sprintf("%d", runtime.InUseBytes()), "Number of bytes in use")
	//tg.AddRow("runtime.InUseObjects", fmt.Sprintf("%d", runtime.InUseObjects()), "Number of objects in use")

	return tg.Render()
}
