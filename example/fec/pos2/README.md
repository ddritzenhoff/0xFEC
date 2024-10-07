# pos2

This experiment uses two machines and two phyiscal interfaces.

According to Francois Michel's thesis, he found a set of parameters for Direct Air-to-Ground communication (DA2GC) and Mobile Satellite Services (MSS). These are both in-flight communication technologies.

Bandwidth: [0.3, 10] Mbps
OWD: [100, 400] Mbs

Loss:
lowest: p=.01, r=.08, k=.98, h=0
highest: p=.08, r=.5, k=1, h=.1

These values are necessary to establish a Gilbert-Elliot loss model. 'p' represents the probability of going from a good state to a bad state. 'r' represents the probability of going from a bad state to a good state. 'k' represents the probability of staying in a good state. 'h' represents the probability of staying in the bad state.
