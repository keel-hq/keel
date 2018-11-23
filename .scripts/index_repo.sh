#!/bin/sh
set -e

echo "Installing curl"
apt update
apt install curl -y

echo "Installing helm"
curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get | bash
helm init -c

echo "Indexing repository"
if [ -f index.yaml ]; then
  helm repo index --url "${REPO_URL}" --merge index.yaml ./temp
else
  helm repo index --url "${REPO_URL}" ./temp
fi