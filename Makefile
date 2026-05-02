.PHONY: proto

proto_go_opts := paths=import,module=github.com/shadiestgoat/bankDataDB:.

proto:
	protoc \
		-I proto \
		-I `go list -m -f '{{.Dir}}' 'github.com/alta/protopatch'` \
		-I `go list -m -f '{{.Dir}}' 'google.golang.org/protobuf'` \
		--go-patch_out=plugin=go,${proto_go_opts} \
		--go-patch_out=plugin=go-grpc,${proto_go_opts} \
		bank_data.proto proto/bank_data/*.proto
