source $HOME/code/home-server/.env

docker compose -f $HOME/code/home-server/apps/qbit/docker-compose.yml up --force-recreate -d
