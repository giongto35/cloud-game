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

# Import optional script.env file.
# This file contains script runtime params.
if [[ ! -z "${ENV_DIR}" ]]; then
  f="$ENV_DIR/script.env"
  if [[ -e "$f" ]]
      then
        echo "Setting script .env"
        set -a
        source $f
        set +a
      fi
fi

# ^._.^
REQUIRED_PACKAGES="cat curl jq ssh"

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

# flags
deploy_coordinator=1
deploy_worker=1

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

for ip in $IP_LIST; do
  echo $ip
  ssh-keyscan -H $ip >> ~/.ssh/known_hosts
  sleep 2

  cmd="ZONE=\$zone docker-compose up -d --remove-orphans --scale worker=\${workers:-$WORKERS}"

  if [ ! $SINGLE_HOST == 1 ]; then
    cmd+=" worker"
    deploy_coordinator=0
    deploy_worker=1
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

  # copy Docker env files if the ENV_DIR is set
  coordinator_env_file=""
  worker_env_file=""
  if [[ ! -z "${ENV_DIR}" ]]; then
    if [ $deploy_coordinator == 1 ]; then
      echo "Copy coordinator .env"
      coordinator_env_file=$(cat $ENV_DIR/coordinator.env)
    fi
        if [ $deploy_worker == 1 ]; then
          echo "Copy worker .env"
          worker_env_file=$(cat $ENV_DIR/worker.env)
        fi
  fi

  # optional ssh key param
  ssh_i=""
  if [[ ! -z "${SSH_KEY}" ]]; then
    ssh_i="-i ${SSH_KEY}"
  fi

  ssh ubuntu@$ip ${ssh_i:-} "\
    mkdir -p $REMOTE_WORK_DIR; \
    cd $REMOTE_WORK_DIR; \
    echo '$compose_src' > ./docker-compose.yml; \
    echo '$coordinator_env_file' > ./coordinator.env; \
    echo '$worker_env_file' > ./worker.env; \
    docker system prune -f; \
    docker-compose pull; \
    echo $run > ./run.sh; \
    chmod +x ./run.sh; \
    ./run.sh"
done

echo 'done'
