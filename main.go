package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	connection2 "plasitol.dev/ootb/connection"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	//need to test MORE (DHT in WIP)
)

func main() {
	con, err := libp2p.New(
		//libp2p.EnableRelay(), // relay
		//libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{
		//megaparse(""), //add more + config add wip
		//}),
		libp2p.EnableHolePunching(), //nat punching
		libp2p.NATPortMap(),

		//todo
	)
	if err != nil {
		panic(err)
	}
	defer con.Close()

	//WIP DHT
	//anoctx := context.Background()
	//kdht, err := dht.New(anoctx, con, dht.Mode(dht.ModeServer))
	//if err != nil {
	//	panic(err)
	//}

	//if err = kdht.Bootstrap(anoctx); err != nil {
	//	panic(err)
	//}

	//bootstrapPeers := []string{ //i stealed this
	//}

	//for _, addr := range bootstrapPeers {
	//	peerinfo, _ := peer.AddrInfoFromString(addr)
	//	if err := con.Connect(anoctx, *peerinfo); err != nil {
	//		fmt.Printf("failed to connect to bootstrap %v \n", err)
	//	}
	//}

	connection2.SetupMessageHandler(con) //i stealed this

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
				connection2.Connectp2p(con, parts[1])
			}
		case "/peers":
			connection2.ListActivePeers()
		default:
			if connection2.HasActiveStreams() {
				connection2.BroadcastMessage(cmd)
			} else {
				fmt.Println("unknown")
			}
		}
		fmt.Print("*")
	}
}

//func megaparse(addr string) peer.AddrInfo {
//	ai, err := peer.AddrInfoFromString(addr)
//	if err != nil {
//		panic(err)
//	}
//	return *ai
//}

func showMyInfo(con host.Host) {
	fmt.Println("\n info")
	fmt.Println("ID:", con.ID())
	fmt.Println("full ID:", con.ID().String())

	fmt.Println("\n adresses:")
	for i, addr := range con.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr, con.ID())
		fmt.Printf("(%d) %s\n", i+1, fullAddr)
	}
}
