# setup ip addresses
ip addr add 11.11.11.11/32 dev enp2s0f0
ip link set dev enp2s0f0 up
ip route add 22.22.22.0/24 dev enp2s0f0

# add network conditions on egress
tc qdisc add dev enp2s0f0 root netem delay 100ms loss 1% rate 1mbit