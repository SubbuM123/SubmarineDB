package main

import (
	"bufio"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
	// "errors"
	"sync"
)

type Value struct {
	Data string
	ParentVideo string
	Descriptors map[string]struct{}
}

type Node struct {
    mu    sync.RWMutex

	hashtable map[string]Value
	pred_replica map[string]Value

	id        uint64
	addr      string
	
	fingers []Fingers

	successor_list []NodeInfo

	successor   NodeInfo
	predecessor NodeInfo
}

type NodeInfo struct {
	Id   uint64
	Addr string
}

type NodePredInfo struct {
	NI NodeInfo
	Fail bool
}

type NodeRef struct {
	id   uint64
	addr string
}

type RPCArgs struct {
	Key   string
	Value Value
}

type RPCReply struct {
	Value  Value
	Exists bool
}

type Fingers struct {
	Num uint64
	FingerNode NodeInfo
}

func hash(key string) uint64 {
	h := 0

	for _, i := range key {
		h += int(i)
	}
	return uint64(h % 256) // change 3 to n once more nodes join EDIT changeing to 256, chord w m = 8
}

func (n *Node) event_manage() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")

		if !scanner.Scan() {
			break // EOF or Ctrl+D
		}

		line := scanner.Text()
		args := strings.Fields(line)

		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "PUT":
			if len(args) != 3 {
				fmt.Println("Wrong arguments")
				continue
			}

			b := n.Put(args[1], Value{Data: args[2]})

			if b {
				fmt.Println("PUT succeeded")
			} else {
				fmt.Println("PUT failed")
			}

		case "GET":
			if len(args) != 2 {
				fmt.Println("Wrong arguments")
				continue
			}

			b, val := n.Get(args[1])

			if b {
				fmt.Println(val.Data)
			} else {
				fmt.Println("GET failed")
			}

		case "DELETE":
			if len(args) != 2 {
				fmt.Println("Wrong arguments")
				continue
			}

			b := n.Delete(args[1])

			if b {
				fmt.Println("DELETE succeeded")
			} else {
				fmt.Println("DELETE failed")
			}
		case "ls":
			for key, value := range n.hashtable {
				fmt.Println("key " + key)
				fmt.Println("Parent Video: " + value.ParentVideo)
				// fmt.Println("Value " + value.Data)
				for k := range value.Descriptors {
					fmt.Println("Desc " + k)
				}
				
			}
		case "lsrep":
			for key := range n.pred_replica {
				fmt.Println("key " + key)
			}
		case "s":
			fmt.Println(n.successor_list[0].Id)
			fmt.Println(n.successor_list[1].Id)
		case "f":
			for i := 0; i < 8; i++ {
				fmt.Println(n.fingers[i].FingerNode.Id)
			}
		case "n":
			fmt.Println(n.successor.Id)
			fmt.Println(n.predecessor.Id)
		case "EXIT":
			return

		default:
			fmt.Println("Invalid Request")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}
}

func (n *Node) JoinSystem(bs_addr string) error {
	if bs_addr == "" || bs_addr == "START" {
		n.successor = NodeInfo{Id: n.id, Addr: n.addr}
		n.predecessor = NodeInfo{Id: n.id, Addr: n.addr}
		n.updateSuccessorList(n.successor)
		log_updates(n.id, "Initialized as the first node")
		return nil
	}

	client, err := rpc.Dial("tcp", bs_addr)
	if err != nil {
		return err
	}
	defer client.Close()

	var successor NodeInfo
	if err := client.Call("Node.RemoteFindSuccessor", n.id, &successor); err != nil {
		return err
	}

	if successor.Addr == "" || successor.Id == 0 {
		n.successor = NodeInfo{Id: n.id, Addr: n.addr}
		n.predecessor = NodeInfo{Id: n.id, Addr: n.addr}
		return nil
	}

	n.successor = successor
	n.updateSuccessorList(successor)
	log_updates(n.id, "Successor assigned: "+strconv.Itoa(int(n.successor.Id)))

	info := NodeInfo{Id: n.id, Addr: n.addr}

	info2 := NodePredInfo { NI : info, Fail : false}

	var pred NodeInfo
	predClient, err := rpc.Dial("tcp", n.successor.Addr)
	if err != nil {
		return err
	}
	defer predClient.Close()

	if err := predClient.Call("Node.RemoteGetPredecessor", struct{}{}, &pred); err != nil || pred.Id == 0 || pred.Addr == "" {
		pred = successor
	}

	n.predecessor = pred
	log_updates(n.id, "Predecessor assigned: "+strconv.Itoa(int(n.predecessor.Id)))

	var reply bool
	if err := predClient.Call("Node.RemoteNotifySuccessor", info2, &reply); err != nil {
		return err
	}

	if pred.Addr != "" {
		if pred.Addr == successor.Addr {
			if err := predClient.Call("Node.RemoteNotifyPredecessor", info, &reply); err != nil {
				return err
			}
		} else {
			oldPredClient, err := rpc.Dial("tcp", pred.Addr)
			if err != nil {
				return err
			}
			defer oldPredClient.Close()

			if err := oldPredClient.Call("Node.RemoteNotifyPredecessor", info, &reply); err != nil {
				return err
			}
		}
	}

	// time.Sleep(50 * time.Millisecond)
	// n.updateSuccessorList(n.successor)

	return nil
}

