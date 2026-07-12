#!/bin/bash

# Cleanup first
echo "Cleaning up old processes..."
pkill -f "cmd/node/main.go"
lsof -ti:9001,9002,9003,8001,8002,8003 | xargs kill -9 2>/dev/null

echo "Starting Raft Cluster..."

# Start Followers first (Node 2, Node 3)
echo "Starting Node 2..."
go run cmd/node/main.go --id=node2 --http_addr=127.0.0.1:8002 --raft_addr=127.0.0.1:9002 --peers=127.0.0.1:9001,127.0.0.1:9003 > node2.log 2>&1 &
PID2=$!

echo "Starting Node 3..."
go run cmd/node/main.go --id=node3 --http_addr=127.0.0.1:8003 --raft_addr=127.0.0.1:9003 --peers=127.0.0.1:9001,127.0.0.1:9002 > node3.log 2>&1 &
PID3=$!

# Wait a moment for them to listen
sleep 2

# Start Leader/Candidate (Node 1)
echo "Starting Node 1..."
go run cmd/node/main.go --id=node1 --http_addr=127.0.0.1:8001 --raft_addr=127.0.0.1:9001 --peers=127.0.0.1:9002,127.0.0.1:9003 > node1.log 2>&1 &
PID1=$!

echo "Cluster running. PIDs: $PID1, $PID2, $PID3"
echo "Logs: node1.log, node2.log, node3.log"
echo "Press Enter to stop..."
read

kill $PID1 $PID2 $PID3
