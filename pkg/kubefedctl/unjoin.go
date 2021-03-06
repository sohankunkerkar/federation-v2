/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubefedctl

import (
	"context"
	goerrors "errors"
	"io"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	fedv1a1 "sigs.k8s.io/kubefed/pkg/apis/core/v1alpha1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
	controllerutil "sigs.k8s.io/kubefed/pkg/controller/util"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/options"
	"sigs.k8s.io/kubefed/pkg/kubefedctl/util"
)

var (
	unjoin_long = `
		Unjoin removes a cluster from a federation.
		Current context is assumed to be a Kubernetes cluster
		hosting the kubefed control plane. Please use the
		--host-cluster-context flag otherwise.`
	unjoin_example = `
		# Unjoin a cluster from a federation by specifying the
		# cluster name and the context name of the federation
		# control plane's host cluster. Cluster name must be
		# a valid RFC 1123 subdomain name. Cluster context
		# must be specified if the cluster name is different
		# than the cluster's context in the local kubeconfig.
		kubefedctl unjoin foo --host-cluster-context=bar`
)

type unjoinFederation struct {
	options.GlobalSubcommandOptions
	options.CommonJoinOptions
	unjoinFederationOptions
}

type unjoinFederationOptions struct {
	forceDeletion bool
}

// Bind adds the unjoin specific arguments to the flagset passed in as an
// argument.
func (o *unjoinFederationOptions) Bind(flags *pflag.FlagSet) {
	flags.BoolVar(&o.forceDeletion, "force", false,
		"Delete federated cluster and secret resources even if resources in the cluster targeted for unjoin are not removed successfully.")
}

