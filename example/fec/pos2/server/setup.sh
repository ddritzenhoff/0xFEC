#!/bin/bash

ip addr add 22.22.22.22/32 dev enp2s0f0
ip link set dev enp2s0f0 up
ip route add 11.11.11.0/24 dev enp2s0f0

# Add network conditions on egress
NETWORK_CONDITION="$(pos_get_variable condition)"
if [ "$NETWORK_CONDITION" == "GE" ]; then
    tc qdisc add dev enp2s0f0 root netem delay 100ms loss gemodel 3% 40% 95% 1% rate 1mbit
    echo "Gilbert-Elliot applied." >> condition.txt
else
    tc qdisc add dev enp2s0f0 root netem delay 100ms loss 5% rate 1mbit
    echo "Constant 5% loss applied" >> condition.txt
fi

SCHEME="$(pos_get_variable scheme)"
./server-linux-x86_64 -scheme="$SCHEME" >> condition.txt 2>&1
