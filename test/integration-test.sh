#!/usr/bin/env bash
set -ex

DIR=$PWD
docker build -t tape:latest .
network=fabric_test
export COMPOSE_PROJECT_NAME=fabric

case $1 in
 1_4)
    unset COMPOSE_PROJECT_NAME
    # sadly, bootstrap.sh from release-1.4 still pulls binaries from Nexus, which is not available anymore
    # Why comment following code? Please check this issue: https://github.com/Hyperledger-TWGC/tape/issues/159
    # curl -vsS https://raw.githubusercontent.com/hyperledger/fabric/release-2.2/scripts/bootstrap.sh | bash
    ./test/bootstraps/bootstrap-v2.2.sh
    cd ./fabric-samples/
    git checkout release-1.4
    cd ./first-network
    # 1.4.10
    echo y | ./byfn.sh up -i 1.4.10
    # comments here for 1.4.8 work around as docker image issue.
    # docker pull hyperledger/fabric-orderer:amd64-1.4  
    # Error response from daemon: manifest for hyperledger/fabric-orderer:amd64-1.4 not found: manifest unknown: manifest unknown
    cp -r crypto-config "$DIR"
    
    CONFIG_FILE=/config/test/config14org1andorg2.yaml
    network=host
    ;;
 2_2)
    curl -sSL https://bit.ly/2ysbOFE | bash -s -- 2.2.2 1.4.9
    cd ./fabric-samples/test-network
    echo y |  ./network.sh down -i 2.2
    echo y |  ./network.sh up createChannel -i 2.2
    cp -r organizations "$DIR"

    CONFIG_FILE=/config/test/config20org1andorg2.yaml

    if [ $2 == "ORLogic" ]; then
      CONFIG_FILE=/config/test/config20selectendorser.yaml
      ARGS=(-ccep "OR('Org1.member','Org2.member')")
    fi

    echo y |  ./network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go/ -ccl go "${ARGS[@]}"
    ;;
 2_3)
    # Why comment following code? Please check this issue: https://github.com/Hyperledger-TWGC/tape/issues/159
    # curl -vsS https://raw.githubusercontent.com/hyperledger/fabric/release-2.3/scripts/bootstrap.sh | bash
    ./test/bootstraps/bootstrap-v2.3.sh
    cd ./fabric-samples/test-network
    echo y |  ./network.sh down
    echo y |  ./network.sh up createChannel
    cp -r organizations "$DIR"

    CONFIG_FILE=/config/test/config20org1andorg2.yaml

    if [ $2 == "ORLogic" ]; then
      CONFIG_FILE=/config/test/config20selectendorser.yaml
      ARGS=(-ccep "OR('Org1.member','Org2.member')")
    fi

    echo y |  ./network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go/ -ccl go "${ARGS[@]}"
    ;;
 latest)
    curl -vsS https://raw.githubusercontent.com/hyperledger/fabric/master/scripts/bootstrap.sh | bash
    cd ./fabric-samples/test-network
    echo y |  ./network.sh down
    echo y |  ./network.sh up createChannel
    cp -r organizations "$DIR"

    CONFIG_FILE=/config/test/config20org1andorg2.yaml

    if [ $2 == "ORLogic" ]; then
      CONFIG_FILE=/config/test/config20selectendorser.yaml
      ARGS=(-ccep "OR('Org1.member','Org2.member')")
    fi

    echo y |  ./network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go/ -ccl go "${ARGS[@]}"
    ;;
 *)
    echo "Usage: $1 [1_4|2_2|2_3|latest]"
    echo "When given version, start byfn or test network basing on specific version of docker image"
    echo "For any value without mock, 1_4, 2_2, 2_3, latest will show this hint"
    exit 0
    ;;
esac

cd "$DIR"
docker ps -a
docker network ls
docker run  -e TAPE_LOGLEVEL=debug --network $network -v $PWD:/config tape tape -c $CONFIG_FILE -n 6974
