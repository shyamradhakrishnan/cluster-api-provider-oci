/*
 *
 * Copyright (c) 2022, Oracle and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * /
 *
 */

package controllers

import (
	"context"
	"fmt"
	expV1beta1 "github.com/oracle/cluster-api-provider-oci/exp/api/v1beta1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	expclusterv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"

	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	"github.com/oracle/cluster-api-provider-oci/cloud/scope"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// OCIMachinePoolReconciler reconciles a OCIMachinePool object
type OCIMachinePoolReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
	ClientProvider *scope.ClientProvider
	Region         string
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimachinepools,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimachinepools/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimachinepools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the machinepool closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *OCIMachinePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	fmt.Println("---- Got reconciliation event for machine pool")
	logger.Info("Got reconciliation event for machine pool")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCIMachinePoolReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	fmt.Println("----  SetupWithManager for machine pool")
	//return ctrl.NewControllerManagedBy(mgr).
	//	WithOptions(options).
	//	For(&expV1beta1.OCIMachinePool{}).
	//	Watches(
	//		&source.Kind{Type: &expclusterv1.MachinePool{}},
	//		handler.EnqueueRequestsFromMapFunc(machinePoolToInfrastructureMapFunc(infrastructurev1beta1.
	//			GroupVersion.WithKind(expV1beta1.OCIMachinePoolKind))),
	//	).
	//	WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
	//	Complete(r)
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&expV1beta1.OCIMachinePool{}).
		Watches(
			&source.Kind{Type: &expclusterv1.MachinePool{}},
			handler.EnqueueRequestsFromMapFunc(machinePoolToInfrastructureMapFunc(infrastructurev1beta1.
				GroupVersion.WithKind(expV1beta1.OCIMachinePoolKind))),
		).
		//Watches(
		//	&source.Kind{Type: &infrastructurev1beta1.OCICluster{}},
		//	handler.EnqueueRequestsFromMapFunc(r.OCIClusterToOCIMachines(ctx)),
		//).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))). // don't queue reconcile if resource is paused
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating OCIMachinePool controller")
	}

	clusterToObjectFunc, err := util.ClusterToObjectsMapper(r.Client, &infrastructurev1beta1.OCIMachineList{}, mgr.GetScheme())
	if err != nil {
		return errors.Wrapf(err, "failed to create mapper for Cluster to OCIMachinePool")
	}

	// Add a watch on clusterv1.Cluster object for unpause & ready notifications.
	if err := c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
		predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

func machinePoolToInfrastructureMapFunc(gvk schema.GroupVersionKind) handler.MapFunc {
	fmt.Println("------ machinePoolToInfrastructureMapFunc")
	return func(o client.Object) []reconcile.Request {
		fmt.Println("------ machinePoolToInfrastructureMapFunc inner func")
		m, ok := o.(*expclusterv1.MachinePool)
		if !ok {
			panic(fmt.Sprintf("Expected a MachinePool but got a %T", o))
		}

		gk := gvk.GroupKind()
		fmt.Println("------ machinePoolToInfrastructureMapFunc gk: ", gk)
		// Return early if the GroupKind doesn't match what we expect
		infraGK := m.Spec.Template.Spec.InfrastructureRef.GroupVersionKind().GroupKind()
		if gk != infraGK {
			return nil
		}

		fmt.Println("------ machinePoolToInfrastructureMapFunc name: ", m.Spec.Template.Spec.InfrastructureRef.Name)
		return []reconcile.Request{
			{
				NamespacedName: client.ObjectKey{
					Namespace: m.Namespace,
					Name:      m.Spec.Template.Spec.InfrastructureRef.Name,
				},
			},
		}
	}
}

func (r *OCIMachinePoolReconciler) OCIClusterToOCIMachines(ctx context.Context) handler.MapFunc {
	fmt.Println("------ OCIClusterToOCIMachines")
	log := ctrl.LoggerFrom(ctx)
	return func(o client.Object) []ctrl.Request {
		result := []ctrl.Request{}

		c, ok := o.(*infrastructurev1beta1.OCICluster)
		if !ok {
			log.Error(errors.Errorf("expected a OCICluster but got a %T", o), "failed to get OCIMachine for OCICluster")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			return result
		case err != nil:
			log.Error(err, "failed to get owning cluster")
			return result
		}

		labels := map[string]string{clusterv1.ClusterLabelName: cluster.Name}
		machinePoolList := &expV1beta1.OCIMachinePoolList{}
		if err := r.List(ctx, machinePoolList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.Error(err, "failed to list MachinePoolss")
			return nil
		}
		for _, m := range machinePoolList.Items {
			fmt.Println("------ machinepool item", m)
			//if m.Spec.Name == "" {
			//	continue
			//}
			//name := client.ObjectKey{Namespace: m.Namespace, Name: m.in.Name}
			//fmt.Println("------ machinepool name", name)
			//result = append(result, ctrl.Request{NamespacedName: name})
		}

		return result
	}
}
