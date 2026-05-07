```bash
export KUBECONFIG=~/.kube/prod

# lint images in your cluster
k8s-images-cli > ./utils/create-sentry-projects/prod-images.txt

# create projects and print cache settings
go run ./utils/create-sentry-projects -config=./pod-admission-controller/config.yaml
```