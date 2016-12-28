zfs-flex-volume
===============

Simple ZFS [flexVolume](http://kubernetes.io/docs/user-guide/volumes/#flexvolume)
driver for Kubernetes.

Status
======

Highly experimental. Master is probably broken. Do not use on a system that you
care about the data.

Linux only.

[Kubernetes FlexVolumes](https://github.com/kubernetes/kubernetes/blob/master/examples/volumes/flexvolume/README.md)
are still considered _alpha_.

Description
============

FlexVolumes allow one to add support for additional storage backends to Kubernetes
without modifying the core Kubernetes code.

zfs-flex-volume provides a simplistic ZFS driver. It will allocate zfs filesystems
from a parent dataset and mount them for usage in a container.

Building
========
You need a go build environment. This has been tested with go 1.7.x - other versions
may work.

Inside the repo, run `./script/build` and you should have a Linux amd64 binary
named `zfs-flex-volume` in the root of the repo.

Usage
=====

Place the `zfs-flex-volume` binary on the target node(s).  You may want to place
this in your PATH to ease usage.  The node must have [ZFS](http://zfsonlinux.org/)
installed and configured.

Create a "parent" dataset. All volumes created by this driver will be children
of it. In my test set up, I have a zpool called `rpool` and I created a parent
filesystem like `zfs create -o mountpoint=/k8s-volumes,compression=lz4 rpool/k8s-volumes`.

You should create a shell wrapper script to pass this parent volume as an argument.
Under Kubernetes, there is no facility for doing this and as an administrator, you
may not wish to directly expose this.

An example wrapper script:

```
#!/bin/bash
exec /usr/local/sbin/zfs-flex-volume -parent=rpool/k8s-volumes "$@"
```

Kubernetes
----------
* Create the directory `/usr/libexec/kubernetes/kubelet-plugins/volume/exec/akins.org~zfs/`
* Create/copy the shell wrapper script from above to `/usr/libexec/kubernetes/kubelet-plugins/volume/exec/akins.org~zfs/zfs`

To create a ZFS, create a [Kubernetes pod](http://kubernetes.io/docs/user-guide/pods/) such as:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  containers:
  - name: nginx
    image: nginx
    volumeMounts:
    - name: test
      mountPath: /data
    ports:
    - containerPort: 80
  volumes:
  - name: test
    flexVolume:
      driver: "akins.org/zfs"
      options:
        dataset: "test-volume"
        quota: "1G"
```

And the driver will create a volume  -- `rpool/k8s-volumes/test-volume` in our
example.
The volume will be mounted under the parent - `/k8s-volumes/test-volume` in our
example.
The volume will also be mounted in the container as `/data` inside the container.
The actual path on the host system is under `/var/lib/kubelet`, by default.

When the pods is deleted, the volume will be unmounted from under `/var/lib/kubelet`,
but remained mounted under the parent dataset.  zfs-flex-driver, at this time, does
**not** destroy the filesystem.

Note: multiple pods may mount the same dataset on the same host.

The following options may be passed:
* dataset - **required** - name of the dataset to create.
* quota - **required** - [Quota](https://www.freebsd.org/doc/handbook/zfs-term.html#zfs-term-quota) for the dataset.
* reservation - [reservation](https://www.freebsd.org/doc/handbook/zfs-term.html#zfs-term-reservation) for the dataset. By default, none is set.
* compression - [compression](https://www.freebsd.org/doc/handbook/zfs-term.html#zfs-term-compression). By default no option is set, which means inherit from parent dataset.

Note: the options are only used at **initial creation**. Changing the options will not
change the dataset.

See Also
========
* https://github.com/kubernetes/kubernetes/blob/master/examples/volumes/flexvolume/lvm
* https://www.diamanti.com/blog/flexvolume-explored/

LICENSE
=======
see [LICENSE](./LICENSE)
