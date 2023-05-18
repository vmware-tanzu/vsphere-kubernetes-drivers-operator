#!/bin/bash
# Copyright 2021 The Kubernetes Authors.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#    http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


[ -z "${namespace}" ] && namespace=vmware-system-csi

service=vsphere-webhook-svc
secret=vsphere-webhook-certs

if [ ! -x "$(command -v openssl)" ]; then
    echo "openssl not found"
    exit 1
fi

cat <<EOF >> server.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
prompt = no
[req_distinguished_name]
CN = ${service}.${namespace}.svc
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth, serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${service}
DNS.2 = ${service}.${namespace}
DNS.3 = ${service}.${namespace}.svc
EOF

# Generate the CA cert and private key
openssl req -nodes -new -x509 -sha256 -keyout ca.key -out ca.crt -subj "/CN=vSphere CSI Admission Controller Webhook CA"
openssl genrsa -out webhook-server-tls.key 2048
openssl req -new -sha256 -key webhook-server-tls.key -subj "/CN=${service}.${namespace}.svc" -config server.conf \
  | openssl x509 -req -sha256 -CA ca.crt -CAkey ca.key -CAcreateserial -out webhook-server-tls.crt -extensions v3_req -extfile server.conf

cat <<eof >webhook.config
[WebHookConfig]
port = "8443"
cert-file = "/run/secrets/tls/tls.crt"
key-file = "/run/secrets/tls/tls.key"
eof

cat <<eof >caBundle
"$(openssl base64 -A <"ca.crt")"
eof
