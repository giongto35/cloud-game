#! /bin/bash

iplist=$1

for ip_address in $iplist
do
    echo $ip_address
    ssh-keyscan -H $ip_address >> ~/.ssh/known_hosts
    sleep 2

    if [ "$ip_address" == "167.172.70.98" ] || [ "$ip_address" == "cloudretro.io" ]
    then
        launchcommand="coordinator > /tmp/startup.log"
    else
        launchcommand="Xvfb :99 & worker --coordinatorhost cloudretro.io --zone \$zone > /tmp/startup.log"
    fi

    ssh root@$ip_address "mkdir -p /cloud-game"
    rsync ./.github/workflows/redeploy/.env root@$ip_address:/cloud-game/.env
    run_content="'#! /bin/bash
    echo $PASSWORD | docker login https://docker.pkg.github.com --username $USERNAME --password-stdin;
    ufw disable;
    docker system prune -f;
    source /etc/profile;
    docker pull docker.pkg.github.com/giongto35/cloud-game/cloud-game:latest;
    docker rm cloud-game -f;
    docker run --privileged -d \
      --network=host \
      --env DISPLAY=:99 \
      --env MESA_GL_VERSION_OVERRIDE=3.3 \
      --env-file .env \
      -v cores:/usr/local/share/cloud-game/assets/cores \
      -v /cloud-game/games:/usr/local/share/cloud-game/assets/games \
      -v /cloud-game/cache:/usr/local/share/cloud-game/assets/cache \
      --name cloud-game \
      docker.pkg.github.com/giongto35/cloud-game/cloud-game \
      bash -c \"$launchcommand\"'"

    ssh root@$ip_address "echo $run_content > ~/run.sh"
    ssh root@$ip_address "chmod +x run.sh; ./run.sh"

done

echo 'done'
