# Kubernetes Setup for ArduSim2

This guide walks you through setting up a Kubernetes cluster for running ArduSim2 in a distributed environment. It covers cluster initialisation, node roles, obtaining the cluster configuration, and ensuring network ports are correctly exposed.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Defining the Master Node (Control Plane)](#2-defining-the-master-node-control-plane)
3. [Obtaining the Join Token](#3-obtaining-the-join-token)
4. [Adding a Worker Node to the Network](#4-adding-a-worker-node-to-the-network)
5. [Obtaining the Configuration File (Kubeconfig)](#5-obtaining-the-configuration-file-kubeconfig)
6. [Port Requirements](#6-port-requirements)
7. [Verifying the Setup](#7-verifying-the-setup)

---

## 1. Prerequisites

Before setting up the cluster, make sure **every node** (both manager and workers) meets the following requirements:

- **Docker installed**: All nodes must have Docker installed and running. See [the official Docker installation guide](https://docs.docker.com/engine/install/) or the existing [`scripts/docker.sh`](../scripts/docker.sh) helper script for Ubuntu/Debian systems.

> [!NOTE]
> All commands in this guide assume you are logged in as a user with `sudo` privileges.

---

## 2. Defining the Master Node (Control Plane)

The master node is not defined in any text file or previous network configuration. It automatically becomes the master simply because it is the machine where you execute the server installation command.

Log in via SSH to the computer you want to be the manager and execute:

```bash
curl -sfL https://get.k3s.io | sh -
```

That's it! That machine is now your Master node. K3s will have installed everything (API, state database, internal network, etc.).

---

## 3. Obtaining the Join Token

Just like Swarm needs a security token to prevent unauthorized servers from joining your cluster, Kubernetes does too. On your master node, run this command to read the auto-generated password:

```bash
sudo cat /var/lib/rancher/k3s/server/node-token
```

*(Copy the long text string it returns).*

---

## 4. Adding a Worker Node to the Network

Now log in via SSH to the second computer (which will be the slave or "worker" node). Instead of installing the server, you install the agent, pointing it to your Master's IP and providing the token:

```bash
curl -sfL https://get.k3s.io | K3S_URL=https://<MASTER-NODE-IP>:6443 K3S_TOKEN=<YOUR-TOKEN> sh -
```

As soon as that command finishes, the worker node will connect to the master, configure its network rules to talk to it (automatically creating the equivalent of your global air network), and will wait to receive Pods.

If you go back to the master node and type `kubectl get nodes`, you will see the full list of your machines ready to work.

---

## 5. Obtaining the Configuration File (Kubeconfig)

Log in via SSH to your Master Node and run this command to display the contents of the configuration file:

```bash
sudo cat /etc/rancher/k3s/k3s.yaml
```

You will see it outputs a long text in YAML format (it will start with `apiVersion: v1` and will have several sections with unintelligible certificates). Copy absolutely all of that text.

If you look closely at the text you just copied, you will see a line near the beginning that says something like this:

```yaml
    server: https://127.0.0.1:6443
```

Since K3s generated that file to be used within the machine itself, it points to itself (`127.0.0.1`). You must delete that local IP and replace it with the real IP of your Master Node on your network.

Save the file in an accessible path.

---

## 6. Port Requirements

ArduSim2 uses several ports for inter-service communication. For the cluster to work correctly, these ports must be **open and reachable between all machines** in the cluster. This typically requires configuring the firewall on each node.

### 6.1 Kubernetes control plane ports (Manager node)

| Port | Protocol | Purpose |
|------|----------|---------|
| `6443` | TCP | Kubernetes API server (required by all nodes and the GUI) |
| `8472` | UDP | Flannel VXLAN (internal networking) |
| `10250` | TCP | Kubelet metrics |

### 6.2 Kubernetes worker node ports

| Port | Protocol | Purpose |
|------|----------|---------|
| `8472` | UDP | Flannel VXLAN (internal networking) |
| `10250` | TCP | Kubelet metrics |
| `30000–32767` | TCP/UDP | NodePort services (used to expose services externally) |

### 6.3 ArduSim2 application ports

Any ports that are going to be used by your ArduSim2 services must be open and accessible between the nodes.

> [!IMPORTANT]
> The ports must not only be **open in the firewall** of each node, but also **accessible from other machines** on the network. A port that is only listening on `localhost` (or `127.0.0.1`) will not be reachable by other nodes. Ensure services bind to `0.0.0.0` or the node's external IP.

### 6.4 Opening ports with `ufw` (Ubuntu)

```bash
# Kubernetes API server (manager only)
sudo ufw allow 6443/tcp

# Flannel & Kubelet (all nodes)
sudo ufw allow 8472/udp
sudo ufw allow 10250/tcp

# NodePort range (all nodes)
sudo ufw allow 30000:32767/tcp
sudo ufw allow 30000:32767/udp

# Custom ArduSim2 ports
# Example: sudo ufw allow <PORT>/tcp
# Example: sudo ufw allow <PORT>/udp

sudo ufw reload
```

### 6.5 Opening ports with `firewalld` (RHEL/Fedora/CentOS)

```bash
sudo firewall-cmd --permanent --add-port=6443/tcp
sudo firewall-cmd --permanent --add-port=8472/udp
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --permanent --add-port=30000-32767/tcp
sudo firewall-cmd --permanent --add-port=30000-32767/udp
# Custom ArduSim2 ports
# Example: sudo firewall-cmd --permanent --add-port=<PORT>/tcp
# Example: sudo firewall-cmd --permanent --add-port=<PORT>/udp
sudo firewall-cmd --reload
```

> [!TIP]
> After opening ports, verify connectivity from another machine using `nc` (netcat):
> ```bash
> # TCP check
> nc -zv <NODE_IP> 6443
> # UDP check
> nc -zuv <NODE_IP> 8472
> ```

---

## 7. Verifying the Setup

Run the following checks from the **manager node** to confirm the cluster is healthy before deploying ArduSim2:

### Cluster nodes

```bash
kubectl get nodes -o wide
```

All nodes should show `Ready` in the `STATUS` column.

### System pods

```bash
kubectl get pods -n kube-system
```

All pods should be in `Running` or `Completed` state.

### Port connectivity

From any worker node, verify the manager's API server is reachable:

```bash
nc -zv <MANAGER_IP> 6443
```

From outside the cluster (e.g., your development machine), verify a NodePort service is reachable:

```bash
nc -zv <ANY_NODE_IP> <NODEPORT>
```

---

## Common Issues

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| `kubectl get nodes` shows `NotReady` | Networking issues or pods not running | Check `kubectl get pods -n kube-system` to see if system pods are running properly |
| Services not reachable from other nodes | Port blocked by firewall | Open the required ports with `ufw` or `firewalld` |
