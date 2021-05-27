#! /bin/bash

iplist="cloudretro.io"
for tagName in cloud-gaming cloud-gaming-eu cloud-gaming-usw; do
    echo "scanning: $tagName"
    regional_iplist=$(curl -X GET -H "Content-Type: application/json" -H "Authorization: Bearer "$DO_TOKEN "https://api.digitalocean.com/v2/droplets?tag_name=$tagName" | jq -r ".droplets[]" | jq -r ".networks.v4[0].ip_address")

    for ip_address in $regional_iplist
    do
	iplist+=" $ip_address"
    done
done

echo "iplist "$iplist

for ip_address in $iplist
do
    .github/workflows/redeploy/redeploy_specific.sh $ip_address
done

echo 'done'
