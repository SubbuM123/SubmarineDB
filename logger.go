package main

import (
	"os"
	"fmt"
	"strconv"
)

func log_updates(node_id uint64, update string) {
	file, err := os.OpenFile(
		"logs/system_log.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	fmt.Fprintf(
		file,
		"Node %d > %s \n",
		node_id,
		update,
	)
}

func log_data(node_id uint64, update string) {
	file, err := os.OpenFile(
		"logs/data_log.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	fmt.Fprintf(
		file,
		"Node %d > %s \n",
		node_id,
		update,
	)
}
// change this to backup data
func (n *Node) BackupData() {
	name := "logs/backup_" + strconv.Itoa(int(n.id)) + ".txt"
	file, err := os.OpenFile(
		name,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0644,
	)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	fmt.Fprintf(
		file,
		"\n Node %d BACKUP \n",
		n.id,
	)

	for key, value := range n.hashtable {
		fmt.Fprintf(
		file,
		"%s : %s \n",
		key,
		value.Data,)
	}
	
}