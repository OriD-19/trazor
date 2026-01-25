//go:build ignore
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

struct http_event {
    __u32 __padding__;

    __u64 timestamp;
    __u64 latency_ns;
    __u32 pid;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);
    __type(value, __u64);
    __uint(max_entries, 192 * 1024);
} latency SEC(".maps");

struct {
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 256 * 1024);
} events SEC(".maps");

SEC("uprobe/ngx_event_accept")
int get_conn_start(struct pt_regs *ctx) {

    u64 ts = bpf_ktime_get_ns();
    u64 pid = bpf_get_current_pid_tgid() >> 32; // left-shift for process id only

    bpf_map_update_elem(&latency, &pid, &ts, BPF_ANY); 

    return 0;
}

SEC("uprobe/ngx_http_finalize_connection")
int get_latency_on_end(struct pt_regs *ctx) {
    struct http_event *req_info;
    u64 pid = bpf_get_current_pid_tgid() >> 32;
    u64 ts = bpf_ktime_get_ns();

    // last value is always 0, for some reason...
    req_info = bpf_ringbuf_reserve(&events, sizeof(*req_info), 0);
    if (!req_info) // no valid memory allocated, returned NULL
        return 0;

    req_info->timestamp = ts;

    // get start time of this request 
    u64 *init = bpf_map_lookup_elem(&latency, &pid);

    u64 delta = ts - *init;

    req_info->latency_ns = delta;
    req_info->pid = pid;

    return 0;
}
