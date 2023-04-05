KUBECONFIG=$(HOME)/.kube/dev
tag=dev
image=paskalmaksim/pod-admission-controller:$(tag)
config=config.yaml
testnamespace=test-pod-admission-controller

test:
	./scripts/validate-license.sh
	go mod tidy
	go fmt ./cmd/... ./pkg/... ./internal/...
	go vet ./cmd/... ./pkg/... ./internal/...
	go test --race -coverprofile coverage.out ./cmd/... ./pkg/...
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run -v

coverage:
	go tool cover -html=coverage.out

.PHONY: e2e
e2e:
	make deploy config=./e2e/testdata/config.yaml
	kubectl delete ns $(testnamespace) || true
	kubectl create ns $(testnamespace)
	kubectl label ns $(testnamespace) environment=dev
	kubectl -n $(testnamespace) apply -f ./e2e/testdata/pods
	kubectl -n $(testnamespace) wait --for=condition=Ready=true pods -lapp=test-pod-admission-controller --timeout=600s
	go test --race ./e2e -kubeconfig=$(KUBECONFIG)
	kubectl delete ns $(testnamespace)

testChart:
	ct lint --charts ./charts/pod-admission-controller

run:
	go run --race ./cmd \
	-log.level=debug \
	-log.pretty \
	-kubeconfig=$(KUBECONFIG) \
	-config=$(config) \
	-cert=./certs/server.crt \
	-key=./certs/server.key \
	-listen=127.0.0.1:8443 \
	-metrics.listen=127.0.0.1:31080

sslInit:
	rm -rf ./certs
	mkdir -p ./certs
	go run github.com/maksim-paskal/envoy-control-plane/cmd/gencerts@latest \
	-cert.path=./certs \
	-dns.names=pod-admission-controller.pod-admission-controller.svc,\
	pod-admission-controller.pod-admission-controller.svc.cluster.local

build:
	git tag -d `git tag -l "helm-chart-*"`
	go run github.com/goreleaser/goreleaser@latest build --rm-dist --snapshot --skip-validate
	mv ./dist/pod-admission-controller_linux_amd64_v1/pod-admission-controller pod-admission-controller
	docker build --pull . -t $(image)

push:
	docker push $(image)

restart:
	kubectl -n pod-admission-controller rollout restart deploy pod-admission-controller

deploy:
	kubectl -n pod-admission-controller scale deploy --all --replicas=0 || true
	
	helm upgrade pod-admission-controller \
	--install \
	--namespace pod-admission-controller \
	--create-namespace \
	./charts/pod-admission-controller \
	--set registry.image=paskalmaksim/pod-admission-controller:dev \
	--set registry.imagePullPolicy=Always \
	--set-file config=$(config)
	kubectl -n pod-admission-controller wait pod --for=condition=ready --all --timeout=600s

clean:
	helm uninstall pod-admission-controller --namespace pod-admission-controller || true
	kubectl delete ns pod-admission-controller
	kubectl delete ns $(testnamespace) || true

chart-release:
	./scripts/chart-release.sh