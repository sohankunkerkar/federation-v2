#!/bin/bash

OVERWRITE_KUBECONFIG=y ./scripts/create-clusters.sh

./scripts/deploy-kubefed.sh quay.io/sohankunkerkar/kubefed:master cluster2

./e2e  -kubeconfig=${HOME}/.kube/config -ginkgo.v -single-call-timeout=1m -ginkgo.trace -ginkgo.randomizeAllSpecs