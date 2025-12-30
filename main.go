package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	//need to test
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

var (
	activeStreams = make(map[peer.ID]network.Stream)
	streamsMux    sync.Mutex
)

func main() {
	con, err := libp2p.New(
		libp2p.EnableRelay(), // relay
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{
			megaparse("/ip4/147.75.80.110/tcp/4001/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb"), //add more + config add wip
		}),
		libp2p.EnableHolePunching(), //nat punching
		libp2p.NATPortMap(),

		//todo
	)
	if err != nil {
		panic(err)
	}
	defer con.Close()

	anoctx := context.Background()
	kdht, err := dht.New(anoctx, con, dht.Mode(dht.ModeServer))
	if err != nil {
		panic(err)
	}

	if err = kdht.Bootstrap(anoctx); err != nil {
		panic(err)
	}

	bootstrapPeers := []string{ //i stealed this
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}

	for _, addr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromString(addr)
		if err := con.Connect(anoctx, *peerinfo); err != nil {
			fmt.Printf("failed to connect to bootstrap %v \n", err)
		}
	}

	setupMessageHandler(con) //i stealed this

	fmt.Println("ID: ", con.ID())

	fmt.Println("\n adresses:")
	for i, addr := range con.Addrs() {
		fadr := fmt.Sprintf("%s/p2p/%s", addr, con.ID())
		fmt.Printf("(%d) %s\n", i+1, fadr)
	}

	comandOP(con)
}

func comandOP(con host.Host) {
	runscanner := bufio.NewScanner(os.Stdin)
	fmt.Print("*")
	for runscanner.Scan() {
		cmd := runscanner.Text()
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			fmt.Print("> ")
			continue
		}
		switch parts[0] {
		case "/info":
			showMyInfo(con)
		case "/martlets":
			fmt.Println("MARTLETS ARE PLENTIFUL!")
		case "/connect":
			if len(parts) < 2 {
				fmt.Println("err")
			} else {
				connect(con, parts[1])
			}
		case "/peers":
			listActivePeers()
		default:
			if len(activeStreams) > 0 {
				broadcastMessage(cmd)
			} else {
				fmt.Println("unknown")
			}

		}
		fmt.Print("*")
	}
}

func megaparse(addr string) peer.AddrInfo {
	ai, err := peer.AddrInfoFromString(addr)
	if err != nil {
		panic(err)
	}
	return *ai
}

func connect(con host.Host, addrStr string) {
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

func setupMessageHandler(con host.Host) {
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

func sendToActivePeer(peerID peer.ID, message string) {
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

func broadcastMessage(message string) {
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

func listActivePeers() {
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

func showMyInfo(node host.Host) {
	fmt.Println("\n info")
	fmt.Println("ID:", node.ID())
	fmt.Println("full ID:", node.ID().String())

	fmt.Println("\n adresses:")
	for i, addr := range node.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr, node.ID())
		fmt.Printf("(%d) %s\n", i+1, fullAddr)
	}
}
