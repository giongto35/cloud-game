#! /bin/bash

iplist=$1

for ip_address in $iplist
do
    echo $ip_address
    ssh-keyscan -H $ip_address >> ~/.ssh/known_hosts
    sleep 2

    if [ "$ip_address" == "cloudretro.io" ]
    then
        launchcommand="coordinator > /tmp/startup.log"
        httpport=8000
    else
        launchcommand="Xvfb :99 & worker --coordinatorhost cloudretro.io --zone \$zone > /tmp/startup.log"
        httpport=9000
    fi

    ssh root@$ip_address "mkdir -p /cloud-game/configs"
    rsync ./.github/workflows/redeploy/config.yaml root@$ip_address:/cloud-game/configs/config.yaml
    run_content="'#! /bin/bash
    ufw disable;
    iptables -t nat -F;
    iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port $httpport; iptables-save;
    echo $PASSWORD | docker login https://docker.pkg.github.com --username $USERNAME --password-stdin; 
    docker system prune -f;
    source /etc/profile; docker stop cloud-game || true; docker rm cloud-game || true;
    docker pull docker.pkg.github.com/giongto35/cloud-game/cloud-game:latest;

    docker run --privileged -d --network=host --env DISPLAY=:99 --env MESA_GL_VERSION_OVERRIDE=3.3 -v cores:/usr/local/share/cloud-game/assets/cores -v /cloud-game/configs:/usr/local/share/cloud-game/configs -v /cloud-game/games:/usr/local/share/cloud-game/assets/games -v /cloud-game/cache:/usr/local/share/cloud-game/assets/cache --name cloud-game docker.pkg.github.com/giongto35/cloud-game/cloud-game bash -c \"$launchcommand\"'"
    #docker run --privileged -d --network=host -v /cloud-game/games:/cloud-game/assets/games -v /cloud-game/cache:/cloud-game/assets/cache -v /cloud-game/conf.d:/etc/supervisor/conf.d --name cloud-game -e zone=\$zone giongto35/cloud-game-prod supervisord > /tmp/startup.log'"

    ssh root@$ip_address "echo $run_content > ~/run.sh"
    ssh root@$ip_address "chmod +x run.sh; ./run.sh"

done

echo 'done'
