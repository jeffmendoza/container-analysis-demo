# Container Analysis Demo

This demo configures your Kubernetes cluster to only allow containers without
known security vulnerabilities to be run. An [External Admission
Webhook](https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks)
is created that validates container images against Google's Container Analysis
service. The webhook uses the [Grafeas](https://grafeas.io/) artifact metadata
API to communicate with the Container Analysis service.

### Prerequisites
Enable [Vulnerability
Scanning](https://cloud.google.com/container-registry/docs/vulnerability-scanning)
in your Google Cloud project.

Container Analysis runs on images in [Container
Registry](https://cloud.google.com/container-registry/docs/), therefore this
demo will restrict your cluster to only run images from your own Container
Registry.

The webhook uses [Application Default
Credentials](https://developers.google.com/identity/protocols/application-default-credentials)
to authenticate with Container Registry and Analysis. Therefore it must run on a
Google Kubernetes Engine cluster with the approprate scopes (instructions
below). To run the demo off of Google Cloud Platform, a service account must be
used to access Container Registry and Analysis. See:
* https://godoc.org/golang.org/x/oauth2/google
* https://cloud.google.com/iam/docs/how-to

A Kubernetes 1.8+ cluster is required with support for the external admission
webhooks **alpha** feature enabled.

## Tutorial

### Infrastructure

Use the gcloud command to create a [Google Kubernetes
Engine](https://cloud.google.com/container-engine/) 1.8 cluster:

```
gcloud alpha container clusters create grafeas \
  --enable-kubernetes-alpha \
  --scopes https://www.googleapis.com/auth/cloud-platform \
  --cluster-version 1.8.4-gke.0
```

### Deploy the Image Signature Webhook

Create the `tls-container-analysis-webhook` secret and store the TLS certs:

```
kubectl create secret tls tls-container-analysis-webhook \
  --key pki/container-analysis-webhook-key.pem \
  --cert pki/container-analysis-webhook.pem
```

Create the `container-analysis-webhook` deployment:

```
kubectl apply -f kubernetes/container-analysis-webhook.yaml
```

Create the `container-analysis-webook`
[ExternalAdmissionHookConfiguration](https://kubernetes.io/docs/admin/extensible-admission-controllers/#how-are-external-admission-webhooks-triggered):

```
kubectl apply -f kubernetes/admission-hook-configuration.yaml
```

> After you create the external admission hook configuration, the system will take a few seconds to honor the new configuration.

### Testing the Admission Webhook

Pull down some images and push them to your registry to be scanned. Scanning
will happen automatically if enabled.

```bash
GCLOUD_PROJECT=<my-project>

docker pull hello-world:latest
docker tag hello-world:latest gcr.io/${GCLOUD_PROJECT}/hello-world:latest
gcloud docker -- push gcr.io/${GCLOUD_PROJECT}/hello-world:latest

docker pull nginx:latest
docker tag nginx:latest gcr.io/${GCLOUD_PROJECT}/nginx:latest
gcloud docker -- push gcr.io/${GCLOUD_PROJECT}/nginx:latest

docker pull alpine:latest
docker tag alpine:latest gcr.io/${GCLOUD_PROJECT}/alpine:latest
gcloud docker -- push gcr.io/${GCLOUD_PROJECT}/alpine:latest
```

Now run the images
```bash

kubectl run hello-world --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/hello-world:latest

kubectl run nginx --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/nginx:latest

kubectl run alpine --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/alpine:latest

kubectl logs $(kubectl get pods -l app=container-analysis-webhook -o jsonpath='{.items[0].metadata.name}')
```
Example:
```console
$ kubectl run hello-world --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/hello-world:latest
pod "hello-world" created
$ kubectl run nginx --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/nginx:latest
The  "" is invalid: : Found 12 occurrences with severity >= HIGH for image https://gcr.io/my-project/nginx@sha256:c464aed91b19addf623eb9457910dd8f0200dcd1cc68efed3a55a667f9f1e0a7
$ kubectl run alpine --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/alpine:latest
pod "alpine" created
$ kubectl logs $(kubectl get pods -l app=container-analysis-webhook -o jsonpath='{.items[0].metadata.name}')
2017/12/04 19:23:13 No vulns found for image: gcr.io/my-project/hello-world:latest
2017/12/04 19:23:15 Found 12 occurrences with severity >= HIGH for image https://gcr.io/my-project/nginx@sha256:c464aed91b19addf623eb9457910dd8f0200dcd1cc68efed3a55a667f9f1e0a7
2017/12/04 19:23:18 No vulns found for image: gcr.io/my-project/alpine:latest
$ 
```

## Cleanup

Run the following commands to remove the Kubernetes resources created during
this tutorial:

```
kubectl delete deployments container-analysis-webhook
kubectl delete pods hello-world nginx alpine
kubectl delete svc container-analysis-webhook
kubectl delete secrets tls-container-analysis-webhook
kubectl delete externaladmissionhookconfiguration container-analysis-webhook
```
