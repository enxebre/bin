# Kubernetes cluster-api-provider-libvirt Project

This repository hosts an implementation of a provider for Libvirt for the [cluster-api project](https://sigs.k8s.io/cluster-api).

## Allowing the actuator to connect to the libvirt daemon running on the host machine:

Edit `/etc/libvirt/libvirtd.conf` to set:
```
listen_tls = 0
listen_tcp = 1
auth_tcp="none"
tcp_port = "16509"
```

Edit `/etc/systemd/system/libvirt-bin.service` to set:
```
/usr/sbin/libvirtd -l
```
Then:
```
systemctl restart libvirtd
```

Allow incoming connections:
```
iptables -I INPUT -p tcp --dport 16509 -j ACCEPT -m comment --comment "Allow insecure libvirt clients"
```

Verify you can connect through your host private ip:
```
virsh -c qemu+tcp://host_private_ip/system
```

Before running the installer make sure you set libvirt uri to the host private ip:
https://github.com/enxebre/installer/blob/libvirt-machine-api/examples/tectonic.libvirt.yaml#L13