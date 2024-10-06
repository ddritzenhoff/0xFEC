# setup ip addresses
ip addr add 11.11.11.11/32 dev enp2s0f0
ip link set dev enp2s0f0 up
ip route add 22.22.22.0/24 dev enp2s0f0

# add network conditions on egress
tc qdisc add dev enp2s0f0 root netem delay 100ms loss 1% rate 1mbit

pos_sync

# give the server enough time to boot up
sleep 0.5
ENDPOINT="$(pos_get_variable endpoint)"
ROUNDS="$(pos_get_variable rounds)"
for ROUND in $ROUNDS; do
    ./client-linux-x86_64 -file=output.txt -round="$ROUND" -host="$SERVER_IP_WITH_CIDR" -endpoint="$ENDPOINT"
