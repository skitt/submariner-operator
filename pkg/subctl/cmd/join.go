/*
© 2019 Red Hat, Inc. and others.

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

package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	submariner "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	lighthouse "github.com/submariner-io/submariner-operator/pkg/subctl/lighthouse/deploy"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinerop"
)

var (
	clusterID       string
	serviceCIDR     string
	clusterCIDR     string
	repository      string
	imageVersion    string
	nattPort        int
	ikePort         int
	colorCodes      string
	disableNat      bool
	ipsecDebug      bool
	submarinerDebug bool
	noLabel         bool
)

func init() {
	addJoinFlags(joinCmd)
	rootCmd.AddCommand(joinCmd)

}

func addJoinFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&clusterID, "clusterid", "", "cluster ID used to identify the tunnels")
	cmd.Flags().StringVar(&serviceCIDR, "servicecidr", "", "service CIDR")
	cmd.Flags().StringVar(&clusterCIDR, "clustercidr", "", "cluster CIDR")
	cmd.Flags().StringVar(&repository, "repository", "", "image repository")
	cmd.Flags().StringVar(&imageVersion, "version", "", "image version")
	cmd.Flags().StringVarP(&operatorImage, "operator-image", "o", DefaultOperatorImage,
		"the operator image you wish to use")
	cmd.Flags().StringVar(&colorCodes, "colorcodes", "blue", "color codes")
	cmd.Flags().IntVar(&nattPort, "nattport", 4500, "IPsec NATT port")
	cmd.Flags().IntVar(&ikePort, "ikeport", 500, "IPsec IKE port")
	cmd.Flags().BoolVar(&disableNat, "disable-nat", false, "Disable NAT for IPsec")
	cmd.Flags().BoolVar(&ipsecDebug, "ipsec-debug", false, "Enable IPsec debugging (verbose logging)")
	cmd.Flags().BoolVar(&submarinerDebug, "subm-debug", false, "Enable Submariner debugging (verbose logging)")
	cmd.Flags().BoolVar(&noLabel, "no-label", false, "skip gateway labeling")
}

const (
	SubmarinerNamespace = "submariner-operator" // We currently expect everything in submariner-operator
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "connect a cluster to an existing broker",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := checkArgumentPassed(args)
		exitOnError("Argument missing", err)
		subctlData, err := datafile.NewFromFile(args[0])
		exitOnError("Error loading the broker information from the given file", err)
		fmt.Printf("* %s says broker is at: %s\n", args[0], subctlData.BrokerURL)
		config, err := getRestConfig(kubeConfig, kubeContext)
		exitOnError("Error connecting to the target cluster", err)
		joinSubmarinerCluster(config, subctlData)
	},
}

func checkArgumentPassed(args []string) error {
	if len(args) == 0 {
		return errors.New("broker-info.subm file generated by 'subctl deploy-broker' not passed")
	}
	return nil
}

func joinSubmarinerCluster(config *rest.Config, subctlData *datafile.SubctlData) {

	// Missing information
	var qs = []*survey.Question{}

	if valid, _ := isValidClusterID(clusterID); !valid {
		qs = append(qs, &survey.Question{
			Name:   "clusterID",
			Prompt: &survey.Input{Message: "What is your cluster ID?"},
			Validate: func(val interface{}) error {
				str, ok := val.(string)
				if !ok {
					return nil
				}
				_, err := isValidClusterID(str)
				return err
			},
		})
	}
	if len(colorCodes) == 0 {
		qs = append(qs, &survey.Question{
			Name:     "colorCodes",
			Prompt:   &survey.Input{Message: "What color codes should be used (e.g. \"blue\")?"},
			Validate: survey.Required,
		})
	}

	if len(qs) > 0 {
		answers := struct {
			ClusterID  string
			ColorCodes string
		}{}

		err := survey.Ask(qs, &answers)
		// Most likely a programming error
		panicOnError(err)

		if len(answers.ClusterID) > 0 {
			clusterID = answers.ClusterID
		}
		if len(answers.ColorCodes) > 0 {
			colorCodes = answers.ColorCodes
		}
	}

	if !noLabel {
		err := handleNodeLabels(config)
		exitOnError("Unable to set the gateway node up", err)
	}

	status := cli.NewStatus()

	status.Start("Deploying the Submariner operator")
	err := submarinerop.Ensure(status, config, OperatorNamespace, operatorImage)
	status.End(err == nil)
	exitOnError("Error deploying the operator", err)

	if subctlData.ServiceDiscovery {
		status.Start("Deploying multi cluster service discovery")
		err = lighthouse.Ensure(status, config, "", "", false)
		status.End(err == nil)
		exitOnError("Error deploying multi cluster service discovery", err)
	}

	fmt.Printf("* Discovering network details\n")
	networkDetails := getNetworkDetails(config)

	serviceCIDR, err = getServiceCIDR(serviceCIDR, networkDetails)
	exitOnError("Error determining the service CIDR", err)

	clusterCIDR, err = getPodCIDR(clusterCIDR, networkDetails)
	exitOnError("Error determining the pod CIDR", err)

	status.Start("Deploying Submariner")
	err = submarinercr.Ensure(config, OperatorNamespace, populateSubmarinerSpec(subctlData))
	status.End(err == nil)
	exitOnError("Error deploying Submariner", err)
}

func getNetworkDetails(config *rest.Config) *network.ClusterNetwork {

	dynClient, clientSet, err := getClients(config)
	exitOnError("Unable to set the Kubernetes cluster connection up", err)
	networkDetails, err := network.Discover(dynClient, clientSet)
	if err != nil {
		fmt.Printf("Error trying to discover network details: %s\n", err)
	} else if networkDetails != nil {
		networkDetails.Show()
	}
	return networkDetails
}

func getPodCIDR(clusterCIDR string, nd *network.ClusterNetwork) (string, error) {
	if clusterCIDR != "" {
		if nd != nil && len(nd.PodCIDRs) > 0 && nd.PodCIDRs[0] != clusterCIDR {
			fmt.Printf("WARNING: your provided cluster CIDR for the pods (%s) does not match discovered (%s)\n",
				clusterCIDR, nd.PodCIDRs[0])
		}
		return clusterCIDR, nil
	} else if len(nd.PodCIDRs) > 0 {
		return nd.PodCIDRs[0], nil
	} else {
		return askForCIDR("Pod")
	}
}

func getServiceCIDR(serviceCIDR string, nd *network.ClusterNetwork) (string, error) {
	if serviceCIDR != "" {
		if nd != nil && len(nd.ServiceCIDRs) > 0 && nd.ServiceCIDRs[0] != serviceCIDR {
			fmt.Printf("WARNING: your provided service CIDR (%s) does not match discovered (%s)\n",
				serviceCIDR, nd.ServiceCIDRs[0])
		}
		return serviceCIDR, nil
	} else if nd != nil && len(nd.ServiceCIDRs) > 0 {
		return nd.ServiceCIDRs[0], nil
	} else {
		return askForCIDR("ClusterIP service")
	}
}

func askForCIDR(name string) (string, error) {
	var qs = []*survey.Question{{
		Name:     "cidr",
		Prompt:   &survey.Input{Message: fmt.Sprintf("What's the %s CIDR for your cluster?", name)},
		Validate: survey.Required,
	}}

	answers := struct {
		Cidr string
	}{}

	err := survey.Ask(qs, &answers)
	if err != nil {
		return "", err
	} else {
		return answers.Cidr, nil
	}
}

func isValidClusterID(clusterID string) (bool, error) {
	// Make sure the clusterid is a valid DNS-1123 string
	if match, _ := regexp.MatchString("^[a-z0-9][a-z0-9.-]*[a-z0-9]$", clusterID); !match {
		return false, fmt.Errorf("Cluster IDs must be valid DNS-1123 names, with only lowercase alphanumerics,\n"+
			"'.' or '-' (and the first and last characters must be alphanumerics).\n"+
			"%s doesn't meet these requirements\n", clusterID)
	}
	return true, nil
}

func populateSubmarinerSpec(subctlData *datafile.SubctlData) submariner.SubmarinerSpec {
	brokerURL := subctlData.BrokerURL
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	if len(repository) == 0 {
		// Default repository
		// This is handled in the operator after 0.0.1 (of the operator)
		repository = "quay.io/submariner"
	}

	if len(imageVersion) == 0 {
		// Default engine version
		// This is handled in the operator after 0.0.1 (of the operator)
		imageVersion = "0.0.3"
	}

	submarinerSpec := submariner.SubmarinerSpec{
		Repository:               repository,
		Version:                  imageVersion,
		CeIPSecNATTPort:          nattPort,
		CeIPSecIKEPort:           ikePort,
		CeIPSecDebug:             ipsecDebug,
		CeIPSecPSK:               base64.StdEncoding.EncodeToString(subctlData.IPSecPSK.Data["psk"]),
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(subctlData.ClientToken.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(subctlData.ClientToken.Data["namespace"]),
		BrokerK8sApiServerToken:  string(subctlData.ClientToken.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		Broker:                   "k8s",
		NatEnabled:               !disableNat,
		Debug:                    submarinerDebug,
		ColorCodes:               colorCodes,
		ClusterID:                clusterID,
		ServiceCIDR:              serviceCIDR,
		ClusterCIDR:              clusterCIDR,
		Namespace:                SubmarinerNamespace,
	}

	return submarinerSpec
}
