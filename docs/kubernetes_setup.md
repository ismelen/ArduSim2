# Kubernetes Setup for ArduSim2

This guide walks you through setting up a Kubernetes cluster for running ArduSim2 in a distributed environment. It covers cluster initialisation, node roles, obtaining the cluster configuration, publishing images to Docker Hub, and ensuring network ports are correctly exposed.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Installing Kubernetes](#2-installing-kubernetes)
3. [Initialising the Manager Node](#3-initialising-the-manager-node)
4. [Joining Worker Nodes to the Cluster](#4-joining-worker-nodes-to-the-cluster)
5. [Obtaining the Kubernetes Config](#5-obtaining-the-kubernetes-config)
6. [Publishing Docker Images to Docker Hub](#6-publishing-docker-images-to-docker-hub)
7. [Port Requirements](#7-port-requirements)
8. [Verifying the Setup](#8-verifying-the-setup)

---

## 1. Prerequisites

Before setting up the cluster, make sure **every node** (both manager and workers) meets the following requirements:

- **OS**: A supported Linux distribution (Ubuntu 20.04 LTS or newer recommended).
- **Hardware**: At least 2 CPUs and 2 GB of RAM per node.
- **Docker installed**: All nodes must have Docker (or another compatible container runtime such as `containerd`) installed and running. See [the official Docker installation guide](https://docs.docker.com/engine/install/) or the existing [`scripts/docker.sh`](../scripts/docker.sh) helper script for Ubuntu/Debian systems.
- **Unique hostnames**: Each node must have a unique hostname. Verify with `hostnamectl` and set a new one if needed:
  ```bash
  sudo hostnamectl set-hostname <new-hostname>
  ```
- **Swap disabled**: Kubernetes requires swap to be off on every node:
  ```bash
  sudo swapoff -a
  # Make it permanent:
  sudo sed -i '/ swap / s/^/#/' /etc/fstab
  ```
- **Time synchronisation**: All nodes must have their clocks in sync (e.g., via `timedatectl` and NTP).

> [!NOTE]
> All commands in this guide assume you are logged in as a user with `sudo` privileges.

---

## 2. Installing Kubernetes

The following steps must be performed on **every node** in the cluster (manager and workers alike).

### 2.1 Install `kubeadm`, `kubelet`, and `kubectl`

```bash
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl gpg

curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.30/deb/Release.key | \
  sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] \
  https://pkgs.k8s.io/core:/stable:/v1.30/deb/ /' | \
  sudo tee /etc/apt/sources.list.d/kubernetes.list

sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

### 2.2 Enable `containerd` as the container runtime

If you installed Docker, `containerd` is typically already present. Configure it to use the `systemd` cgroup driver:

```bash
sudo mkdir -p /etc/containerd
containerd config default | sudo tee /etc/containerd/config.toml

# Set the cgroup driver to systemd
sudo sed -i 's/SystemdCgroup = false/SystemdCgroup = true/' /etc/containerd/config.toml

sudo systemctl restart containerd
sudo systemctl enable containerd
```

---

## 3. Initialising the Manager Node

The **manager node** (also called the *control plane*) is the machine that coordinates the entire cluster. Run the following command **only on the manager node**:

```bash
sudo kubeadm init --pod-network-cidr=10.244.0.0/16
```

> [!IMPORTANT]
> The `--pod-network-cidr` value above (`10.244.0.0/16`) is required when using **Flannel** as the pod network add-on (see below). If you choose a different CNI plugin, adjust this value accordingly.

After initialisation completes, `kubeadm` will print a `kubeadm join` command — **copy and save it**. You will need it in the next section to add worker nodes.

### 3.1 Install a Pod Network Add-on (Flannel)

Kubernetes requires a CNI (Container Network Interface) plugin so pods can communicate across nodes. Install Flannel on the manager node:

```bash
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

Wait until all system pods are running before proceeding:

```bash
kubectl get pods -n kube-system --watch
```

---

## 4. Joining Worker Nodes to the Cluster

On each **worker node**, run the `join` command that was printed during manager initialisation. It looks like this:

```bash
sudo kubeadm join <MANAGER_IP>:6443 \
  --token <TOKEN> \
  --discovery-token-ca-cert-hash sha256:<HASH>
```

Replace `<MANAGER_IP>`, `<TOKEN>`, and `<HASH>` with the actual values from the output of `kubeadm init`.

> [!NOTE]
> The bootstrap token expires after **24 hours**. If you need to add a node later, generate a new token on the manager node:
> ```bash
> kubeadm token create --print-join-command
> ```

Verify all nodes appear in the cluster from the manager:

```bash
kubectl get nodes
```

All nodes should eventually show a `Ready` status.

---

## 5. Obtaining the Kubernetes Config

The Kubernetes config file (commonly called `kubeconfig`) is the credential file that grants access to your cluster. It is needed by `kubectl` and by ArduSim2's GUI to deploy workloads.

### 5.1 On the manager node itself

After `kubeadm init`, the config is placed at `/etc/kubernetes/admin.conf`. Copy it to your user's home directory so `kubectl` can use it without root:

```bash
mkdir -p $HOME/.kube
sudo cp /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

Verify access:

```bash
kubectl cluster-info
```

### 5.2 Accessing the cluster from another machine

To control the cluster from a remote machine (e.g., your development laptop or the ArduSim2 GUI host), copy the config file from the manager:

```bash
# Run this on the remote machine
scp <manager-user>@<MANAGER_IP>:/etc/kubernetes/admin.conf ~/.kube/config
```

> [!CAUTION]
> The `admin.conf` file contains cluster-admin credentials. Treat it like a private key — do not share it, do not commit it to version control, and restrict file permissions:
> ```bash
> chmod 600 ~/.kube/config
> ```

### 5.3 Providing the config to ArduSim2

ArduSim2's GUI expects the kubeconfig to be available at the default location (`~/.kube/config`), or via the `KUBECONFIG` environment variable:

```bash
export KUBECONFIG=/path/to/your/kubeconfig
```

---

## 6. Publishing Docker Images to Docker Hub

ArduSim2's containers must be pushed to a Docker registry (Docker Hub by default) so that all Kubernetes nodes can pull them. Docker Hub is a public registry — images you push there are accessible by any node in the cluster without additional configuration.

### 6.1 Log in to Docker Hub

> [!IMPORTANT]
> You must be logged in to Docker Hub **before** building and pushing images. Run this on the machine where you are building the images (typically your development machine or the CI runner):
> ```bash
> docker login
> ```
> You will be prompted for your Docker Hub username and password (or an access token). Without an active session, the push will be rejected with a `denied: requested access to the resource is denied` error.

To avoid entering credentials repeatedly, you can create an **access token** in your Docker Hub account settings and use it instead of your password:

```bash
docker login -u <your-dockerhub-username>
# When prompted for a password, paste your access token
```

### 6.2 The destination repository must exist

Docker Hub does **not** automatically create repositories. You must create the target repository before pushing to it for the first time.

1. Log in to [hub.docker.com](https://hub.docker.com).
2. Click **Create repository**.
3. Set the name to match the image tag you plan to use (e.g., `ardusim2-netsim`).
4. Choose **Public** (required for nodes to pull without credentials) or **Private** (requires configuring an image pull secret in Kubernetes — see section 6.4).
5. Click **Create**.

> [!WARNING]
> If you push to a repository that does not exist, Docker Hub will return a `repository does not exist` error. Always create the repository first.

### 6.3 Building and pushing an image

Tag the image to include your Docker Hub username and the target repository:

```bash
# Build the image
docker build -t <your-dockerhub-username>/<repository-name>:<tag> ./src/<module>/

# Push the image
docker push <your-dockerhub-username>/<repository-name>:<tag>
```

**Example** for the `netsim` module:

```bash
docker build -t myuser/ardusim2-netsim:latest ./src/netsim/
docker push myuser/ardusim2-netsim:latest
```

### 6.4 Pulling private images in Kubernetes (optional)

If your Docker Hub repositories are set to **Private**, each Kubernetes node needs credentials to pull the images. Create an image pull secret and reference it in your deployment manifests:

```bash
kubectl create secret docker-registry regcred \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=<your-dockerhub-username> \
  --docker-password=<your-access-token> \
  --docker-email=<your-email>
```

Then add `imagePullSecrets` to your Pod spec:

```yaml
spec:
  imagePullSecrets:
    - name: regcred
  containers:
    - name: netsim
      image: myuser/ardusim2-netsim:latest
```

---

## 7. Port Requirements

ArduSim2 uses several ports for inter-service communication. For the cluster to work correctly, these ports must be **open and reachable between all machines** in the cluster. This typically requires configuring the firewall on each node.

### 7.1 Kubernetes control plane ports (Manager node)

| Port | Protocol | Purpose |
|------|----------|---------|
| `6443` | TCP | Kubernetes API server (required by all nodes and the GUI) |
| `2379–2380` | TCP | etcd server client API |
| `10250` | TCP | Kubelet API |
| `10257` | TCP | kube-controller-manager |
| `10259` | TCP | kube-scheduler |

### 7.2 Kubernetes worker node ports

| Port | Protocol | Purpose |
|------|----------|---------|
| `10250` | TCP | Kubelet API |
| `30000–32767` | TCP/UDP | NodePort services (used to expose ArduSim2 services externally) |

### 7.3 ArduSim2 application ports

The following ports are used by ArduSim2 services and must be open between UAV nodes and the ground station:

| Port | Protocol | Service | Notes |
|------|----------|---------|-------|
| `3400` | UDP | Communication Module (broker) | Internal pub/sub broker |
| `9876` | UDP | UAV Controller | MAVLink over UDP |
| `14550` | UDP | GCS Telemetry | Standard MAVLink ground station port |
| `8765` | TCP | NetSim Gateway | Swarm-wide network simulation bus |

> [!IMPORTANT]
> The ports listed above must not only be **open in the firewall** of each node, but also **accessible from other machines** on the network. A port that is only listening on `localhost` (or `127.0.0.1`) will not be reachable by other nodes. Ensure services bind to `0.0.0.0` or the node's external IP.

### 7.4 Opening ports with `ufw` (Ubuntu)

```bash
# Kubernetes API server (manager only)
sudo ufw allow 6443/tcp

# Kubelet (all nodes)
sudo ufw allow 10250/tcp

# NodePort range (all nodes)
sudo ufw allow 30000:32767/tcp
sudo ufw allow 30000:32767/udp

# ArduSim2 services
sudo ufw allow 3400/udp
sudo ufw allow 9876/udp
sudo ufw allow 14550/udp
sudo ufw allow 8765/tcp

sudo ufw reload
```

### 7.5 Opening ports with `firewalld` (RHEL/Fedora/CentOS)

```bash
sudo firewall-cmd --permanent --add-port=6443/tcp
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --permanent --add-port=30000-32767/tcp
sudo firewall-cmd --permanent --add-port=30000-32767/udp
sudo firewall-cmd --permanent --add-port=3400/udp
sudo firewall-cmd --permanent --add-port=9876/udp
sudo firewall-cmd --permanent --add-port=14550/udp
sudo firewall-cmd --permanent --add-port=8765/tcp
sudo firewall-cmd --reload
```

> [!TIP]
> After opening ports, verify connectivity from another machine using `nc` (netcat):
> ```bash
> # TCP check
> nc -zv <NODE_IP> 6443
> # UDP check
> nc -zuv <NODE_IP> 3400
> ```

---

## 8. Verifying the Setup

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

### Docker Hub connectivity (on each worker)

```bash
docker pull hello-world
```

If the pull succeeds, the node can reach Docker Hub and pull images.

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
| `kubectl get nodes` shows `NotReady` | CNI plugin not installed or pods not running | Check `kubectl get pods -n kube-system` and apply the Flannel manifest again |
| `docker push` fails with `denied` | Not logged in to Docker Hub | Run `docker login` and retry |
| `docker push` fails with `repository does not exist` | Destination repo not created | Create the repository on hub.docker.com first |
| Pods stuck in `ImagePullBackOff` | Node cannot pull the image | Check repository visibility; if private, configure `imagePullSecrets` |
| Services not reachable from other nodes | Port blocked by firewall | Open the required ports with `ufw` or `firewalld` |
| `kubeadm join` fails | Token expired (>24 h) | Run `kubeadm token create --print-join-command` on the manager and use the new command |
