#!/usr/bin/env bash

# stop program in case of error, treat unset variables as an error, and print out executed commands
set -eux

# Create two network namespaces.
ip netns add ns_client
ip netns add ns_server

# create a virtual ethernet interface pair. These will link the two network namespaces.
ip link add veth_client type veth peer name veth_server

# Move veth_client to the ns_client namespace and veth_server to the ns_server namespace:
ip link set veth_client netns ns_client
ip link set veth_server netns ns_server

# bring up the interfaces. 
ip netns exec ns_client ip link set dev veth_client up
ip netns exec ns_server ip link set dev veth_server up

# Assign IP addresses to each interface in the respective namespaces:
CLIENT_IP_WITH_CIDR="$(pos_get_variable client-ip)"
SERVER_IP_WITH_CIDR="$(pos_get_variable server-ip)"
ip netns exec ns_client ip addr add "$CLIENT_IP_WITH_CIDR" dev veth_client
ip netns exec ns_server ip addr add "$SERVER_IP_WITH_CIDR" dev veth_server

# Adding network conditions like delay, loss, and bandwidth limits using tc and netem
ip netns exec ns_client tc qdisc add dev veth_client root netem delay 100ms loss 1% rate 1mbit
ip netns exec ns_server tc qdisc add dev veth_server root netem delay 50ms rate 1mbit

# Execute binaries within the network namespaces
ip netns exec ns_server ./server-linux-x86_64 &

ENDPOINT="$(pos_get_variable endpoint)"
ROUNDS="$(pos_get_variable rounds)"
for ROUND in $ROUNDS; do
    ip netns exec ns_client ./client-linux-x86_64 -file=output.txt -round="$ROUND" -host="$SERVER_IP_WITH_CIDR" -endpoint="$ENDPOINT"
