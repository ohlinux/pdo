#!/bin/bash

CURRENT_DIR=$(pwd)
BUILD_DIR=_output
BUILD_PREFIX=pdo
GOPATH=${CURRENT_DIR}/${BUILD_DIR}
DESTINATION_DIR=${GOPATH}/src/${BUILD_PREFIX}

echo $DESTINATION_DIR

export GOPATH=$GOPATH
mkdir -p ${DESTINATION_DIR}

echo "sync ${BUILD_PREFIX} to ${DESTINATION_DIR}"

rsync -a ./src/ --exclude=bin --exclude=${BUILD_DIR}  --exclude=hack  ${DESTINATION_DIR}/
rsync -a ./vendor --exclude=bin --exclude=${BUILD_DIR}  --exclude=hack  ${DESTINATION_DIR}/

cd ${DESTINATION_DIR}

echo "ready to build ${BUILD_PREFIX} in ${DESTINATION_DIR}"
go build -tags vfs "$@"

rsync -a ${BUILD_PREFIX} ${CURRENT_DIR}/bin/

echo "new ${BUILD_PREFIX} in ${CURRENT_DIR}/bin"
ls -l ${CURRENT_DIR}/bin/

