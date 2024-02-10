all: k8s.local.yaml k8s.production.yaml

k8s.local.yaml: generate
	./generate --env=local > $@

k8s.production.yaml: generate
	./generate --env=production > $@

generate: main.go
	go build -o generate main.go
