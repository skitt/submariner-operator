/*
© 2021 Red Hat, Inc. and others.

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
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	smClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

var validatePodsCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Validate the submariner deployment",
	Long: "This command checks that the submariner components are properly deployed and running" +
		" with no overlapping CIDRs.",
	Run: validateSubmarinerDeployment,
}

func init() {
	validateCmd.AddCommand(validatePodsCmd)
}

func validateSubmarinerDeployment(cmd *cobra.Command, args []string) {
	configs, err := getMultipleRestConfigs(kubeConfig, kubeContext)
	exitOnError("error getting REST config for cluster", err)

	for _, item := range configs {
		submariner := getSubmarinerResource(item.config)
		if submariner == nil {
			status.QueueWarningMessage(submMissingMessage)
			status.End(cli.Success)
			return
		}

		checkPods(item, submariner, OperatorNamespace)
		checkOverlappingCIDRs(item, submariner)
	}
}

func checkOverlappingCIDRs(item restConfig, submariner *v1alpha1.Submariner) {
	var message string
	submarinerClient, err := smClientset.NewForConfig(item.config)
	exitOnError("Unable to get the Submariner client", err)

	localClusterName := submariner.Status.ClusterID
	endpointList, err := submarinerClient.SubmarinerV1().Endpoints(submariner.Namespace).List(metav1.ListOptions{})
	if err != nil {
		exitOnError("Error while listing the endpoints", err)
	}

	status.Start("Verifying if cluster CIDRs overlap")

	var localEndpoint submarinerv1.Endpoint
	overlappingClusters := make(map[string]submarinerv1.Endpoint)
	for i, source := range endpointList.Items {
		if localClusterName == source.Spec.ClusterID {
			localEndpoint = source
		}

		for _, dest := range endpointList.Items[i+1:] {
			// Currently we dont support multiple endpoints in a cluster, hence return an error.
			// When the corresponding support is added, this check needs to be updated.
			if source.Spec.ClusterID == dest.Spec.ClusterID {
				message = fmt.Sprintf("Looks like some stale endpoints are present for cluster %q",
					source.Spec.ClusterID)
				status.QueueFailureMessage(message)
				status.End(cli.Failure)
				break
			}

			for _, subnet := range dest.Spec.Subnets {
				overlap, err := util.IsOverlappingCIDR(source.Spec.Subnets, subnet)
				if err != nil {
					// Ideally this case will never hit, as the subnets are valid CIDRs
					message = fmt.Sprintf("Error parsing the CIDR %s in cluster %q", err, dest.Spec.ClusterID)
					status.QueueFailureMessage(message)
					status.End(cli.Failure)
					break
				}

				if overlap {
					overlappingClusters[source.Spec.ClusterID] = source
					overlappingClusters[dest.Spec.ClusterID] = dest
				}
			}
		}
	}

	if len(overlappingClusters) > 0 {
		if _, exists := overlappingClusters[localClusterName]; exists {
			delete(overlappingClusters, localClusterName)
			message = fmt.Sprintf("localCluster %q with CIDRs %q overlaps with clusters %#v",
				localClusterName, localEndpoint.Spec.Subnets, overlappingClusters)
		} else {
			message = fmt.Sprintf("localCluster %q does not have overlapping CIDRs with other member clusters. "+
				"However, the following submariner member clusters have overlapping CIDRs %#v", localClusterName,
				getClusterDetails(overlappingClusters))
		}
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
	}

	status.QueueSuccessMessage("Clusters do not have overlapping CIDRs")
	status.End(cli.Success)
}

func getClusterDetails(overlappingClusters map[string]submarinerv1.Endpoint) string {
	output := []string{}
	for _, cluster := range overlappingClusters {
		output = append(output, fmt.Sprintf("clusterID: %s, subnets: %s", cluster.Spec.ClusterID, cluster.Spec.Subnets))
	}
	return strings.Join(output, " ; ")
}

func checkPods(item restConfig, submariner *v1alpha1.Submariner, operatorNamespace string) {
	message := fmt.Sprintf("Validating submariner pods in %q", item.clusterName)
	status.Start(message)
	fmt.Println()

	kubeClientSet, err := kubernetes.NewForConfig(item.config)

	if err != nil {
		exitOnError("error creating Kubernetes client", err)
	}

	if !CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-gateway") {
		return
	}

	if !CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-routeagent") {
		return
	}

	// Check if service-discovery components are deployed and running if enabled
	if submariner.Spec.ServiceDiscoveryEnabled {
		// Check lighthouse-agent
		if !CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-agent") {
			return
		}

		// Check lighthouse-coreDNS
		if !CheckDeployment(kubeClientSet, operatorNamespace, "submariner-lighthouse-coredns") {
			return
		}
	}
	// Check if globalnet components are deployed and running if enabled
	if submariner.Spec.GlobalCIDR != "" {
		if !CheckDaemonset(kubeClientSet, operatorNamespace, "submariner-globalnet") {
			return
		}
	}

	message = "All Submariner pods are up and running"
	status.QueueSuccessMessage(message)
	status.End(cli.Success)
}

func CheckDeployment(k8sClient kubernetes.Interface, namespace, deploymentName string) bool {
	deployment, err := k8sClient.AppsV1().Deployments(namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("Error obtaining Deployment %q: %v", deploymentName, err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	var replicas int32 = 1
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	if deployment.Status.AvailableReplicas != replicas {
		message := fmt.Sprintf("The desired number of replicas for Deployment %q (%d)"+
			" does not match the actual number running (%d)", deploymentName, replicas,
			deployment.Status.AvailableReplicas)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	return true
}

func CheckDaemonset(k8sClient kubernetes.Interface, namespace, daemonSetName string) bool {
	daemonSet, err := k8sClient.AppsV1().DaemonSets(namespace).Get(daemonSetName, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("Error obtaining Daemonset %q: %v", daemonSetName, err)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	if daemonSet.Status.CurrentNumberScheduled != daemonSet.Status.DesiredNumberScheduled {
		message := fmt.Sprintf("The desired number of running pods for DaemonSet %q (%d)"+
			" does not match the actual number (%d)", daemonSetName, daemonSet.Status.DesiredNumberScheduled,
			daemonSet.Status.CurrentNumberScheduled)
		status.QueueFailureMessage(message)
		status.End(cli.Failure)
		return false
	}

	return true
}