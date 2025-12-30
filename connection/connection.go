package connection

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	activeStreams = make(map[peer.ID]network.Stream)
	streamsMux    sync.Mutex
)

func Connectp2p(con host.Host, addrStr string) {
	fmt.Printf("try to connect: %s \n", addrStr)
	addr, err := peer.AddrInfoFromString(addrStr)
	if err != nil {
		fmt.Printf("wrong address %v \n", err)
		return
	}
	err = con.Connect(context.Background(), *addr)
	if err != nil {
		fmt.Printf("connection not established %v \n", err)
		return
	}
	fmt.Printf("success %s! \n", addr.ID.ShortString())

	stream, err := con.NewStream(context.Background(), addr.ID, "/chat/1.0")
	if err != nil {
		fmt.Printf("stream unsuccess. PANIC %v \n", err)
		return
	}

	streamsMux.Lock()
	activeStreams[addr.ID] = stream
	streamsMux.Unlock()

	// goroutine accept
	go func() {
		reader := bufio.NewReader(stream)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				streamsMux.Lock()
				delete(activeStreams, addr.ID)
				streamsMux.Unlock()
				fmt.Printf("\n[%s] disconnected\n* ", addr.ID.ShortString())
				return
			}
			fmt.Printf("\n[%s]: %s\n* ", addr.ID.ShortString(), strings.TrimSpace(msg))
		}
	}()
}

func SetupMessageHandler(con host.Host) {
	con.SetStreamHandler("/chat/1.0", func(stream network.Stream) {
		remotePeer := stream.Conn().RemotePeer()
		fmt.Printf("\n new connection from %s! \n* ", remotePeer.ShortString())

		streamsMux.Lock()
		activeStreams[remotePeer] = stream
		streamsMux.Unlock()

		reader := bufio.NewReader(stream)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				streamsMux.Lock()
				delete(activeStreams, remotePeer)
				streamsMux.Unlock()
				fmt.Printf("\n[%s] connection terminated\n* ", remotePeer.ShortString())
				stream.Close()
				return
			}
			fmt.Printf("\n[%s]: %s\n* ", remotePeer.ShortString(), strings.TrimSpace(msg))
		}
	})
	fmt.Println("debug: handler success")
}

func SendToActivePeer(peerID peer.ID, message string) {
	streamsMux.Lock()
	stream, exists := activeStreams[peerID]
	streamsMux.Unlock()

	if !exists {
		fmt.Printf("no active connection to %s \n", peerID.ShortString())
		return
	}

	writer := bufio.NewWriter(stream)
	_, err := writer.WriteString(message + "\n")
	if err != nil {
		fmt.Printf("failed to send message: %v \n", err)
		return
	}
	writer.Flush()
}

func BroadcastMessage(message string) {
	streamsMux.Lock()
	defer streamsMux.Unlock()

	for peerID, stream := range activeStreams {
		writer := bufio.NewWriter(stream)
		_, err := writer.WriteString(message + "\n")
		if err != nil {
			fmt.Printf("failed to send to %s: %v \n", peerID.ShortString(), err)
			continue
		}
		writer.Flush()
	}
}

func ListActivePeers() {
	streamsMux.Lock()
	defer streamsMux.Unlock()

	if len(activeStreams) == 0 {
		fmt.Println("no active connections")
		return
	}

	fmt.Println("active connections:")
	for peerID := range activeStreams {
		fmt.Printf("  - %s \n", peerID.ShortString())
	}
}

func HasActiveStreams() bool {
	streamsMux.Lock()
	defer streamsMux.Unlock()
	return len(activeStreams) > 0
}
