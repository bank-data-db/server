.PHONY: proto

proto_go_opts := paths=import,module=github.com/bank-data-db/server:.

proto:
	protoc \
		-I proto \
		-I `go list -m -f '{{.Dir}}' 'github.com/alta/protopatch'` \
		-I `go list -m -f '{{.Dir}}' 'google.golang.org/protobuf'` \
		--go-patch_out=plugin=go,${proto_go_opts} \
		--go-patch_out=plugin=go-grpc,${proto_go_opts} \
		user.proto errors.proto bank_data.proto proto/bank_data/*.proto

test:
	docker compose -f ./docker-compose.yml run --build --remove-orphans --rm test-runner
