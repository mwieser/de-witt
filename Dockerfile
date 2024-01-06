FROM golang:1.21-bookworm as development

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

ENV MAKEFLAGS "-j 8 --no-print-directory"

# Install required system dependencies
RUN apt-get update \
    && apt-get install -y \
    #
    # Mandadory minimal linux packages
    # Installed at development stage and app stage
    # Do not forget to add mandadory linux packages to the final app Dockerfile stage below!
    #
    # -- START MANDADORY --
    ca-certificates \
    # --- END MANDADORY ---
    # -- START DEVELOPMENT --
    apt-utils \
    dialog \
    openssh-client \
    less \
    iproute2 \
    procps \
    lsb-release \
    locales \
    sudo \
    bash-completion \
    bsdmainutils \
    # --- END DEVELOPMENT ---
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN sed -i -e 's/# en_US.UTF-8 UTF-8/en_US.UTF-8 UTF-8/' /etc/locale.gen && \
    dpkg-reconfigure --frontend=noninteractive locales && \
    update-locale LANG=en_US.UTF-8

ENV LANG en_US.UTF-8

# go gotestsum: (this package should NOT be installed via go get)
# https://github.com/gotestyourself/gotestsum/releases
RUN mkdir -p /tmp/gotestsum \
    && cd /tmp/gotestsum \
    && ARCH="$(arch | sed s/aarch64/arm64/ | sed s/x86_64/amd64/)" \
    && wget "https://github.com/gotestyourself/gotestsum/releases/download/v1.9.0/gotestsum_1.9.0_linux_${ARCH}.tar.gz" \
    && tar xzf "gotestsum_1.9.0_linux_${ARCH}.tar.gz" \
    && cp gotestsum /usr/local/bin/gotestsum \
    && rm -rf /tmp/gotestsum

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b $(go env GOPATH)/bin v1.52.2

# gsdev
# The sole purpose of the "gsdev" cli util is to provide a handy short command for the following (all args are passed):
# go run -tags scripts /app/scripts/main.go "$@"
RUN printf '#!/bin/bash\nset -Eeo pipefail\ncd /app && go run -tags scripts ./scripts/main.go "$@"' > /usr/bin/gsdev && chmod 755 /usr/bin/gsdev
    

ARG USERNAME=development
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd -s /bin/bash --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

RUN mkdir -p /home/$USERNAME/.vscode-server/extensions \
    /home/$USERNAME/.vscode-server-insiders/extensions \
    && chown -R $USERNAME \
    /home/$USERNAME/.vscode-server \
    /home/$USERNAME/.vscode-server-insiders

RUN mkdir -p /$GOPATH/pkg && chown -R $USERNAME /$GOPATH

WORKDIR /app
ENV GOBIN /app/bin
ENV PATH $PATH:$GOBIN