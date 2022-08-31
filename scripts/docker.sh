#!/usr/bin/env bash
set -e

# PR / prod
ACTION=$1

# build / push
BUILD=$2

ACR_REPO=${REGISTRY_NAME}.azurecr.io/${APP_NAME}

if [[ ${BUILD} =~ ^pr-.* ]]; then
  if [[ ${ACTION} == "build" ]]; then
    docker build . -t "${ACR_REPO}:${BUILD}"
  elif [[ ${ACTION} == "push" ]]; then
    docker push "${ACR_REPO}:${BUILD}"
  else
    echo "Action $ACTION not found. build and push are valid actions"
  fi
elif [[ ${BUILD} == "prod" ]]; then
  TAG="prod-$(git show --no-patch --no-notes --pretty=format:"%h-%ad" --date=format:'%Y%m%d%H%M%S' "${GITHUB_SHA}")"
  echo "Promoting ${ACR_REPO}:pr-${GITHUB_EVENT_NUMBER} to ${ACR_REPO}:${TAG}"
  az acr import --force -n "${REGISTRY_NAME}" --subscription "${REGISTRY_SUB}" --source "${ACR_REPO}:pr-${GITHUB_EVENT_NUMBER}" -t "${APP_NAME}:${TAG}"

else
  echo "Build type not recognised, use pr-{pr_number} or prod"
  exit 1
fi