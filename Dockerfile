FROM hashicorp/terraform

LABEL maintainer="git-josip"

WORKDIR /srv

ADD providers.tf .
ADD main.tf .

RUN terraform init