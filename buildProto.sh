#!/bin/bash
protoc --go_out=./services --go-grpc_out=./services protobufs/UserServiceSchema/userService.proto