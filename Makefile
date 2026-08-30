.PHONY: proto test

proto:
	go get -u github.com/bank-data-db/proto

test:
	docker compose -f ./docker-compose.yml run --build --remove-orphans --rm test-runner