// NewCmdUnjoin defines the `unjoin` command that unjoins a cluster from a
// federation.
func NewCmdUnjoin(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &unjoinFederation{}

	cmd := &cobra.Command{
		Use:     "unjoin CLUSTER_NAME --host-cluster-context=HOST_CONTEXT",
		Short:   "Unjoin a cluster from a federation",
		Long:    unjoin_long,
		Example: unjoin_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.Complete(args)
			if err != nil {
				klog.Fatalf("Error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				klog.Fatalf("Error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.GlobalSubcommandBind(flags)
	opts.CommonSubcommandBind(flags)
	opts.Bind(flags)

	return cmd
}

// Complete ensures that options are valid and marshals them if necessary.
func (j *unjoinFederation) Complete(args []string) error {
	err := j.SetName(args)
	if err != nil {
		return err
	}

	if j.ClusterContext == "" {
		klog.V(2).Infof("Defaulting cluster context to unjoining cluster name %s", j.ClusterName)
		j.ClusterContext = j.ClusterName
	}

	if j.HostClusterName != "" && strings.ContainsAny(j.HostClusterName, ":/") {
		return goerrors.New("host-cluster-name may not contain \"/\" or \":\"")
	}

	if j.HostClusterName == "" && strings.ContainsAny(j.HostClusterContext, ":/") {
		return goerrors.New("host-cluster-name must be set if the name of the host cluster context contains one of \":\" or \"/\"")
	}

	klog.V(2).Infof("Args and flags: name %s, host-cluster-context: %s, host-system-namespace: %s, kubeconfig: %s, cluster-context: %s, dry-run: %v",
		j.ClusterName, j.HostClusterContext, j.KubefedNamespace, j.Kubeconfig, j.ClusterContext, j.DryRun)

	return nil
}

// Run is the implementation of the `unjoin federation` command.
func (j *unjoinFederation) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		// TODO(font): Return new error with this same text so it can be output
		// by caller.
		klog.V(2).Infof("Failed to get host cluster config: %v", err)
		return err
	}

	clusterConfig, err := config.ClusterConfig(j.ClusterContext, j.Kubeconfig)
	if err != nil {
		klog.V(2).Infof("Failed to get unjoining cluster config: %v", err)

		if !j.forceDeletion {
			return err
		}
		// If configuration for the member cluster cannot be successfully loaded,
		// forceDeletion indicates that resources associated with the member cluster
		// should still be removed from the host cluster.
	}

	hostClusterName := j.HostClusterContext
	if j.HostClusterName != "" {
		hostClusterName = j.HostClusterName
	}

	return UnjoinCluster(hostConfig, clusterConfig, j.KubefedNamespace,
		hostClusterName, j.HostClusterContext, j.ClusterContext, j.ClusterName, j.forceDeletion, j.DryRun)
}

// UnjoinCluster performs all the necessary steps to unjoin a cluster from the
// federation provided the required set of parameters are passed in.
func UnjoinCluster(hostConfig, clusterConfig *rest.Config, kubefedNamespace, hostClusterName, hostClusterContext,
	unjoiningClusterContext, unjoiningClusterName string, forceDeletion, dryRun bool) error {

	hostClientset, err := util.HostClientset(hostConfig)
	if err != nil {
		klog.V(2).Infof("Failed to get host cluster clientset: %v", err)
		return err
	}

	var clusterClientset *kubeclient.Clientset
	if clusterConfig != nil {
		clusterClientset, err = util.ClusterClientset(clusterConfig)
		if err != nil {
			klog.V(2).Infof("Failed to get unjoining cluster clientset: %v", err)
			if !forceDeletion {
				return err
			}
		}
	}

	client, err := genericclient.New(hostConfig)
	if err != nil {
		klog.V(2).Infof("Failed to get federation clientset: %v", err)
		return err
	}

	var deletionSucceeded bool
	if clusterClientset != nil {
		deletionSucceeded = deleteRBACResources(clusterClientset, kubefedNamespace, unjoiningClusterName, hostClusterName, dryRun)

		err = deleteFedNSFromUnjoinCluster(hostClientset, clusterClientset, kubefedNamespace, unjoiningClusterName, dryRun)
		if err != nil {
			klog.Errorf("Error deleting kubefed namespace from unjoin cluster: %v", err)
			deletionSucceeded = false
		}
	}

	// deletionSucceeded when all operations in deleteRBACResources and deleteFedNSFromUnjoinCluster succeed.
	if deletionSucceeded || forceDeletion {
		deleteKubefedClusterAndSecret(hostClientset, client, kubefedNamespace, unjoiningClusterName, dryRun)
	}

	return nil
}

// deleteKubefedClusterAndSecret deletes a federated cluster resource that associates
// the cluster and secret.
func deleteKubefedClusterAndSecret(hostClientset kubeclient.Interface, client genericclient.Client,
	kubefedNamespace, unjoiningClusterName string, dryRun bool) {
	if dryRun {
		return
	}

	klog.V(2).Infof("Deleting federated cluster resource from namespace: %s for unjoin cluster: %s",
		kubefedNamespace, unjoiningClusterName)

	fedCluster := &fedv1a1.KubefedCluster{}
	err := client.Get(context.TODO(), fedCluster, kubefedNamespace, unjoiningClusterName)
	if err != nil {
		klog.Errorf("Failed to get KubefedCluster resource from namespace: %s for unjoin cluster: %s due to: %v", kubefedNamespace, unjoiningClusterName, err)
		return
	}

	err = hostClientset.CoreV1().Secrets(kubefedNamespace).Delete(fedCluster.Spec.SecretRef.Name,
		&metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("Failed to delete Secret resource from namespace: %s for unjoin cluster: %s due to: %v", kubefedNamespace, unjoiningClusterName, err)
	} else {
		klog.V(2).Infof("Deleted Secret resource from namespace: %s for unjoin cluster: %s", kubefedNamespace, unjoiningClusterName)
	}

	err = client.Delete(context.TODO(), fedCluster, fedCluster.Namespace, fedCluster.Name)
	if err != nil {
		klog.Errorf("Failed to delete KubefedCluster resource from namespace: %s for unjoin cluster: %s due to: %v", kubefedNamespace, unjoiningClusterName, err)
	} else {
		klog.V(2).Infof("Deleted KubefedCluster resource from namespace: %s for unjoin cluster: %s", kubefedNamespace, unjoiningClusterName)
	}
}

// deleteRBACResources deletes the cluster role, cluster rolebindings and service account
// from the unjoining cluster.
func deleteRBACResources(unjoiningClusterClientset kubeclient.Interface,
	namespace, unjoiningClusterName, hostClusterName string, dryRun bool) bool {

	saName := util.ClusterServiceAccountName(unjoiningClusterName, hostClusterName)

	klog.V(2).Infof("Deleting cluster role binding for service account: %s in unjoining cluster: %s",
		saName, unjoiningClusterName)

	deletionSucceeded := deleteClusterRoleAndBinding(unjoiningClusterClientset, saName, namespace, dryRun)
	if deletionSucceeded {
		klog.V(2).Infof("Deleted cluster role binding for service account: %s in unjoining cluster: %s",
			saName, unjoiningClusterName)
	}

	klog.V(2).Infof("Deleting service account %s in unjoining cluster: %s", saName, unjoiningClusterName)

	err := deleteServiceAccount(unjoiningClusterClientset, saName, namespace, dryRun)
	if err != nil {
		deletionSucceeded = false
		klog.Errorf("Error deleting service account: %s in unjoining cluster. %v", saName, err)
	} else {
		klog.V(2).Infof("Deleted service account %s in unjoining cluster: %s", saName, unjoiningClusterName)
	}

	return deletionSucceeded
}

// deleteFedNSFromUnjoinCluster deletes the kubefed namespace from
// the unjoining cluster so long as the unjoining cluster is not the
// host cluster.
func deleteFedNSFromUnjoinCluster(hostClientset, unjoiningClusterClientset kubeclient.Interface,
	kubefedNamespace, unjoiningClusterName string, dryRun bool) error {

	if dryRun {
		return nil
	}

	hostClusterNamespace, err := hostClientset.CoreV1().Namespaces().Get(kubefedNamespace, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Error retrieving namespace %q from host cluster", kubefedNamespace)
	}

	unjoiningClusterNamespace, err := unjoiningClusterClientset.CoreV1().Namespaces().Get(kubefedNamespace, metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "Error retrieving namespace %q from unjoining cluster %q", kubefedNamespace, unjoiningClusterName)
	}

	if controllerutil.IsPrimaryCluster(hostClusterNamespace, unjoiningClusterNamespace) {
		klog.V(2).Infof("The kubefed namespace %q does not need to be deleted from the host cluster by unjoin.", kubefedNamespace)
		return nil
	}

	klog.V(2).Infof("Deleting kubefed namespace %q from unjoining cluster %q", kubefedNamespace, unjoiningClusterName)
	err = unjoiningClusterClientset.CoreV1().Namespaces().Delete(kubefedNamespace, &metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		klog.V(2).Infof("The kubefed namespace %q no longer exists in unjoining cluster %q", kubefedNamespace, unjoiningClusterName)
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "Could not delete kubefed namespace %q from unjoining cluster %q", kubefedNamespace, unjoiningClusterName)
	}
	klog.V(2).Infof("Deleted kubefed namespace %q from unjoining cluster %q", kubefedNamespace, unjoiningClusterName)
	return nil
}

