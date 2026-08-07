package main

// import ("fmt")

func (n *Node) RemotePut(args RPCArgs, reply *bool) error {
	n.hashtable[args.Key] = args.Value
	*reply = true
	// TODO add errors if key doesnt exist
	return nil
}

func (n *Node) RemoteGet(args string, reply *RPCReply) error {
	value, exists := n.hashtable[args]
	reply.Value = value
	reply.Exists = exists

	return nil
}

func (n *Node) RemoteDelete(args string, reply *bool) error {
	//fmt.Println(args.Key)
	_, exists := n.hashtable[args]
	
	if exists {
		delete(n.hashtable, args)
		//fmt.Println("true it existed, and deleted it")
		*reply = true
	} else {
		//fmt.Println("false it didnt existed")
		*reply = false
	}

	return nil
}