# GCP Night King

A service for resurrecting pre-empted GCE instances.

See also [a Python version](https://github.com/itamaro/gcp-night-king) of the same project.

## Overview

The Night King service is a service that restarts preempted GCE instances.

It uses Google Cloud Pub/Sub for reporting instance preemption.

When a machine is about to be preempted, if it should be restarted, it should publish a Pub/Sub message to a known topic (e.g. "night-king-preempt"):

```json
{
    "name": "<instance-name>",
    "zone": "<instance-zone>"
}
```

The Night King service subscribes to the Pub/Sub topic, and tries to restart instances accordingly, once they are terminated.

## Setting Up Pub/Sub

Create a Pub/Sub topic & subscription:

```sh
gcloud pubsub topics create night-king-preempt
gcloud pubsub subscriptions create night-king-preempt --topic night-king-preempt
```

## Configure Shutdown Script

To have preempted instances publish a message, use the included [shutdown script](https://cloud.google.com/compute/docs/shutdownscript) (or integrate it with an existing shutdown script):

```sh
gcloud compute instances create my-resurrectable-instance \
    --preemptible --metadata-from-file shutdown-script=zombie.sh [...]
```

Note: when providing explicit scopes, make sure to include the `https://www.googleapis.com/auth/pubsub` scope to allow the instance to publish messages to topics (it is included in the default scopes).

## Running The Night King service

The service is implemented in Go, and [prebuilt Docker images](https://hub.docker.com/r/itamarost/gcp-night-king/tags/) are provided.

Build & run it yourself:

```sh
go get github.com/itamaro/gcp-go-night-king
gcp-go-night-king -project PROJECT_ID -subscription-name SUBSCRIPTION_NAME
```

Get it from Docker Hub and use Docker to run it:

```sh
docker pull itamarost/gcp-night-king:v1-golang
docker run -d -v $HOME/.config/gcloud:/root/.config/gcloud itamarost/gcp-night-king:v1-golang \
    -project PROJECT_ID -subscription-name SUBSCRIPTION_NAME
```

In either case, you'll need to have [Google Cloud SDK authorization](https://cloud.google.com/sdk/docs/) set up for the service to be able to receive messages from Google Pub/Sub and use the GCE Compute API.

The Docker bind-mount is useful to share your host Google Cloud credentials - feel free to use other methods to obtain appropriate Google Cloud credentials inside Docker.

There are multiple ways to have the service running "in production" (e.g., not in the foreground of your dev-machine terminal).

You can use whatever method fits your environment (deployment-setup contributions are welcome). See below details for already-supprted methods.

## Deploying the Night King service to Kubernetes

If you have a Kubernetes cluster, it makes sense deploying this service to the cluster.

Using Helm (the service is available via the official stable Helm repository):

```sh
helm repo update
helm install --set projectID=YOUR_PROJECT_ID stable/gcp-night-king
```

Alternatively, use `kubectl` directly:

The following assumes you have [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/), and it is configured with the correct cluster.

First, create a config-map with your project ID:

```sh
kubectl create configmap nightking-config --from-literal=project.id=<YOUR-PROJECT-ID>
```

Now you can deploy the service by applying the deployment YAML:

```sh
kubectl apply -f k8s-deploy.yaml
```

Feel free to modify the deployment YAML to fit your needs (resources, Docker image, other flags, etc.).

## Development

This project is written in Go, and uses [Dep](https://github.com/golang/dep) for dependency management.

To hack on this project:

```sh
go get -d github.com/itamaro/gcp-go-night-king
cd $GOPATH/itamaro/gcp-go-night-king
dep ensure

# Build & run from source
go run main.go nightking.go -project PROJECT_ID -subscription-name SUBSCRIPTION_NAME

# Run unit tests
go test
```

Building the Docker image:

```sh
docker build -t itamarost/gcp-night-king:v1-golang .
```
