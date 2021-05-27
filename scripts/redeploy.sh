#! /bin/bash

./build_image.sh

iplist="47.244.229.182"
for tagName in cloud-gaming cloud-gaming-eu cloud-gaming-usw; do
    echo "scanning: $tagName"
    regional_iplist=$(curl -X GET -H "Content-Type: application/json" -H "Authorization: Bearer "$DO_TOKEN "https://api.digitalocean.com/v2/droplets?tag_name=$tagName" | jq -r ".droplets[]" | jq -r ".networks.v4[0].ip_address")

    for ip_address in $regional_iplist
    do
	iplist+=" $ip_address"
    done
done

echo "iplist "$iplist

 #change /etc/ssh/ssh_config StrictHostKeyChecking to accept-new
for ip_address in $iplist
do
    ./redeploy_specific.sh $ip_address
done

echo 'done'
