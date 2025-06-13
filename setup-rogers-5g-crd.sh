#!/bin/bash
# Setup Custom Resource Definitions for Rogers 5G deployment

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

NAMESPACE="rogers-5g-production"

echo -e "${GREEN}=== Installing Custom Resource Definitions for Rogers 5G Security System ===${NC}"

# Create namespace
echo -e "${YELLOW}Creating namespace: $NAMESPACE...${NC}"
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Create SLA CRD
echo -e "${YELLOW}Creating SLA Custom Resource Definition...${NC}"
cat <<EOF | kubectl apply -f -
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: slas.dgla.io
spec:
  group: dgla.io
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                customer:
                  type: object
                  properties:
                    name:
                      type: string
                    id:
                      type: string
                    tier:
                      type: string
                  required:
                    - name
                    - id
                metrics:
                  type: array
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      threshold:
                        type: string
                      criticalThreshold:
                        type: string
                notifications:
                  type: array
                  items:
                    type: object
                    properties:
                      type:
                        type: string
                      endpoint:
                        type: string
                regulations:
                  type: array
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      region:
                        type: string
  scope: Namespaced
  names:
    plural: slas
    singular: sla
    kind: SLA
    shortNames:
    - sl
EOF

# Create Security Config CRD
echo -e "${YELLOW}Creating Security Configuration Custom Resource Definition...${NC}"
cat <<EOF | kubectl apply -f -
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: securityconfigs.dgla.io
spec:
  group: dgla.io
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                name:
                  type: string
                version:
                  type: string
                components:
                  type: array
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                      enabled:
                        type: boolean
                cryptographic:
                  type: object
                  properties:
                    merkleEnabled:
                      type: boolean
                    zeroKnowledgeProofEnabled:
                      type: boolean
                    dataResidency:
                      type: string
                compliance:
                  type: array
                  items:
                    type: object
                    properties:
                      standard:
                        type: string
                      version:
                        type: string
  scope: Namespaced
  names:
    plural: securityconfigs
    singular: securityconfig
    kind: SecurityConfig
    shortNames:
    - sc
EOF

echo -e "${GREEN}Custom Resource Definitions installed successfully!${NC}"
echo -e "You can now continue with the full deployment script."
