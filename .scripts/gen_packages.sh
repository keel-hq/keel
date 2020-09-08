#!/bin/bash
set -e

echo "Installing curl"
apt update
apt install curl -y

echo "Installing helm"
curl https://raw.githubusercontent.com/kubernetes/helm/master/scripts/get | bash
helm init -c

echo "Packaging charts from source code"
mkdir -p temp
for d in chart/*
do
 # shellcheck disable=SC3010
 if [[ -d $d ]]
 then
    # Will generate a helm package per chart in a folder
    echo "$d"
    helm package "$d"
    # shellcheck disable=SC2035
    mv *.tgz temp/
  fi
done
