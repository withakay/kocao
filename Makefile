SHELL := /bin/bash

LOCALBIN ?= $(CURDIR)/.local/bin
KIND ?= $(LOCALBIN)/kind

KIND_CLUSTER_NAME ?= kocao-dev

API_IMAGE ?= kocao/control-plane-api
OPERATOR_IMAGE ?= kocao/control-plane-operator
IMAGE_TAG ?= dev

.PHONY: help
help:
	@printf "%s\n" "Targets:" \
		"  bootstrap           Install tools + download deps" \
		"  tools               Install kind locally" \
		"  test                Run Go tests" \
		"  lint                gofmt (check) + go vet" \
		"  build               Build Go binaries" \
		"  kind-up             Create local kind cluster" \
		"  kind-down           Delete local kind cluster" \
		"  images              Build local Docker images" \
		"  kind-load-images    Load images into kind" \
		"  deploy              Apply kustomize overlay" \
		"  undeploy            Delete kustomize overlay" \
		"  web-install         Install web deps (pnpm)" \
		"  web-dev             Run web dev server"

.PHONY: bootstrap
bootstrap: tools
	go mod download

.PHONY: tools
tools:
	@mkdir -p "$(LOCALBIN)"
	@if [ -x "$(KIND)" ]; then \
		echo "kind already installed: $(KIND)"; \
	else \
		echo "installing kind to $(KIND)"; \
		GOBIN="$(LOCALBIN)" go install sigs.k8s.io/kind@v0.27.0; \
	fi

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	@bad=$$(gofmt -l . | wc -l | tr -d ' '); \
	if [ "$$bad" != "0" ]; then \
		echo "gofmt required (run: gofmt -w .)"; \
		gofmt -l .; \
		exit 1; \
	fi
	go vet ./...

.PHONY: build
build:
	@mkdir -p bin
	go build -o bin/control-plane-api ./cmd/control-plane-api
	go build -o bin/control-plane-operator ./cmd/control-plane-operator

.PHONY: kind-up
kind-up: tools
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/up.sh

.PHONY: kind-down
kind-down: tools
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/down.sh

.PHONY: images
images:
	docker build -f build/Dockerfile.api -t "$(API_IMAGE):$(IMAGE_TAG)" .
	docker build -f build/Dockerfile.operator -t "$(OPERATOR_IMAGE):$(IMAGE_TAG)" .

.PHONY: kind-load-images
kind-load-images: tools
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/load-image.sh "$(API_IMAGE):$(IMAGE_TAG)"
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/load-image.sh "$(OPERATOR_IMAGE):$(IMAGE_TAG)"

.PHONY: deploy
deploy:
	kubectl apply -k deploy/overlays/dev-kind

.PHONY: undeploy
undeploy:
	kubectl delete -k deploy/overlays/dev-kind

.PHONY: web-install
web-install:
	cd web && pnpm install

.PHONY: web-dev
web-dev:
	cd web && pnpm dev