func (n *Node) updateSuccessorList(succ NodeInfo) {
	if len(n.successor_list) != 2 {
		n.successor_list = make([]NodeInfo, 2)
	}

	if succ.Id == 0 || succ.Addr == "" {
		n.successor_list[0] = n.successor
		n.successor_list[1] = n.successor
		return
	}

	n.successor_list[0] = succ

	var next NodeInfo
	for attempt := 0; attempt < 2; attempt++ {
		client, err := rpc.Dial("tcp", succ.Addr)
		if err == nil {
			if err := client.Call("Node.RemoteGetSuccessor", struct{}{}, &next); err == nil && next.Id != 0 && next.Addr != "" {
				client.Close()
				break
			}
			client.Close()
		}

		if attempt < 1 {
			time.Sleep(50 * time.Millisecond)
		}
	}

	if next.Id == 0 || next.Addr == "" || next.Addr == succ.Addr {
		next = succ
	}

	n.successor_list[1] = next
}

func (n *Node) Stabilize() {
	ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        n.updateSuccessorList(n.successor)
    }
}

func (n *Node) DiskBackup() {
	ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        n.BackupData()
    }
}

// func (n *Node) UpdatePred(pred NodeInfo) {
// 	if pred.Id == 0 || pred.Addr == "" {
// 		return
// 	}

// 	// line 288 and 293-298 are an or, only ones needs to be done.use context to figure it out and proceed

// 	// if a new node is added and we call update pred, we can call the loop
// 	// or tbh just add a flag
// 	n.TransferReplicaData()
	
// 	n.predecessor.Addr = pred.Addr
// 	n.predecessor.Id = pred.Id

// 	for key, value := range n.hashtable {
// 		if hash(key) <= pred.Id {
// 			n.SendPut(pred.Addr, key,value)
// 			n.Delete(key)
// 		}
// 	}
// 	log_updates(n.id, "Predecessor updated: "+strconv.Itoa(int(n.predecessor.Id)))
// }

func (n *Node) UpdatePred(pred NodeInfo, fail bool) {
	if pred.Id == 0 || pred.Addr == "" {
		return
	}

	if fail {
		n.TransferReplicaData()
	}

	// line 288 and 293-298 are an or, only ones needs to be done.use context to figure it out and proceed

	// if a new node is added and we call update pred, we can call the loop
	// or tbh just add a flag
	
	
	n.predecessor.Addr = pred.Addr
	n.predecessor.Id = pred.Id
	if !fail {
		for key, value := range n.hashtable {
			if hash(key) <= pred.Id {
				n.SendPut(pred.Addr, key,value)
				n.Delete(key)
			}
		}
	}
	
	log_updates(n.id, "Predecessor updated: "+strconv.Itoa(int(n.predecessor.Id)))
}

func (n *Node) UpdateSucc(succ NodeInfo) {
	if succ.Id == 0 || succ.Addr == "" {
		return
	}
	n.successor.Addr = succ.Addr
	n.successor.Id = succ.Id
	log_updates(n.id, "Successor updated: "+strconv.Itoa(int(n.successor.Id)))

	n.updateSuccessorList(succ)
}

// func (n *Node) RemoteNotifySuccessor(pred NodeInfo, reply *bool) error {
// 	n.UpdatePred(pred)

// 	*reply = true
// 	return nil
// }

func (n *Node) RemoteNotifySuccessor(pred NodePredInfo, reply *bool) error {
	n.UpdatePred(pred.NI, pred.Fail)

	*reply = true
	return nil
}

func (n *Node) RemoteNotifyPredecessor(succ NodeInfo, reply *bool) error {
	n.UpdateSucc(succ)

	*reply = true
	return nil
}

func (n *Node) GetPredecessor() NodeInfo {
	return n.predecessor
}

func (n *Node) GetSuccessor() NodeInfo {
	return n.successor
}

func (n *Node) RemoteGetPredecessor(_ struct{}, reply *NodeInfo) error {
	pred := n.GetPredecessor()

	*reply = pred
	return nil
}

func (n *Node) RemoteGetSuccessor(_ struct{}, reply *NodeInfo) error {
	succ := n.GetSuccessor()

	*reply = succ
	return nil
}

// JoinSystem
// pass in bootstrap, current node
// bootstrap returns successor
// need to notify successor, and pred of successor
// notifySucc, notifyPred
// data transfer later

// func main() {
// 	// start with the following format "./node id ip" -> "./node 1 127.0.0.1:8001"
// 	name  := os.Args[1]
// 	log_addr := os.Args[2]
// 	fmt.Println(name + log_addr)

// 	// hashtable map[uint64]string
// 	// id uint64
// 	// neighbors map[uint64]string
// 	node_id, _ := strconv.ParseUint(name, 10, 64)

// 	n := Node{
//         hashtable: make(map[uint64]string),
//         id: uint64(node_id),
// 		neighbors: make(map[uint64]string),
//     }

// 	n.neighbors[1] = "127.0.0.1:8001"
// 	n.neighbors[2] = "127.0.0.1:8002"
// 	n.neighbors[3] = "127.0.0.1:8003"

// 	rpc.Register(&n)

//     // Start listening
//     listener, err := net.Listen("tcp", log_addr)
//     if err != nil {
//         log.Fatal(err)
//     }

//     // Accept RPC connections in the background
//     go func() {
//         for {
//             conn, err := listener.Accept()
//             if err != nil {
//                 continue
//             }
//             go rpc.ServeConn(conn)
//         }
//     }()

// 	n.event_manage()
// 	// switch {}
// }
