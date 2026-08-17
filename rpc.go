package main

// import ("fmt")

func (n *Node) RemotePut(args RPCArgs, reply *bool) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.hashtable[args.Key] = args.Value
	*reply = true
	// TODO add errors if key doesnt exist
	return nil
}

func (n *Node) RemoteGet(args string, reply *RPCReply) error {
	n.mu.RLock()
	defer n.mu.RUnlock()
	value, exists := n.hashtable[args]
	reply.Value = value
	reply.Exists = exists

	return nil
}

func (n *Node) RemoteDelete(args string, reply *bool) error {
	//fmt.Println(args.Key)
	n.mu.Lock()
	defer n.mu.Unlock()
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