// Package gonetmap is a wrapper around the netmap library.
// +build freebsd

package gonetmap

/*
#include <stdio.h>
#define NETMAP_WITH_LIBS
//#cgo CFLAGS: -DNETMAP_WITH_LIBS -g3
#include <net/netmap_user.h>

int nm_stats(struct nm_desc *d, struct nm_stat *stat) {
    stat->ps_recv = d->st.ps_recv;
    stat->ps_drop = d->st.ps_drop;
    stat->ps_ifdrop = d->st.ps_ifdrop;

    return 0;
}

int nm_getfd(struct nm_desc *d) {
    return NETMAP_FD(d);
}

static struct nm_desc *open_netmap(const char *ifname) {
    printf("%s\n", ifname);
    return nm_open(ifname, NULL, 0, NULL);
}

*/
import "C"

import (
	"errors"
	"time"
	"unsafe"
)

const (
	NM_OPEN_NO_MMAP  = 0x040000 /* reuse mmap from parent */
	NM_OPEN_IFNAME   = 0x080000 /* nr_name, nr_ringid, nr_flags */
	NM_OPEN_ARG1     = 0x100000
	NM_OPEN_ARG2     = 0x200000
	NM_OPEN_ARG3     = 0x400000
	NM_OPEN_RING_CFG = 0x800000 /* tx|rx rings|slots */
)

var (
	OPEN_FAILED   = errors.New("open netmap failed")
	BUFF_IS_NULL  = errors.New("buffer is nil")
	INJECT_FAILED = errors.New("netmap inject failed")
)

type Netmap struct {
	cptr *C.struct_nm_desc
	Fd   int
}

type Packet struct {
	Time   time.Time // packet time
	Caplen uint32    // bytes stored in the file (caplen <= len)
	Len    uint32    // bytes sent/received
	Data   []byte    // raw packet data
}

type Stat struct {
	Received  uint32
	Dropped   uint32
	IfDropped uint32
}

func OpenNetmap(device string) (handle *Netmap, err error) {
	dev := C.CString(device)
	defer C.free(unsafe.Pointer(dev))

	h := new(Netmap)

	h.cptr = C.nm_open(dev, nil, 0, nil)
	if h.cptr == nil {
		return nil, OPEN_FAILED
	}

	h.Fd = int(C.nm_getfd(h.cptr))
	handle = h

	return

}

func (p *Netmap) SetFilter(expr string) (err error) {
	return
}

func (p *Netmap) Next() (pkt *Packet) {
	rv, _ := p.NextPkt()
	return rv
}

func (p *Netmap) NextPkt() (pkt *Packet, err error) {
	var pkthdr_ptr C.struct_nm_pkthdr

	var buf_ptr *C.u_char
	var buf unsafe.Pointer
	buf_ptr = C.nm_nextpkt(p.cptr, &pkthdr_ptr)

	buf = unsafe.Pointer(buf_ptr)

	if nil == buf {
		return nil, BUFF_IS_NULL
	}

	//defer C.free(unsafe.Pointer(buf))

	pkt = new(Packet)
	pkt.Time = time.Unix(int64(pkthdr_ptr.ts.tv_sec), int64(pkthdr_ptr.ts.tv_usec)*1000)
	pkt.Caplen = uint32(pkthdr_ptr.caplen)
	pkt.Len = uint32(pkthdr_ptr.len)

	pkt.Data = C.GoBytes(unsafe.Pointer(buf), C.int(pkthdr_ptr.caplen))

	return
}

func (p *Netmap) Getstats() (stat *Stat, err error) {
	var cstats _Ctype_struct_nm_stat
	C.nm_stats(p.cptr, &cstats)

	stats := new(Stat)
	stats.Received = uint32(cstats.ps_recv)
	stats.Dropped = uint32(cstats.ps_drop)
	stats.IfDropped = uint32(cstats.ps_ifdrop)

	return stats, nil
}

func (p *Netmap) Inject(data []byte) (err error) {

	buf := C.CString(string(data))
	defer C.free(unsafe.Pointer(buf))

	if -1 == C.nm_inject(p.cptr, unsafe.Pointer(buf), (C.size_t)(len(data))) {
		err = INJECT_FAILED
	}

	return
}

func (p *Netmap) Close() {
	C.nm_close(p.cptr)
}
