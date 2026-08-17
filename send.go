package main

import (
	"errors"
	//"fmt"
	"net/rpc"
)

func (n *Node) SendPut(addr string, key string, value Value) error {
	// n_id := curr.id
	// addr := curr.addr
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer client.Close()

	args := RPCArgs{
		Key:   key,
		Value: value,
	}

	var reply bool

	err = client.Call("Node.RemotePut", args, &reply)
	if err != nil {
		return err
	}

	if !reply {
		return errors.New("put failed")
	}

	return nil
}

func (n *Node) SendGet(addr string, key string) (Value, error) {
	// addr := curr.addr
	//fmt.Println("sent rpc")
	// fmt.Println(addr)
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		//fmt.Println("failed to open conn")
		return Value{}, err
	}
	defer client.Close()

	// args := RPCArgs{
	// 	Key:   key,
	// 	Value: "",
	// }

	args := key

	var reply RPCReply

	err = client.Call("Node.RemoteGet", args, &reply)
	if err != nil {
		//fmt.Println("failed to rpc")
		return Value{}, err
	}

	if reply.Exists {
		return reply.Value, nil
	} else {
		//fmt.Println("failed to find key")
		return Value{}, errors.New("key not found")
	}
}

func (n *Node) SendDelete(addr string, key string) error {
	// addr := curr.addr
	// fmt.Println(n_id)
	// fmt.Println(addr)
	client, err := rpc.Dial("tcp", addr)
	if err != nil {
		//fmt.Println("failed to open conn")
		return err
	}
	defer client.Close()

	// args := RPCArgs{
	// 	Key:   key,
	// 	Value: "",
	// }

	args := key

	var reply bool

	err = client.Call("Node.RemoteDelete", args, &reply)
	if err != nil {
		//fmt.Println("failed to rpc")
		return err
	}

	if reply {
		return nil
	} else {
		//fmt.Println("failed to find key")
		return errors.New("key not found")
	}
}
