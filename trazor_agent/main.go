package main
//go:generate go tool bpf2go -tags linux trazor_agent monitoring.c
import (
	"fmt"
	"log"
	"os"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

func main() {
	// boilerplate code
	if err := rlimit.RemoveMemlock(); err != nil {
		log.Fatal("Removing Memlock: ", err)
	}

	var objs trazor_agentObjects
	if err := loadTrazor_agentObjects(&objs, nil); err != nil {
		log.Fatal("Loading eBPF objects: ", err)
	}
	defer objs.Close()

	// attach the programs to their respective uprobes
	link.Kprobe("get_conn_start", objs.GetConnStart, nil)
	link.Kprobe("get_latency_on_end", objs.GetLatencyOnEnd, nil)

	ringBuf, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		log.Fatal("Opening ringbuf reader: ", err)	
		os.Exit(1);
	}

	go func() {
		for {
			record, err := ringBuf.Read()

			if err != nil {
				log.Fatal("Reading ringbuf: ", err)
			}

			fmt.Println("Event: ", string(record.RawSample))

			if err := ringBuf.Close(); err != nil {
				log.Fatal("Closing ringbuf reader: ", err)
			}
		}
	}()

	for {

	}
}
