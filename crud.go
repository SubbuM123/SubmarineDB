package main

// import "fmt"

func (n *Node) Insert(key string, value Value) {
	n.mu.Lock()
    defer n.mu.Unlock()
	n.hashtable[key] = value
}

func (n *Node) Find(key string) (Value, bool) {
	n.mu.RLock()
    defer n.mu.RUnlock()
	val, exists := n.hashtable[key]
	return val, exists
}

func (n *Node) Remove(key string) {
	n.mu.Lock()
    defer n.mu.Unlock()
	delete(n.hashtable, key)
}

func (n *Node) Put(key string, value Value) bool {
	h := hash(key)
	if n.Owns(h) {
		log_data(n.id, "PUT "+key+", "+value.Data)
		n.Insert(key, value)
	} else {
		err := n.SendPut(n.FindOwner(h), key, value)

		if err != nil {
			return false
		}
	}
	return true
}

// func (n *Node) Get(key string) (bool, Value) {
// 	h := hash(key)
// 	if n.Owns(h) {
// 		if value, exists := n.hashtable[key]; exists {
// 			log_data(n.id, "GOT "+key)
// 			return true, value
// 		}
// 		return false, Value{}
// 	} else {
// 		val, err := n.SendGet(n.FindOwner(h), key)

// 		if err != nil {
// 			return false, Value{}
// 		}
// 		return true, val
// 	}
// }

// func (n *Node) Delete(key string) bool {
// 	h := hash(key)
// 	if n.Owns(h) {
// 		if _, exists := n.hashtable[key]; exists {
// 			log_data(n.id, "DELETED "+key)
// 			delete(n.hashtable, key)
// 			return true
// 		}
// 		return false
// 	} else {
// 		err := n.SendDelete(n.FindOwner(h), key)
// 		// fmt.Println(err)
// 		if err != nil {
// 			return false
// 		}
// 	}
// 	return true
// }

func (n *Node) Get(key string) (bool, Value) {
	h := hash(key)
	if n.Owns(h) {
		if value, exists := n.Find(key); exists {
			log_data(n.id, "GOT "+key)
			return true, value
		}
		return false, Value{}
	} else {
		val, err := n.SendGet(n.FindOwner(h), key)
		if err != nil {
			return false, Value{}
		}
		return true, val
	}
}

func (n *Node) Delete(key string) bool {
	h := hash(key)
	if n.Owns(h) {
		if _, exists := n.Find(key); exists {
			log_data(n.id, "DELETED "+key)
			n.Remove(key)
			return true
		}
		return false
	} else {
		err := n.SendDelete(n.FindOwner(h), key)
		if err != nil {
			return false
		}
	}
	return true
}

func (n *Node) Search(tag string) []string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	res := []string{}
	for _, value := range n.hashtable {
		if _, exists := value.Descriptors[tag]; exists {
			res = append(res, value.ParentVideo)
		}
	}
	return res
}