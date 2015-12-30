package main

import (
	"flag"
	"github.com/jonsen/gonetmap"
	"github.com/jonsen/kqueue"
	"log"
	"os"
	"syscall"
	"time"
)

var (
	iface = flag.String("i", "", "interface")
)

func main() {
	flag.Parse()

	if *iface == "" {
		log.Println("usage netmaptest -i netmap:em1")
		os.Exit(1)
	}

	np, err := gonetmap.OpenNetmap(*iface)
	if err != nil {
		log.Println(err)
		return
	}

	defer np.Close()

	time.Sleep(time.Second * 4)

	kq, err := kqueue.NewKqueue()
	if err != nil {
		log.Println(err)
		return
	}

	kq.Add(uintptr(np.Fd), syscall.EVFILT_READ, syscall.EV_CLEAR, 0)

	// wait for events
	events := make([]syscall.Kevent_t, 10)
	for {
		// create kevent
		_, err := kq.Wait(events)
		if err != nil && err != syscall.EINTR {
			log.Println("Error creating kevent")
			continue
		}

		pkt, err := np.NextPkt()
		if err != nil {
			log.Println(err)
			continue
		}

		if pkt != nil {
			log.Println("got pkt..", pkt.Caplen)
			pkt = nil
		} else {
			log.Println("pkt is nil")
		}

	}

	st, err := np.Getstats()
	if err != nil {
		log.Println(err)
	}

	log.Println(st)

}
