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



## Cleanup

Run the following commands to remove the Kubernetes resources created during this tutorial:

```
kubectl delete deployments grafeas container-analysis-webhook
kubectl delete pods echod
kubectl delete svc grafeas container-analysis-webhook
kubectl delete secrets tls-container-analysis-webhook
kubectl delete configmap container-analysis-webhook
```
