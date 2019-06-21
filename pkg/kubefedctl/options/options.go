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

package options

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/rest"

	fedv1b1 "github.com/openshift/kubefed/pkg/apis/core/v1beta1"
	genericclient "github.com/openshift/kubefed/pkg/client/generic"
	"github.com/openshift/kubefed/pkg/controller/util"
)

// GlobalSubcommandOptions holds the configuration required by the subcommands of
// `kubefedctl`.
type GlobalSubcommandOptions struct {
	HostClusterContext string
	KubeFedNamespace   string
	Kubeconfig         string
	DryRun             bool
}

// GlobalSubcommandBind adds the global subcommand flags to the flagset passed in.
func (o *GlobalSubcommandOptions) GlobalSubcommandBind(flags *pflag.FlagSet) {
	flags.StringVar(&o.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")
	flags.StringVar(&o.HostClusterContext, "host-cluster-context", "", "Host cluster context")
	flags.StringVar(&o.KubeFedNamespace, "kubefed-namespace", util.DefaultKubeFedSystemNamespace,
		"Namespace in the host cluster where the KubeFed system components are installed. This namespace will also be the target of propagation if the controller manager is running with namespaced scope.")
	flags.BoolVar(&o.DryRun, "dry-run", false,
		"Run the command in dry-run mode, without making any server requests.")
}

// CommonJoinOptions holds the common configuration required by the join and
// unjoin subcommands of `kubefedctl`.
type CommonJoinOptions struct {
	ClusterName     string
	ClusterContext  string
	HostClusterName string
}

// CommonSubcommandBind adds the common subcommand flags to the flagset passed in.
func (o *CommonJoinOptions) CommonSubcommandBind(flags *pflag.FlagSet) {
	flags.StringVar(&o.ClusterContext, "cluster-context", "",
		"Name of the cluster's context in the local kubeconfig. Defaults to cluster name if unspecified.")
	flags.StringVar(&o.HostClusterName, "host-cluster-name", "",
		"If set, overrides the use of host-cluster-context name in resource names created in the target cluster. This option must be used when the context name has characters invalid for kubernetes resources like \"/\" and \":\".")
}

// SetName sets the name from the args passed in for the required positional
// argument.
func (o *CommonJoinOptions) SetName(args []string) error {
	if len(args) == 0 {
		return errors.New("NAME is required")
	}

	o.ClusterName = args[0]
	return nil
}

func GetScopeFromKubeFedConfig(hostConfig *rest.Config, namespace string) (apiextv1b1.ResourceScope, error) {
	client, err := genericclient.New(hostConfig)
	if err != nil {
		err = errors.Wrap(err, "Failed to get kubefed clientset")
		return "", err
	}

	fedConfig := &fedv1b1.KubeFedConfig{}
	err = client.Get(context.TODO(), fedConfig, namespace, util.KubeFedConfigName)
	if err != nil {
		config := util.QualifiedName{
			Namespace: namespace,
			Name:      util.KubeFedConfigName,
		}
		err = errors.Wrapf(err, "Error retrieving KubeFedConfig %q", config)
		return "", err
	}

	return fedConfig.Spec.Scope, nil
}

// CommonEnableOptions holds the common configuration required by the enable
// and disable subcommands of `kubefedctl`.
type CommonEnableOptions struct {
	TargetName     string
	FederatedGroup string
	TargetVersion  string
}

// Default values for the federated group and version used by
// the enable and disable subcommands of `kubefedctl`.
const (
	DefaultFederatedGroup   = "types.kubefed.k8s.io"
	DefaultFederatedVersion = "v1beta1"
)

// CommonSubcommandBind adds the common subcommand flags to the flagset passed in.
func (o *CommonEnableOptions) CommonSubcommandBind(flags *pflag.FlagSet, federatedGroupUsage, targetVersionUsage string) {
	flags.StringVar(&o.FederatedGroup, "federated-group", DefaultFederatedGroup, federatedGroupUsage)
	flags.StringVar(&o.TargetVersion, "version", "", targetVersionUsage)
}

// SetName sets the name from the args passed in for the required positional
// argument.
func (o *CommonEnableOptions) SetName(args []string) error {
	if len(args) == 0 {
		return errors.New("NAME is required")
	}

	o.TargetName = args[0]
	return nil
}
