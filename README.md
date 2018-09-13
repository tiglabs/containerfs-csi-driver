[![Build Status](https://travis-ci.org/kubernetes-csi/drivers.svg?branch=master)](https://travis-ci.org/kubernetes-csi/drivers)
# CSI CFS Driver for K8S

## Kubernetes
### Requirements

The folllowing feature gates and runtime config have to be enabled to deploy the driver

```
FEATURE_GATES=CSIPersistentVolume=true,MountPropagation=true
RUNTIME_CONFIG="storage.k8s.io/v1alpha1=true"
```

Mountprogpation requries support for privileged containers. So, make sure privileged containers are enabled in the cluster.

### Get csi sidecar images

```
docker pull quay.io/k8scsi/csi-attacher:v0.2.0
docker pull quay.io/k8scsi/driver-registrar:v0.2.0
docker pull quay.io/k8scsi/csi-provisioner:v0.2.0
```

### Build cfscsi driver image

```docker build -t cfscsi:v1 deploy/.```

### Create configmap for csi driver

```kubectl create configmap kubecfg --from-file=deploy/kubernetes/kubecfg```

### Deploy cfs csi driver

```kubectl create -f deploy/kubernetes/cfs.yaml```

### Pre Volume: you must know volumeName first, example Nginx application

Please update the cfs Master Hosts & volumeName information in nginx-pre.yaml file.

```
docker pull nginx
kubectl create -f deploy/examples/nginx-pre.yaml
```

### Dynamic volume: Example Nginx application

```
docker pull nginx
kubectl create -f deploy/examples/cfs-pvc.yaml
kubectl create -f deploy/examples/cfs-pv.yaml
kubectl create -f deploy/examples/nginx-dynamic.yaml
```

## Other sample drivers
Please read [Drivers](https://kubernetes-csi.github.io/docs/Drivers.html) for more information

## Adding new sample drivers
Please, DO NOT submit PRs to add new drivers here unless they are just examples. Real CSI drivers are to be housed on their own repo separate from this one. You are then welcomed to send a PR to https://github.com/kubernetes-csi/docs to add the [Driver](https://github.com/kubernetes-csi/docs/wiki/Drivers) page.
