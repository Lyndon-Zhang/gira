
all: gen

gen:
	protoc --go_out=gen/service --go-grpc_out=gen/service --go-gclient_out=gen/service doc/service/hall.proto

.PHONY: gen 
