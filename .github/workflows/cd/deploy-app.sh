#!/bin/sh

# parse commandline arguments
for arg in "$@"
do
    case $arg in
        -e=*|--env-dir=*)
        ENV_DIR="${arg#*=}"
        shift # Remove --ssh-key= from processing
        ;;
        -s=*|--ssh-key=*)
        SSH_KEY="${arg#*=}"
        shift # Remove --ssh-key= from processing
        ;;
        *)
        REST_ARGUMENTS+=("$1")
        shift # Remove generic argument from processing
        ;;
    esac
done

# Environment merging
#
# Import optional script.env file.
# This file contains script runtime params.
if [[ ! -z "${ENV_DIR}" ]]; then
  f="$ENV_DIR/script.env"
  if [[ -e "$f" ]]
      then
        echo $'\n'"script.env:"
        cat "$f"
        set -a
        source $f
        set +a
        echo ""
      fi
fi

# ^._.^
REQUIRED_PACKAGES="cat curl docker-compose jq ssh"

# Deployment addresses
#
# The chose of deployment app is following:
# - by default it will deploy worker app onto each server in the IP_LIST list
# - if the current address is in the COORDINATORS list, then it will deploy coordinator app instead
# - if the SINGLE_HOST param is set, then it will deploy both apps
#
# a list of machines to deploy to
IP_LIST=${IP_LIST:-}
# a list of machines mark some addresses to deploy only a coordinator there
COORDINATORS=${COORDINATORS:-}
# deploy everything on the same host
SINGLE_HOST=${SINGLE_HOST:-0}

# Digital Ocean operations
#DO_TOKEN
DO_ADDRESS_LIST=${DO_ADDRESS_LIST:-}
DO_API_ENDPOINT=${DO_API_ENDPOINT:-"https://api.digitalocean.com/v2/droplets?tag_name="}

LOCAL_WORK_DIR=${LOCAL_WORK_DIR:-"./.github/workflows/cd"}
REMOTE_WORK_DIR=${REMOTE_WORK_DIR:-"/cloud-game"}
DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG:-latest}
echo "Docker tag:$DOCKER_IMAGE_TAG"
# the total number of worker replicas to deploy
WORKERS=${WORKERS:-5}

echo "Starting deployment"

for pkg in $REQUIRED_PACKAGES; do
  which $pkg > /dev/null 2>&1
  if [ ! $? == 0 ]; then
    echo "Required package: $pkg is not installed"
    exit;
  fi
done

if [[ ! -z "${DO_TOKEN}" ]]; then
  for tag in $DO_ADDRESS_LIST; do
    echo "$tag processing..."
    call=$(curl -Ss -X GET -H "Content-Type: application/json" -H "Authorization: Bearer $DO_TOKEN" $DO_API_ENDPOINT$tag)
    res=$?
    if test "$res" == "0"; then
      IP_LIST+=$(echo "$call" | jq -r -j \
        ".droplets[] | .networks.v4[] | select(.type | contains(\"public\")).ip_address, \" \"")
    else
      echo "curl failed with the code [$res]"
    fi
  done
fi
echo "IPs:" $IP_LIST

# Run command builder
#
# By default it will run docker-compose with both coordinator and worker apps.
# With the SPLIT_HOSTS parameter specified, it will run either coordinator app
# if the current server address is found in the IP_LIST variable, otherwise it
# will run just the worker app.
#
# flags
deploy_coordinator=1
deploy_worker=1
# build run command
cmd="ZONE=\$zone docker-compose up -d --remove-orphans --scale worker=\${workers:-$WORKERS}"
if [ ! -z "$SPLIT_HOSTS" ]; then
  cmd+=" worker"
  deploy_coordinator=0
  deploy_worker=1
fi

for ip in $IP_LIST; do
  echo "Processing "$ip
  if ! ssh-keygen -q -F $ip &>/dev/null; then
    echo "Adding new host to the known_hosts file"
    ssh-keyscan $ip >> ~/.ssh/known_hosts
  fi

  # override run command
  if [ ! -z "$SPLIT_HOSTS" ]; then
    for addr in $COORDINATORS; do
       if [ "$ip" == $addr ]; then
         cmd="docker-compose up -d --remove-orphans coordinator"
         deploy_coordinator=1
         deploy_worker=0
         break
       fi
     done
  fi

  run="'#!/bin/bash
  ufw disable 2> /dev/null;
  source /etc/profile;
  export IMAGE_TAG=$DOCKER_IMAGE_TAG;
  export APP_DIR=$REMOTE_WORK_DIR;
  $cmd'"
  compose_src=$(cat $LOCAL_WORK_DIR/docker-compose.yml)

  # build Docker container env file
  run_env=""
  if [[ ! -z "${ENV_DIR}" ]]; then
    if [ $deploy_coordinator == 1 ]; then
      env_f=$ENV_DIR/coordinator.env
      if [[ -e "$env_f" ]]; then
        echo "Merge coordinator .env -> run.env"
        run_env+=$(cat $env_f)$'\n'
      fi
    fi
    if [ $deploy_worker == 1 ]; then
      env_f=$ENV_DIR/worker.env
      if [[ -e "$env_f" ]]; then
        echo "Merge worker .env -> run.env"
        run_env+=$(cat $env_f)
      fi
    fi
  fi
  echo $'\n'"run.env:"$'\n'"$run_env"$'\n'

  # optional ssh key param
  ssh_i=""
  if [[ ! -z "${SSH_KEY}" ]]; then
    ssh_i="-i ${SSH_KEY}"
  fi

  ssh ubuntu@$ip ${ssh_i:-} "\
    mkdir -p $REMOTE_WORK_DIR; \
    cd $REMOTE_WORK_DIR; \
    echo '$compose_src' > ./docker-compose.yml; \
    echo '$run_env' > ./run.env; \
    docker-compose pull; \
    echo $run > ./run.sh; \
    chmod +x ./run.sh; \
    ./run.sh"
done
