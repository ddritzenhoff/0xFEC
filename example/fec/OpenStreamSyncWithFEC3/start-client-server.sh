#!/bin/bash

# Directories for server and client source code
SERVER_DIR="./server"
CLIENT_DIR="./client"

# Binary paths
SERVER_BINARY="$SERVER_DIR/server"
CLIENT_BINARY="$CLIENT_DIR/client"

# Build the server binary
echo "Building server..."
go build -o $SERVER_BINARY $SERVER_DIR

# Check if server binary was built successfully
if [ ! -f "$SERVER_BINARY" ]; then
    echo "Failed to build server binary."
    exit 1
fi

# Build the client binary
echo "Building client..."
go build -o $CLIENT_BINARY $CLIENT_DIR

# Check if client binary was built successfully
if [ ! -f "$CLIENT_BINARY" ]; then
    echo "Failed to build client binary."
    exit 1
fi

# Start the server in the background and capture its stdout
echo "Starting server..."
$SERVER_BINARY -addr="localhost:4241" -len=22000 > server_output.txt &
SERVER_PID=$!

# Wait for the server to initialize
sleep 1

# Start the client in the background and capture its stdout
echo "Starting client..."
$CLIENT_BINARY -addr="localhost:4241" -len=22000 > client_output.txt &
CLIENT_PID=$!

# Function to kill server and client on script exit, remove binaries, and compare outputs
cleanup() {
    echo "Stopping server and client..."
    if kill -0 $SERVER_PID 2>/dev/null; then
        kill $SERVER_PID
        wait $SERVER_PID
    fi
    if kill -0 $CLIENT_PID 2>/dev/null; then
        kill $CLIENT_PID
        wait $CLIENT_PID
    fi
    wait $SERVER_PID
    wait $CLIENT_PID

    echo "Removing server and client binaries..."
    rm $SERVER_BINARY
    rm $CLIENT_BINARY

    # Directly compare the output files
    if diff server_output.txt client_output.txt > /dev/null; then
        echo "Server and client outputs are identical."
    else
        echo "Server and client outputs differ."
        # Optionally, show the differences
        diff server_output.txt client_output.txt
    fi

    # Cleanup output files
    rm -f server_output.txt client_output.txt
}

# Trap script exit for cleanup
trap cleanup EXIT

# Wait for server and client to exit
wait $SERVER_PID
wait $CLIENT_PID