// This example shows how to mix different transport protocols and have nodes
// with tcp and kcp running in the network at the same time.

// Run with default options: go run main.go

// Show usage: go run main.go -h

package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/nknorg/nnet"
	"github.com/nknorg/nnet/log"
)

func create(transport string, port uint16) (*nnet.NNet, error) {
	conf := &nnet.Config{
		Port:                  port,
		Transport:             transport,
		BaseStabilizeInterval: 233 * time.Millisecond,
	}

	nn, err := nnet.NewNNet(nil, conf)
	if err != nil {
		return nil, err
	}

	return nn, nil
}

func main() {
	numNodesPtr := flag.Int("n", 10, "number of nodes")
	flag.Parse()

	if *numNodesPtr < 1 {
		log.Error("Number of nodes must be greater than 0")
		return
	}

	const createPort uint16 = 23333
	transports := []string{"tcp", "kcp"}
	nnets := make([]*nnet.NNet, 0)

	for i := 0; i < *numNodesPtr; i++ {
		nn, err := create(transports[rand.Intn(len(transports))], createPort+uint16(i))
		if err != nil {
			log.Error(err)
			return
		}

		nnets = append(nnets, nn)
	}

	for i := 0; i < len(nnets); i++ {
		time.Sleep(112358 * time.Microsecond)

		err := nnets[i].Start(i == 0)
		if err != nil {
			log.Error(err)
			return
		}

		if i > 0 {
			err = nnets[i].Join(nnets[0].GetLocalNode().Addr)
			if err != nil {
				log.Error(err)
				return
			}
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	log.Info("\nReceived an interrupt, stopping...\n")

	var wg sync.WaitGroup
	for i := 0; i < len(nnets); i++ {
		wg.Add(1)
		go func(nn *nnet.NNet) {
			nn.Stop(nil)
			wg.Done()
		}(nnets[len(nnets)-1-i])
	}
	wg.Wait()
}
