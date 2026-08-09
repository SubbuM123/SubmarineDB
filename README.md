# SubmarineDB - Personal Cloud Video Catalog

SubmarineDB is a small Chord-like distributed hash table (DHT) for storing chunked video data across multiple nodes. The project combines a Go-based node runtime with a Python client so you can upload video files, retrieve them later, search by YOLO-detected object tags, and download the reconstructed output.

This project was forked from https://github.com/SubbuM123/DistributedHashTable, an old project of mine. This project expands upon the old one by added support for video storage, advanced CLI, video tagging, rather than key-value storage. The underlying DHT system is mostly unchanged from the previous project, though.

## What this project does

- Runs a distributed key-value store across multiple Go nodes
- Uses a Chord-style ring with successor/predecessor tracking
- Stores video chunks, parent-video information, and YOLO-detected object tags for uploaded files
- Supports replication, backup logging, and simple failure detection
- Exposes HTTP endpoints for Python-based file upload/download/search workflows

## Repository layout

- main.go: entry point for starting a DHT node
- node.go: core node state and join logic
- chord.go: Chord-style successor/finger table logic
- crud.go: put/get/delete/search operations
- endpoints.go: HTTP handlers for Python integration
- replica.go: replica synchronization logic
- fd.go: failure detector and successor failover logic
- logger.go: logging and backup files
- app.py: Python client for upload/download/search workflows

## Tech stack

SubmarineDB uses a lightweight distributed systems stack built around Go and Python. The Go service implements the DHT node logic, RPC-based communication, and ring maintenance, while the Python layer handles video chunking, HTTP requests, and YOLO-assisted object tagging. The project also relies on standard networking, file I/O, and optional computer-vision libraries for video analysis.

## Prerequisites

- Go 1.26 or newer
- Python 3
- Python packages for video tagging and client communication

Install the Python dependencies with:

```bash
pip install -r requirements.txt
```

## Build the Go binary

From the repository root:

```bash
git clone <repo-url>
cd SubmarineDB
go build -o dht .
```

## Start the DHT nodes

The node startup command is:

```bash
./dht <bootstrap-or-start> <node-id>
```

Examples:

1. Start the first/bootstrap node:

```bash
./dht START 1
```

2. Start a second node that joins node 1:

```bash
./dht 1 2
```

3. Start additional nodes as needed:

```bash
./dht 1 3
```

Node IDs map to local ports in the Go code:

- Node 1 uses RPC on port 8001 and HTTP on port 7001
- Node 2 uses RPC on port 8002 and HTTP on port 7002
- Node 3 uses RPC on port 8003 and HTTP on port 7003

If you use a different node-id/port layout, update the server list in app.py accordingly.

## Run the Python client

In another terminal, start the Python interface:

```bash
python app.py
```

The client supports these commands:

- put <file>: upload a file and split it into chunks
- get <file>: download a previously uploaded file
- search <tag>: search for files by YOLO-detected object tag
- get_tag <tag>: search and download matching files
- clear: clear the output folder
- view <file>: open a downloaded file
- exit: quit the client

Example workflow:

```text
put sample.mp4
search car
get_tag car
```

YOLO-based tagging means each video can be annotated with object labels such as person, car, dog, or other detected classes. Those labels are used as searchable tags across the DHT.

## Optional direct DHT CLI

If you want a simpler, low-level test interface, you can also run:

```bash
python kv_app.py
```

That client supports:

```text
put <key> <value>
get <key>
delete <key>
exit
```

## Logging and output

- System logs are written to logs/system_log.txt
- Data-operation logs are written to logs/data_log.txt
- Backup snapshots are written to logs/backup_<node-id>.txt
- Downloaded files are placed in the output/ directory

## Notes

- This is a prototype system and is intended for local development and experimentation.
- The Python client assumes the DHT nodes are reachable at the addresses configured in app.py.
- For a real deployment, you would want stronger node discovery, better load balancing, and a more robust replication strategy.
