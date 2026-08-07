package main

import ("time"
		"net/rpc"
		"errors")


type ReplicaArgs struct {
	Replica map[string]Value
}

func (n *Node) Replication() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		n.SendReplica()
	}
}

func (n *Node) SendReplica() error {
	client, err := rpc.Dial("tcp", n.successor.Addr)
	if err != nil {
		return err
	}
	defer client.Close()

	args := ReplicaArgs{
		Replica:   n.hashtable,
	}

	var reply bool

	err = client.Call("Node.RemoteUpdateReplica", args, &reply)
	if err != nil {
		return err
	}

	if !reply {
		return errors.New("Replication Failed")
	}

	return nil
}

func (n *Node) RemoteUpdateReplica(args ReplicaArgs, reply *bool) error {
	n.UpdateReplica(args.Replica)
    *reply = true
	// TODO add errors if key doesnt exist
    return nil
}

func (n *Node) UpdateReplica(rep map[string]Value) {
	n.pred_replica = rep
}