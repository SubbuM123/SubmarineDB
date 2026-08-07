package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
)

func main() {
	// start with the following format "./node bs id ip" -> "./node 127.0.0.1:8033 1 127.0.0.1:8001"
	// for starting node, bs can be START
	bs := os.Args[1]
	id := os.Args[2]
	// log_addr := os.Args[3]
	node_id, _ := strconv.ParseUint(id, 10, 64)
	log_addr := "127.0.0.1:" + strconv.Itoa(int(8000 + node_id))
	py_addr := "127.0.0.1:" + strconv.Itoa(int(7000 + node_id))

	node_bs, _ := strconv.ParseUint(bs, 10, 64)
	bs_addr := "127.0.0.1:" + strconv.Itoa(int(8000 + node_bs))

	n := Node{
		hashtable:      make(map[string]Value),
		successor_list: make([]NodeInfo, 2),
		id:             uint64(node_id),
		addr:           log_addr,
		// neighbors: make(map[uint64]string),
	}

	log_updates(n.id, "Joined the system, yet to be placed")

	rpc.Register(&n)

	// Start listening
	listener, err := net.Listen("tcp", log_addr)
	if err != nil {
		log.Fatal(err)
	}

	// Accept RPC connections on the existing startup port.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()

	http.HandleFunc("/put_chunk", n.put_chunk)
	http.HandleFunc("/get_chunk", n.get_chunk)
	http.HandleFunc("/delete_chunk", n.delete_chunk)
	go func() {
		log.Printf("HTTP endpoint listening on " + py_addr)
		if err := http.ListenAndServe(py_addr, nil); err != nil {
			log.Printf("http server stopped: %v", err)
		}
	}()

	if err := n.JoinSystem(bs_addr); err != nil {
		log_updates(n.id, "Join failed: "+err.Error())
	}
	n.fingers = make([]Fingers, 8)

	// make ft here

	go n.FailureDetector()
	go n.MakeFingerTable()
	go n.Stabilize()
	go n.Replication()
	go n.DiskBackup()
	n.event_manage()
	// switch {}
}
