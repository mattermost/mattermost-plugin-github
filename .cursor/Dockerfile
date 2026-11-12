# Cloud agent image for building the plugin and running it against the stack in
# e2e/integration/docker-compose.yml (Mattermost and Postgres) and the fake GitHub from
# e2e/integration/fake-github/, from the agent's browser or the Playwright suite. Adapted from
# mattermost-plugin-gitlab's .cursor environment.
FROM ubuntu:24.04

ARG DEBIAN_FRONTEND=noninteractive
# install.sh warns when these drift from go.mod and .nvmrc.
ARG GO_VERSION=1.25.8
ARG NODE_VERSION=24.13.1
ARG DOCKER_VERSION=5:28.5.2-1~ubuntu.24.04~noble

RUN apt-get update && apt-get install -y --no-install-recommends \
        acl \
        build-essential \
        ca-certificates \
        curl \
        fuse-overlayfs \
        git \
        gnupg \
        iptables \
        jq \
        make \
        procps \
        python3 \
        sudo \
        xz-utils \
    && rm -rf /var/lib/apt/lists/*

RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu noble stable" \
        > /etc/apt/sources.list.d/docker.list \
    && apt-get update && apt-get install -y --no-install-recommends \
        docker-ce=${DOCKER_VERSION} \
        docker-ce-cli=${DOCKER_VERSION} \
        containerd.io \
        docker-buildx-plugin \
        docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# Docker runs nested inside the agent's container.
RUN mkdir -p /etc/docker \
    && echo '{"storage-driver": "fuse-overlayfs"}' > /etc/docker/daemon.json \
    && update-alternatives --set iptables /usr/sbin/iptables-legacy \
    && update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy

RUN arch="$(dpkg --print-architecture)" \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" | tar -C /usr/local -xz \
    && node_arch="$([ "$arch" = amd64 ] && echo x64 || echo "$arch")" \
    && mkdir -p /usr/local/node \
    && curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${node_arch}.tar.xz" \
        | tar -C /usr/local/node -xJ --strip-components=1 \
    # Cursor can run the hooks and terminals without the ENV lines below (like the Playbooks, core
    # and Wrangler images, which do the same), so the tools are also on the default PATH.
    && for bin in go gofmt; do ln -sf /usr/local/go/bin/$bin /usr/local/bin/$bin; done \
    && for bin in node npm npx; do ln -sf /usr/local/node/bin/$bin /usr/local/bin/$bin; done

RUN (id ubuntu >/dev/null 2>&1 || useradd -m -s /bin/bash ubuntu) \
    && usermod -aG docker,sudo ubuntu \
    && echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/ubuntu \
    && chmod 0440 /etc/sudoers.d/ubuntu \
    && printf '\nexport GOPATH=/home/ubuntu/go\nexport PATH=/usr/local/go/bin:/usr/local/node/bin:/home/ubuntu/go/bin:$PATH\n' >> /home/ubuntu/.bashrc

USER ubuntu
ENV GOPATH=/home/ubuntu/go
ENV PATH=/usr/local/go/bin:/usr/local/node/bin:/home/ubuntu/go/bin:${PATH}
WORKDIR /home/ubuntu
