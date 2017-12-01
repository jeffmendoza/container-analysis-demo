# Container Analysis Demo



## Tutorial

### Infrastructure

A Kubernetes 1.8+ cluster is required with support for the [external admission
webhooks](https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks)
alpha feature enabled.

If you have access to [Google Container
Engine](https://cloud.google.com/container-engine/) use the gcloud command to
create a 1.8 Kubernetes cluster:

```
gcloud alpha container clusters create grafeas \
  --enable-kubernetes-alpha \
  --scopes https://www.googleapis.com/auth/cloud-platform \
  --cluster-version 1.8.4-gke.0
```

> Any Kubernetes 1.8 cluster with support for external admission webhooks will work. 

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

Create the `container-analysis-webook` [ExternalAdmissionHookConfiguration](https://kubernetes.io/docs/admin/extensible-admission-controllers/#how-are-external-admission-webhooks-triggered):

```
kubectl apply -f kubernetes/admission-hook-configuration.yaml
```

> After you create the external admission hook configuration, the system will take a few seconds to honor the new configuration.

### Testing the Admission Webhook

```bash
GCLOUD_PROJECT=<my-project>

docker pull hello-world:latest
docker tag hello-world:latest gcr.io/${GCLOUD_PROJECT}/hello-world:latest
gcloud docker -- push gcr.io/${GCLOUD_PROJECT}/hello-world:latest

# Wait for scan
kubectl run hello-world --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/hello-world:latest
kubectl logs $(kubectl get pods -l app=container-analysis-webhook -o jsonpath='{.items[0].metadata.name}')
```

```shell
$ GCLOUD_PROJECT=<my-project>

$ docker pull hello-world:latest
latest: Pulling from library/hello-world
[0B
[1BDigest: sha256:be0cd392e45be79ffeffa6b05338b98ebb16c87b255f48e297ec7f98e123905c
Status: Downloaded newer image for hello-world:latest
$ docker tag hello-world:latest gcr.io/${GCLOUD_PROJECT}/hello-world:latest
$ gcloud docker -- push gcr.io/${GCLOUD_PROJECT}/hello-world:latest
The push refers to a repository [gcr.io/my-project/hello-world]

[1Blatest: digest: sha256:0b1396cdcea05f91f38fc7f5aecd58ccf19fb5743bbb79cff5eb3c747b36d909 size: 524
$ 
$ kubectl run hello-world --restart=Never --image=gcr.io/${GCLOUD_PROJECT}/hello-world:latest
pod "hello-world" created
$ 
$ kubectl logs $(kubectl get pods -l app=container-analysis-webhook -o jsonpath='{.items[0].metadata.name}')
2017/11/30 23:53:39 No vulns found for image: gcr.io/my-project/hello-world:latest
$ 
```



## Cleanup

Run the following commands to remove the Kubernetes resources created during this tutorial:

```
kubectl delete deployments container-analysis-webhook
kubectl delete pods hello-world
kubectl delete svc container-analysis-webhook
kubectl delete secrets tls-container-analysis-webhook
```
