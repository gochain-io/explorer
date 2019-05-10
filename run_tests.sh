cleanup_containers() {
    for imagename in test_mongo test_explorer_grabber test_explorer_server test_explorer_go_integration; do
        echo "* Trying to stop ${imagename}..."
        result="$(docker ps -a | grep $imagename || true)"
        if [ -n "$result" ]; then
            echo "Container $imagename exists, removing..."
            docker rm --force $imagename
        else
            echo "'$imagename' does not exist."
        fi
    done
}
set -e
trap cleanup_containers SIGINT EXIT SIGHUP
docker run --name test_mongo -d -p 8545:8545 -p 8080:8080 -p 27017:27017 mongo
varA=`docker ps --no-trunc -q | cut -c 1-12`
# build
docker build -t gochain/explorer:test_ci .
# launch required containers
docker run --name test_explorer_grabber -d --network="container:$varA" gochain/explorer:test_ci grabber -u https://testnet-rpc.gochain.io -s 10
docker run --name test_explorer_server -d --network="container:$varA" gochain/explorer:test_ci server -d /explorer/ -u https://testnet-rpc.gochain.io
# this will run both integration and unit tests, integration test will require mongo running that why it should be running in the same network with mongo
docker run --name test_explorer_go_integration -d -w /explorer --network="container:$varA" golang:alpine /bin/sh -c "while true; do sleep 15 ; done"
# looks like due to some limitations it's not possilbe to just map the current directory to a container, that why you have to use docker cp
docker cp . test_explorer_go_integration:/explorer
# Installing build deps for tests
docker exec test_explorer_go_integration apk add git gcc linux-headers g++ make
# run the tests inside running container
docker exec test_explorer_go_integration go test -tags=integration ./...
sleep 5 # let's wait until server start
# docker exec test_explorer npm test
echo "Docker logs for grabber"
docker logs test_explorer_grabber
echo "Docker logs for server"
docker logs test_explorer_server
echo "Trying curl"
docker run --rm --network="container:$varA" byrnedo/alpine-curl -f http://localhost:8080/
docker run --rm --network="container:$varA" byrnedo/alpine-curl -f http://localhost:8080/api/blocks/10
docker run --rm --network="container:$varA" byrnedo/alpine-curl -f http://localhost:8080/api/blocks/10/transactions
