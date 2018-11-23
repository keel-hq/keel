#!/bin/bash
set -e

# the repo path to this repository
REPO_URL="https://charts.keel.sh"

function gen_packages() {
  echo "Packaging charts from source code"
  mkdir -p temp
  for d in chart/*
  do
   if [[ -d $d ]]
   then
      # Will generate a helm package per chart in a folder
      echo "$d"
      helm package "$d"
      # shellcheck disable=SC2035
      mv *.tgz temp/
    fi
  done
}

function index() {
  echo "Fetch charts and index.yaml"
  gsutil rsync gs://charts.keel.sh ./temp/

  echo "Indexing repository"
  if [ -f index.yaml ]; then
    helm repo index --url ${REPO_URL} --merge index.yaml ./temp
  else
    helm repo index --url ${REPO_URL} ./temp
  fi
}

function upload() {
  echo "Upload charts to GCS bucket"
  gsutil rsync ./temp/ gs://charts.keel.sh
}

# generate helm chart packages
gen_packages

# create index
index

# upload to GCS bucket
upload
