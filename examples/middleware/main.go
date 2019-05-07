// This example shows how to interact with nnet by middleware. This example only
// shows a few middleware, please see the document for a complete list of
// middleware.

// Run with default options: go run main.go

// Show usage: go run main.go -h

package main

import (
	"errors"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/nknorg/nnet"
	"github.com/nknorg/nnet/log"
	"github.com/nknorg/nnet/node"
	"github.com/nknorg/nnet/overlay"
	"github.com/nknorg/nnet/overlay/chord"
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
	transportPtr := flag.String("t", "tcp", "transport type, tcp or kcp")
	numNodesPtr := flag.Int("n", 10, "number of nodes")
	flag.Parse()

	if *numNodesPtr < 1 {
		log.Error("Number of nodes must be greater than 0")
		return
	}

	const createPort uint16 = 23333
	nnets := make([]*nnet.NNet, 0)

	nn, err := create(*transportPtr, createPort)
	if err != nil {
		log.Error(err)
		return
	}

	nn.MustApplyMiddleware(overlay.NetworkWillStart{func(network overlay.Network) bool {
		log.Infof("Network will start")
		return true
	}, 0})

	nn.MustApplyMiddleware(overlay.NetworkStarted{func(network overlay.Network) bool {
		log.Infof("Network started")
		return true
	}, 0})

	nn.MustApplyMiddleware(overlay.NetworkWillStop{func(network overlay.Network) bool {
		log.Infof("Network will stop")
		return true
	}, 0})

	nn.MustApplyMiddleware(overlay.NetworkStopped{func(network overlay.Network) bool {
		log.Infof("Network stopped")
		return true
	}, 0})

	nn.MustApplyMiddleware(node.RemoteNodeConnected{func(remoteNode *node.RemoteNode) bool {
		log.Infof("Remote node connected: %v", remoteNode)
		return true
	}, 0})

	nn.MustApplyMiddleware(node.RemoteNodeReady{func(remoteNode *node.RemoteNode) bool {
		log.Infof("Remote node ready (normal priority): %v", remoteNode)
		return true
	}, 0})

	nn.MustApplyMiddleware(node.RemoteNodeReady{func(remoteNode *node.RemoteNode) bool {
		log.Infof("Remote node ready (higher priority): %v", remoteNode)
		return true
	}, 1})

	nn.MustApplyMiddleware(node.RemoteNodeReady{func(remoteNode *node.RemoteNode) bool {
		log.Infof("Remote node ready (lower priority): %v", remoteNode)
		return true
	}, -1})

	nn.MustApplyMiddleware(node.RemoteNodeDisconnected{func(remoteNode *node.RemoteNode) bool {
		log.Infof("Remote node disconnected: %v", remoteNode)
		return true
	}, 0})

	nn.MustApplyMiddleware(chord.NeighborAdded{func(remoteNode *node.RemoteNode, index int) bool {
		log.Infof("New neighbor %d: %v", index, remoteNode)
		return true
	}, 0})

	nn.MustApplyMiddleware(chord.SuccessorAdded{func(remoteNode *node.RemoteNode, index int) bool {
		log.Infof("New successor %d: %v", index, remoteNode)
		return true
	}, 0})

	nn.MustApplyMiddleware(chord.PredecessorAdded{func(remoteNode *node.RemoteNode, index int) bool {
		log.Infof("New predecessor %d: %v", index, remoteNode)
		return true
	}, 0})

	nn.MustApplyMiddleware(chord.FingerTableAdded{func(remoteNode *node.RemoteNode, fingerIndex, nodeIndex int) bool {
		log.Infof("New finger table %d-%d: %v", fingerIndex, nodeIndex, remoteNode)
		return true
	}, 0})

	err = nn.Start(true)
	if err != nil {
		log.Error(err)
		return
	}

	nnets = append(nnets, nn)

	for i := 0; i < *numNodesPtr-1; i++ {
		time.Sleep(112358 * time.Microsecond)

		nn, err := create(*transportPtr, createPort+uint16(i)+1)
		if err != nil {
			log.Error(err)
			return
		}

		nn.MustApplyMiddleware(node.RemoteNodeConnected{func(remoteNode *node.RemoteNode) bool {
			if rand.Float64() < 0.23333 {
				remoteNode.Stop(errors.New("YOU ARE UNLUCKY"))
				// stop propagate to the next middleware
				return false
			}
			return true
		}, 0})

		nn.MustApplyMiddleware(node.RemoteNodeConnected{func(remoteNode *node.RemoteNode) bool {
			log.Infof("Only lucky remote node can get here :)")
			return true
		}, 0})

		err = nn.Start(false)
		if err != nil {
			log.Error(err)
			return
		}

		err = nn.Join(nnets[0].GetLocalNode().Addr)
		if err != nil {
			log.Error(err)
			return
		}

		nnets = append(nnets, nn)
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
