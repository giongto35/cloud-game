#!/bin/sh
#sudo

iptables -P INPUT ACCEPT
iptables -P OUTPUT ACCEPT
iptables -P FORWARD ACCEPT
iptables -F

iptables --flush

iptables-save > /etc/iptables.conf

f=~/.iptables.lock
if [ ! -e "$f" ]; then
  echo "iptables-restore < /etc/iptables.conf" > /etc/rc.local
  touch "$f"
fi
