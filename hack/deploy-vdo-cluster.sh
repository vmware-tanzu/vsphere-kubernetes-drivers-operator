#!/bin/bash

set -o errexit
set -o pipefail
set -o nounset

IS_DOCKER=$(which docker &> /dev/null && echo false || echo true)
IMG_EXPORT_DIR="export"
SSHARGS="-q -o PubkeyAuthentication=no -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no"
imgName=$1
SPEC_FILE="artifacts/vanilla/vdo-spec.yaml"
REMOTE_DEST_DIR="/tmp"
REMOTE_DEST_SPEC_FILE="vdo-spec.yaml"

function verifyEnvironmentVariables() {
    if [[ -z ${K8S_MASTER_IP:-} ]]; then
        error "Error: The K8S_MASTER_IP environment variable must be set" \
             "to point to a valid K8S Master"
        exit 1
    fi

    if [[ -z ${K8S_MASTER_SSH_PWD:-} ]]; then
        # Often the K8S_MASTER_SSH_PWD is set to a default. The below sets a
        # common default so the user of this script does not need to set it.
        k8sMasterPwd='ca$hc0w'
    else
        k8sMasterPwd=$K8S_MASTER_SSH_PWD
    fi

    if [[ -z ${K8S_MASTER_SSH_USER:-} ]]; then
        # Often the K8S_MASTER_SSH_USER is set to a default. The below sets a
        # common default so the user of this script does not need to set it.
        k8sMasterUser='root'
    else
        k8sMasterUser=$K8S_MASTER_SSH_USER
    fi

    k8sMasterIP=$K8S_MASTER_IP
}

function transfer_image() {
  echo "$k8sMasterIP: Copying local $IMG_EXPORT_DIR/vdo.tar to remote ${REMOTE_DEST_DIR}/vdo.tar."
  SSHPASS="$k8sMasterPwd" sshpass -e scp $SSHARGS \
           "$IMG_EXPORT_DIR/vdo.tar" "$k8sMasterUser@$k8sMasterIP:${REMOTE_DEST_DIR}/vdo.tar"
}

function load_image() {
  if [ ${IS_DOCKER} ]; then
      SSHPASS="$k8sMasterPwd" sshpass -e ssh $SSHARGS \
      "$k8sMasterUser@$k8sMasterIP" "docker import ${REMOTE_DEST_DIR}/vdo.tar ${imgName}"
      SSHPASS="$k8sMasterPwd" sshpass -e scp $SSHARGS \
      ${SPEC_FILE} "$k8sMasterUser@$k8sMasterIP:${REMOTE_DEST_DIR}/${REMOTE_DEST_SPEC_FILE}"
  else
      SSHPASS="$k8sMasterPwd" sshpass -e ssh $SSHARGS \
      "$k8sMasterUser@$k8sMasterIP" "ctr -n=k8s.io images import  ${IMG_EXPORT_DIR}/${imgName}.tar"
      SSHPASS="$k8sMasterPwd" sshpass -e scp $SSHARGS \
      ${SPEC_FILE} "$k8sMasterUser@$k8sMasterIP:${REMOTE_DEST_DIR}/${REMOTE_DEST_SPEC_FILE}"
  fi
}

function apply_spec() {
      SSHPASS="$k8sMasterPwd" sshpass -e ssh $SSHARGS \
      "$k8sMasterUser@$k8sMasterIP" "kubectl --kubeconfig=/etc/kubernetes/admin.conf apply -f ${REMOTE_DEST_DIR}/${REMOTE_DEST_SPEC_FILE}"
}

verifyEnvironmentVariables
transfer_image
load_image
apply_spec