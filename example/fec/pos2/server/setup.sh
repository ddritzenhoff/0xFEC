#!/bin/bash

ip addr add 22.22.22.22/32 dev enp2s0f0
ip link set dev enp2s0f0 up
ip route add 11.11.11.0/24 dev enp2s0f0

tc qdisc add dev enp2s0f0 root netem delay 100ms loss 5% rate 1mbit

SCHEME="$(pos_get_variable scheme)"
./server-linux-x86_64 -scheme="$SCHEME"
