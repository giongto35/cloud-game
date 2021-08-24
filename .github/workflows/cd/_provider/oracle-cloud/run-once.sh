#!/bin/bash
#sudo

iptables -P INPUT ACCEPT
iptables -P OUTPUT ACCEPT
iptables -P FORWARD ACCEPT
iptables -F

iptables --flush

which "iptables-persistent" > /dev/null 2>&1
if [ $? = 0 ]; then
  apt-get install iptables-persistent
fi
netfilter-persistent save
