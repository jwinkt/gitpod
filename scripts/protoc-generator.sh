#!/bin/bash
# Copyright (c) 2022 Gitpod GmbH. All rights reserved.
# Licensed under the GNU Affero General Public License (AGPL).
# See License-AGPL.txt in the project root for license information.


install_dependencies() {
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0

    go get google.golang.org/protobuf/runtime/protoimpl@v1.28.0
    go get google.golang.org/protobuf/reflect/protoreflect@v1.28.0
	go get google.golang.org/protobuf/types/known/timestamppb@v1.28.0

    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0

    go install github.com/golang/mock/mockgen@v1.6.0

    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.10.0

    curl -sSo /tmp/protoc-gen-grpc-java https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.45.0/protoc-gen-grpc-java-1.45.0-linux-x86_64.exe
    chmod +x /tmp/protoc-gen-grpc-java
}

lint() {
    local PROTO_DIR=${1:-.}

    docker run --volume "$PWD/$PROTO_DIR:/workspace" --workdir /workspace bufbuild/buf lint || exit 1
}

go_protoc() {
    local ROOT_DIR=$1
    local PROTO_DIR=${2:-.}
    # shellcheck disable=2035
    protoc \
        -I /usr/lib/protoc/include -I"$ROOT_DIR" -I. \
        --go_out=go \
        --go_opt=paths=source_relative \
        --go-grpc_out=go \
        --go-grpc_opt=paths=source_relative \
        "${PROTO_DIR}"/*.proto
}

typescript_protoc() {
    local ROOT_DIR=$1
    local PROTO_DIR=${2:-.}
    local MODULE_DIR
    # Assigning external program output directly
    # after the `local` keyword masks the return value (Could be an error).
    # Should be done in a separate line.
    MODULE_DIR=$(pwd)

    pushd typescript > /dev/null || exit

    yarn install

    rm -rf "$MODULE_DIR"/typescript/src/*pb*.*

    echo "[protoc] Generating TypeScript files"
    protoc \
        --plugin=protoc-gen-grpc="$MODULE_DIR"/typescript/node_modules/.bin/grpc_tools_node_protoc_plugin \
        --js_out=import_style=commonjs,binary:src \
        --grpc_out=grpc_js:src \
        -I /usr/lib/protoc/include -I"$ROOT_DIR" -I.. -I"../$PROTO_DIR" \
        "../$PROTO_DIR"/*.proto

    protoc \
        --plugin=protoc-gen-ts="$MODULE_DIR"/typescript/node_modules/.bin/protoc-gen-ts \
        --ts_out=grpc_js:src \
        -I /usr/lib/protoc/include -I"$ROOT_DIR" -I.. -I"../$PROTO_DIR" \
        "../$PROTO_DIR"/*.proto

    # shellcheck disable=SC2011
    # ls -1 "$MODULE_DIR"/typescript/src/*_pb.d.ts | xargs sed -i -e "s/[[:space:]]*$//" || exit

    popd > /dev/null || exit
}

update_license() {
    leeway run components:update-license-header
}
