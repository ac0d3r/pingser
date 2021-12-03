package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"pingser"
	"syscall"
	"time"
)

func main() {
	client, err := pingser.NewClient("xxx.xxx.xxx.xxx")
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Listen(); err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	go client.Run()

	client.OnRecv = func(pkt *pingser.Packet) {
		fmt.Printf("ID: %d  Seq: %d  data: %s\n", pkt.ID, pkt.Seq, string(pkt.Data))
	}

	for i := 0; i < 10; i++ {
		client.Send([]byte("hello"))
		time.Sleep(time.Second)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
