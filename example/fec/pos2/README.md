# pos2

This experiment uses two machines and two phyiscal interfaces.

According to Francois Michel's thesis, he found a set of parameters for Direct Air-to-Ground communication (DA2GC) and Mobile Satellite Services (MSS). These are both in-flight communication technologies.

Bandwidth: [0.3, 10] Mbps
OWD: [100, 400] Mbs

Loss:
lowest: p=.01, r=.08, h=0, k=.98,
highest: p=.08, r=.5, h=.1, k=1,

- p will have a value between [1%, 8%]
  - p represents the transition probability from a good state to a bad state.
- r will have a value between [8%, 50%]
  - r represents the transition probabiltiy from a bad state to a good state.
- 1-h will have a value between [1-.1, 1-0] --> [90%, 100%]
  - 1-h represents the loss probability in the bad state.
- 1-k will have a value between [1-1, 1-.98] --> [0%, 2%]
  - 1-k represents the loss probability in the good state.

tc qdisc add dev eth0 root netem loss gemodel <p> [<r> [<1-h> [<1-k>]]]
tc qdisc add dev <INTERFACE> root netem loss gemodel 3% 40% 95% 1%

p=.03 --> 3%
r=.4 --> 40%
h=.05 --> 1-h=.95 --> 95%
k=.99 --> 1-k=.01 --> 99%

As known, p and r are the transition probabilities between the bad and the good states, 1-h is the loss probability in the bad state and 1-k is the loss probability in the good states

tc qdisc add dev <interface> root netem loss gemodel p <p> r <r> 1 <k> h <h>
lowest: tc qdisc add dev eth0 root netem loss gemodel p 0.01 r 0.08 1 0.98 h 0
highest: tc qdisc add dev eth0 root netem loss gemodel p 0.08 r 0.5 1 1 h 0.1
intermediate 1: tc qdisc add dev eth0 root netem loss gemodel p 0.03 r 0.2 1 0.99 h 0.05
intermediate 2: tc qdisc add dev eth0 root netem loss gemodel p 0.05 r 0.3 1 0.995 h 0.02
intermediate 3: tc qdisc add dev eth0 root netem loss gemodel p 0.07 r 0.4 1 0.999 h 0.08

These values are necessary to establish a Gilbert-Elliot loss model. 'p' represents the probability of going from a good state to a bad state. 'r' represents the probability of going from a bad state to a good state. 'k' represents the probability of staying in a good state. 'h' represents the probability of staying in the bad state.
