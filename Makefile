SHELL := /bin/bash

LOCALBIN ?= $(CURDIR)/.local/bin
KIND ?= $(LOCALBIN)/kind

KIND_CLUSTER_NAME ?= kocao-dev
K8S_NAMESPACE ?= kocao-system

API_IMAGE ?= kocao/control-plane-api
OPERATOR_IMAGE ?= kocao/control-plane-operator
HARNESS_IMAGE ?= kocao/harness-runtime
SIDECAR_IMAGE ?= kocao/kocao-sidecar
WEB_IMAGE ?= kocao/control-plane-web
IMAGE_TAG ?= dev

.PHONY: help
help:
	@printf "%s\n" "Targets:" \
		"  bootstrap           Install tools + download deps" \
		"  tools               Install kind locally" \
		"  ci                  Run lint + tests" \
		"  test                Run Go tests" \
		"  lint                gofmt (check) + go vet" \
		"  build               Build Go binaries" \
		"  build-cli           Build kocao CLI binary" \
		"  kind-up             Create local kind cluster" \
		"  kind-down           Delete local kind cluster" \
		"  images              Build local Docker images" \
		"  harness-images      Build all harness image profiles" \
		"  kind-load-images    Load images into kind" \
		"  kind-prepull-harness-profiles Build and load harness profiles into kind" \
		"  registry-prepull-harness-profiles Pre-pull registry-backed harness profiles on any dev cluster" \
		"  microk8s-prepull-harness-profiles Pre-pull registry-backed harness profiles on MicroK8s" \
		"  seed-agent-secrets  Copy local OAuth auth into k8s secret" \
		"  deploy              Apply kustomize overlay" \
		"  deploy-restart      Apply overlay + restart control-plane pods" \
		"  deploy-wait         Wait for control-plane rollout" \
		"  kind-refresh        Build + load + deploy-restart for kind" \
		"  images-live-agent   Build API/operator/web/harness/sidecar images" \
		"  kind-load-images-live-agent Load API/operator/web/harness/sidecar images into kind" \
		"  test-agent-live-kind Run live agent lifecycle verification in kind" \
		"  undeploy            Delete kustomize overlay" \
		"  harness-smoke       Build + smoke-test harness image" \
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

.PHONY: ci
ci: lint test

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
	go build -o bin/kocao ./cmd/kocao

.PHONY: build-cli
build-cli:
	@mkdir -p bin
	go build -o bin/kocao ./cmd/kocao

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
	docker build -f build/Dockerfile.web -t "$(WEB_IMAGE):$(IMAGE_TAG)" .
	docker build -f build/Dockerfile.harness --target harness-profile-full -t "$(HARNESS_IMAGE):$(IMAGE_TAG)" .
	docker build -f build/Dockerfile.sidecar -t "$(SIDECAR_IMAGE):$(IMAGE_TAG)" .

.PHONY: harness-images
harness-images:
	bash ./build/harness/build-profiles.sh

.PHONY: images-live-agent
images-live-agent: images

.PHONY: kind-load-images
kind-load-images: tools
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/load-image.sh "$(API_IMAGE):$(IMAGE_TAG)"
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/load-image.sh "$(OPERATOR_IMAGE):$(IMAGE_TAG)"
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/load-image.sh "$(WEB_IMAGE):$(IMAGE_TAG)"
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/load-image.sh "$(HARNESS_IMAGE):$(IMAGE_TAG)"
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/kind/load-image.sh "$(SIDECAR_IMAGE):$(IMAGE_TAG)"

.PHONY: kind-load-images-live-agent
kind-load-images-live-agent: kind-load-images

.PHONY: kind-prepull-harness-profiles
kind-prepull-harness-profiles: tools harness-images
	KIND_CLUSTER_NAME="$(KIND_CLUSTER_NAME)" KIND_BIN="$(KIND)" bash ./hack/dev/prepull-harness-images.sh kind

.PHONY: registry-prepull-harness-profiles
registry-prepull-harness-profiles:
	bash ./hack/dev/prepull-harness-images.sh registry

.PHONY: microk8s-prepull-harness-profiles
microk8s-prepull-harness-profiles:
	bash ./hack/dev/prepull-harness-images.sh microk8s

.PHONY: harness-smoke
harness-smoke: harness-images
	bash ./build/harness/smoke-profiles.sh

.PHONY: seed-agent-secrets
seed-agent-secrets:
	bash ./hack/seed-agent-secrets.sh

.PHONY: deploy
deploy:
	kubectl apply -k deploy/overlays/dev-kind

.PHONY: deploy-wait
deploy-wait:
	kubectl -n "$(K8S_NAMESPACE)" rollout status deploy/control-plane-api --timeout=300s
	kubectl -n "$(K8S_NAMESPACE)" rollout status deploy/control-plane-operator --timeout=300s

.PHONY: deploy-restart
deploy-restart: deploy
	kubectl -n "$(K8S_NAMESPACE)" rollout restart deploy/control-plane-api deploy/control-plane-operator
	$(MAKE) deploy-wait

.PHONY: kind-refresh
kind-refresh: images kind-load-images deploy-restart

.PHONY: undeploy
undeploy:
	kubectl delete -k deploy/overlays/dev-kind

.PHONY: test-agent-live-kind
test-agent-live-kind:
	bash ./test/live/agent_session_lifecycle_kind.sh

.PHONY: web-install
web-install:
	cd web && pnpm install

.PHONY: web-dev
web-dev:
	cd web && pnpm dev
