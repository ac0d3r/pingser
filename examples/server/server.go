package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"pingser"
	"syscall"
)

func main() {
	server := pingser.NewServer()
	if err := server.Listen(); err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	go server.Run()

	server.OnRecv = func(pkt *pingser.Packet) {
		fmt.Printf("ID: %d  Seq: %d  data: %s\n", pkt.ID, pkt.Seq, string(pkt.Data))
		pkt.Data = []byte("world")
		if err := server.SendWitRecv(pkt); err != nil {
			fmt.Printf("SendWitRecv - error: %s", err)
		}
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
