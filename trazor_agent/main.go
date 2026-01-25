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
	executable, err := link.OpenExecutable("/usr/sbin/nginx")
	if err != nil {
		log.Fatalf("opening executable: %v", err)
	}

	conn_start, err := executable.Uprobe("ngx_event_accept", objs.GetConnStart, nil)
	if err != nil {
		log.Fatalf("opening uprobe 'ngx_event_accept': %v", err)
	}
	defer conn_start.Close()

	conn_end, err := executable.Uprobe("ngx_http_finalize_request", objs.GetLatencyOnEnd, nil)
	if err != nil {
		log.Fatalf("opening uprobe 'ngx_http_finalize_connection': %v", err)
	}
	defer conn_end.Close()
	

	ringBuf, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		log.Fatal("Opening ringbuf reader: ", err)	
		os.Exit(1);
	}

	go func() {
		defer ringBuf.Close()
		for {
			record, err := ringBuf.Read()

			if err != nil {
				log.Fatal("Reading ringbuf: ", err)
			}

			fmt.Println("Event: ", string(record.RawSample))
		}
	}()

	for {

	}
}
