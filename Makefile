
all: cli service framework

framework:
	make -C framework/smallgame 

cli:
	go build -o gira bin/cli/main.go
	cp -rf gira $$(go env GOPATH)/bin/gira
	go build -o protoc-gen-go-gclient bin/gen_gclient/*.go
	cp -rf protoc-gen-go-gclient $$(go env GOPATH)/bin/protoc-gen-go-gclient
	go build -o protoc-gen-go-gserver bin/gen_gserver/*.go
	cp -rf protoc-gen-go-gserver $$(go env GOPATH)/bin/protoc-gen-go-gserver

service:
	protoc --go_out=service/admin --go-grpc_out=service/admin --go-gclient_out=service/admin --go-gserver_out=service/admin service/admin/admin.proto
	protoc --go_out=service/peer --go-grpc_out=service/peer --go-gclient_out=service/peer service/peer/peer.proto
	protoc --go-gclient_out=proto_and_grpc_pacakge=google.golang.org/grpc/channelz/grpc_channelz_v1:service/channelz service/channelz/channelz.proto


.PHONY: gen cli service framework
