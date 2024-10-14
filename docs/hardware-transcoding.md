
It is a tricky beast to get hardware transcoding working in a LXC container. The following is some information on how I got it working.

## Host
```shell
root@pve:~# ls -al /dev/dri
crw-rw----+  1 root video  226,   1 Oct 14 17:28 card1
crw-rw-rw-+  1 root render 226, 128 Oct 14 17:28 renderD128
```

This GRUB config lets me still access the host via a HDMI cable.
```shell
root@pve:~# cat /etc/default/grub
GRUB_CMDLINE_LINUX_DEFAULT="intel_iommu=on"
```

## LXC
```shell
root@pve:~# cat /etc/pve/lxc/101.conf
# Bind mount the device from the host to the container
arch: amd64
cores: 10
features: nesting=1,keyctl=1
hostname: primary
memory: 10000
mp0: /mnt/host-ssd,mp=/storage,ro=0
net0: name=eth0,bridge=vmbr0,firewall=1,gw=192.168.0.1,hwaddr=BC:24:11:49:ED:89,ip=192.168.0.100/24,type=veth
onboot: 1
ostype: debian
rootfs: container:vm-101-disk-0,size=64G
swap: 1000
lxc.cgroup2.devices.allow: c 10:200 rwm
lxc.cgroup2.devices.allow: c 226:0 rwm
lxc.cgroup2.devices.allow: c 226:128 rwm
lxc.mount.entry: /dev/dri/renderD128 dev/dri/renderD128 none bind,optional,create=file 0 0
lxc.mount.entry: /dev/net dev/net none bind,create=dir
lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
```

```shell
➜  ~ ls -al /dev/dri
total 0
crw-rw-rw-+  1 root sgx  226, 128 Oct 14 16:28 renderD128
```

Output from plex docker container:
```shell
**** permissions for /dev/dri/renderD128 are good ****
```

