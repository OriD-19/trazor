// go:build ignore

#include "vmlinux.h"
#include <bpf/bpf_helpers.h>

struct trace_event_raw_sys_enter_connect {
    u64 __unused__;
    u32 _syscall_nr;
    u64 fd;
    struct sockaddr *uservaddr;
    u64 addrlen;
};

SEC("tp/syscalls/sys_enter_connect")
int fernandito(struct trace_event_raw_sys_enter_connect *ctx) {
    return 0;
}

char __license[] SEC("license") = "Dual MIT/GPL";