// deleteServiceAccount deletes a service account in the cluster associated
// with clusterClientset with credentials that are used by the host cluster
// to access its API server.
func deleteServiceAccount(clusterClientset kubeclient.Interface, saName,
	namespace string, dryRun bool) error {
	if dryRun {
		return nil
	}

	// Delete a service account.
	return clusterClientset.CoreV1().ServiceAccounts(namespace).Delete(saName,
		&metav1.DeleteOptions{})
}

// deleteClusterRoleAndBinding deletes an RBAC cluster role and binding that
// allows the service account identified by saName to access all resources in
// all namespaces in the cluster associated with clusterClientset.
func deleteClusterRoleAndBinding(clusterClientset kubeclient.Interface, saName, namespace string, dryRun bool) bool {
	var deletionSucceeded = true

	if dryRun {
		return deletionSucceeded
	}

	roleName := util.RoleName(saName)
	healthCheckRoleName := util.HealthCheckRoleName(saName, namespace)

	// Attempt to delete all role and role bindings created by join
	// and ignore if there is any error

	for _, name := range []string{roleName, healthCheckRoleName} {
		err := clusterClientset.RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			deletionSucceeded = false
			klog.Errorf("Could not delete cluster role binding %q in unjoining cluster: %v", name, err)
		}

		err = clusterClientset.RbacV1().ClusterRoles().Delete(name, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			deletionSucceeded = false
			klog.Errorf("Could not delete cluster role %q in unjoining cluster: %v", name, err)
		}
	}

	err := clusterClientset.RbacV1().RoleBindings(namespace).Delete(roleName, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		deletionSucceeded = false
		klog.Errorf("Could not delete role binding for service account: %s in unjoining cluster: %v",
			saName, err)
	}

	err = clusterClientset.RbacV1().Roles(namespace).Delete(roleName, &metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		deletionSucceeded = false
		klog.Errorf("Could not delete role for service account: %s in unjoining cluster: %v",
			saName, err)
	}

	return deletionSucceeded
}
