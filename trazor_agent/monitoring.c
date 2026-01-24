// go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

struct connection_info {
    int port;
};

struct {
    __uint(type, BPF_MAP_TYPE_QUEUE);
    __type(value, struct connection_info);
    __uint(max_entries, 500);
} sock_info SEC(".maps");

struct trace_event_raw_sys_enter_connect {
    u64 __unused__;
    u32 _syscall_nr;
    u64 fd;
    struct sockaddr *uservaddr;
    u64 addrlen;
};

SEC("tp/syscalls/sys_enter_connect")
int collect_enter_traces(struct trace_event_raw_sys_enter_connect *ctx) {
    
    struct sockaddr_in addr;

    if (ctx->uservaddr != NULL) {
        bpf_probe_read_user(&addr, sizeof(addr), ctx->uservaddr); // addr recieves the data
    }

    return 0;
}

char __license[] SEC("license") = "Dual MIT/GPL";
