#!/bin/bash
INTERFACE="$(pos_get_variable interface)"
ip addr add 22.22.22.22/32 dev "$INTERFACE"
ip link set dev "$INTERFACE" up
ip route add 11.11.11.0/24 dev "$INTERFACE"

# Add network conditions on egress
NETWORK_CONDITION="$(pos_get_variable condition)"
if [ "$NETWORK_CONDITION" == "GE" ]; then
    tc qdisc add dev "$INTERFACE" root netem delay 100ms loss gemodel 3% 40% 95% 1% rate 1mbit
    echo "Gilbert-Elliot applied." >> condition.txt
elif [ "$NETWORK_CONDITION" == ".01" ]; then
    tc qdisc add dev "$INTERFACE" root netem delay 50ms loss 0.01% rate 50mbit
    echo "Constant .01% loss applied" >> condition.txt
elif [ "$NETWORK_CONDITION" == ".1" ]; then
    tc qdisc add dev "$INTERFACE" root netem delay 50ms loss 0.1% rate 50mbit
    echo "Constant .1% loss applied" >> condition.txt
elif [ "$NETWORK_CONDITION" == "1" ]; then
    tc qdisc add dev "$INTERFACE" root netem delay 50ms loss 1% rate 50mbit
    echo "Constant 1% loss applied" >> condition.txt
else
    tc qdisc add dev "$INTERFACE" root netem delay 100ms loss 5% rate 1mbit
    echo "Constant 5% loss applied" >> condition.txt
fi

SCHEME="$(pos_get_variable scheme)"
./server-linux-x86_64 -scheme="$SCHEME" >> condition.txt 2>&1
