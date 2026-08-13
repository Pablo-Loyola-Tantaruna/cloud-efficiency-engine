APP_NAME=cloud-efficiency-engine
IMAGE=$(APP_NAME):dev
HELM_CHART=deployments/helm/cloud-efficiency-engine
HELM_RELEASE=cloud-efficiency-engine
K8S_NAMESPACE=cloud-efficiency-engine

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: build
build:
	go build -o bin/$(APP_NAME) ./cmd/api

.PHONY: run
run:
	go run ./cmd/api

.PHONY: format
format:
	gofmt -w .

.PHONY: check-format
check-format:
	@test -z "$$(gofmt -l .)" || \
		(echo "Files are not formatted:" && gofmt -l . && exit 1)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: verify
verify:
	go mod verify

.PHONY: check
check: format check-format tidy verify test vet build

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE) .

.PHONY: docker-run
docker-run:
	docker run --rm -p 8080:8080 $(IMAGE)

.PHONY: helm-lint
helm-lint:
	helm lint $(HELM_CHART)

.PHONY: helm-template
helm-template:
	helm template $(HELM_RELEASE) \
		$(HELM_CHART) \
		--namespace $(K8S_NAMESPACE)

.PHONY: helm-install
helm-install:
	helm upgrade --install $(HELM_RELEASE) \
		$(HELM_CHART) \
		--namespace $(K8S_NAMESPACE) \
		--create-namespace \
		--set image.repository=$(APP_NAME) \
		--set image.tag=dev \
		--set image.pullPolicy=IfNotPresent

.PHONY: kind-load
kind-load:
	kind load docker-image $(IMAGE)

.PHONY: k8s-status
k8s-status:
	kubectl get pods -n $(K8S_NAMESPACE)
	kubectl get svc -n $(K8S_NAMESPACE)
	kubectl get endpoints -n $(K8S_NAMESPACE)

.PHONY: clean
clean:
	rm -rf bin