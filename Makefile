BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
UNIX_DATE := $(shell date -u +"%s")
VCS_REF := $(shell git rev-parse HEAD)

build-local:
	@go build -o ./bin/fsbench-server ./cmd/server
	@go build -o ./bin/fsbench-worker ./cmd/worker
	@go build -o ./bin/compare ./cmd/compare

build:
	docker pull reg.deeproute.ai/deeproute-public/go/golang:alpine
	docker build --tag reg.deeproute.ai/deeproute-public/tools/fsbench-server:$(VCS_REF) --build-arg "TYPE=server" --build-arg "BUILD_DATE=$(BUILD_DATE)" --build-arg "VCS_REF=$(VCS_REF)" .
	docker build --tag reg.deeproute.ai/deeproute-public/tools/fsbench-worker:$(VCS_REF) --build-arg "TYPE=worker" --build-arg "BUILD_DATE=$(BUILD_DATE)" --build-arg "VCS_REF=$(VCS_REF)" .

debug-server:
	docker run --rm --name=fsbench-server -it reg.deeproute.ai/deeproute-public/tools/fsbench-server:$(VCS_REF) sh

debug-worker:
	docker run --rm --name=fsbench-worker -it reg.deeproute.ai/deeproute-public/tools/fsbench-worker:$(VCS_REF) sh

release:
	docker tag reg.deeproute.ai/deeproute-public/tools/fsbench-server:$(VCS_REF) reg.deeproute.ai/deeproute-public/tools/fsbench-server:latest
	docker tag reg.deeproute.ai/deeproute-public/tools/fsbench-worker:$(VCS_REF) reg.deeproute.ai/deeproute-public/tools/fsbench-worker:latest
	docker push reg.deeproute.ai/deeproute-public/tools/fsbench-server:latest
	docker push reg.deeproute.ai/deeproute-public/tools/fsbench-worker:latest

push-dev:
	docker build --tag reg.deeproute.ai/deeproute-public/tools/fsbench-server:$(UNIX_DATE) --build-arg "TYPE=server" --build-arg "BUILD_DATE=$(BUILD_DATE)" --build-arg "VCS_REF=$(VCS_REF)" .
	docker build --tag reg.deeproute.ai/deeproute-public/tools/fsbench-worker:$(UNIX_DATE) --build-arg "TYPE=worker" --build-arg "BUILD_DATE=$(BUILD_DATE)" --build-arg "VCS_REF=$(VCS_REF)" .
	docker push reg.deeproute.ai/deeproute-public/tools/fsbench-server:$(UNIX_DATE)
	docker push reg.deeproute.ai/deeproute-public/tools/fsbench-worker:$(UNIX_DATE)

test:
	go test -v `go list ./...`
