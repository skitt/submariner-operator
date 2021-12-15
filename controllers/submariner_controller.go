/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	submarineriov1alpha1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	submv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
)

// SubmarinerReconciler reconciles a Submariner object
type SubmarinerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=submariner.io,resources=submariners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=submariner.io,resources=submariners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=submariner.io,resources=submariners/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Submariner object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *SubmarinerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubmarinerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watch for changes to the gateway status in the same namespace
	mapFn := handler.MapFunc(
		func(object client.Object) []ctrl.Request {
			return []ctrl.Request{
				{NamespacedName: types.NamespacedName{
					Name:      "submariner",
					Namespace: object.GetNamespace(),
				}},
			}
		})

	return ctrl.NewControllerManagedBy(mgr).
		Named("submariner-controller").
		// Watch for changes to primary resource Submariner
		For(&submarineriov1alpha1.Submariner{}).
		// Watch for changes to secondary resource DaemonSets and requeue the owner Submariner
		Owns(&appsv1.DaemonSet{}).
		Watches(&source.Kind{Type: &submv1.Gateway{}}, handler.EnqueueRequestsFromMapFunc(mapFn)).
		Complete(r)
}
