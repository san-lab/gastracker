package main

import (
	"flag"
	"os"
	"os/signal"
	"log"
	"github.com/kouhin/envflag"
)

var (
	CURRENCIES = []string{"USD", "EUR"}
)

var (
	rpcEndpoint = flag.String("ethereum-rpc", "http://localhost:8545", "the URL of the RPC server")
	influxEndpoint = flag.String("influxdb", "http://localhost:8086", "the URL of the RPC server")
)

func main() {
	envflag.Parse()
	kill := make(chan os.Signal, 1)
	signal.Notify(kill, os.Interrupt)
	//start
	gasTracker, err := StartGasTracker(*rpcEndpoint)
	if err != nil {
		log.Fatalf("Could not start GasTracker: %s\n", err)
	}
	defer gasTracker.Stop()
	//stop
	<-kill
	log.Println("Closing down...")
}
