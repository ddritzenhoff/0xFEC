# pos2

This experiment uses two machines and two phyiscal interfaces.

The destination network address --> this will be the IP I assign to tartu and tallinn
Name of the local interface connected to the destination network.

Assign an IP address to a physical address
Add a route for the interface
Bring the interface up

Client:
`ip addr add 11.11.11.11/32 dev enp2s0f0`
`ip link set dev enp2s0f0 up`
`ip route add 22.22.22.0/24 dev enp2s0f0`

Server:
`ip addr add 22.22.22.22/32 dev enp2s0f0`
`ip link set dev enp2s0f0 up`
`ip route add 11.11.11.0/24 dev enp2s0f0`

There are two interfaces provided on both Tallinn and Tartu, but I'm only using one of them as its full duplex.
