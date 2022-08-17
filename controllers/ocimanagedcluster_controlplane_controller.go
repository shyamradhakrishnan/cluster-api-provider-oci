/*
Copyright (c) 2021, 2022 Oracle and/or its affiliates.

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
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	"github.com/oracle/cluster-api-provider-oci/cloud/scope"
	"github.com/oracle/oci-go-sdk/v63/containerengine"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	DefaultMappingTimeout = 60 * time.Second
)

// OCIManagedClusterControlPlaneReconciler reconciles a OciCluster object
type OCIManagedClusterControlPlaneReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Recorder       record.EventRecorder
	Region         string
	ClientProvider *scope.ClientProvider
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimanagedcontrolplanes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimanagedcontrolplanes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimanagedcontrolplanes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the machine closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *OCIManagedClusterControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues(scope.OCIManagedClusterKind, req.NamespacedName)

	logger.Info("Inside cluster reconciler")

	// Fetch the OCICluster instance
	controlPlane := &infrastructurev1beta1.OCIManagedControlPlane{}
	err := r.Get(ctx, req.NamespacedName, controlPlane)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, controlPlane.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		r.Recorder.Eventf(controlPlane, corev1.EventTypeNormal, "OwnerRefNotSet", "Cluster Controller has not yet set OwnerRef")
		logger.Info("Cluster Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, controlPlane) {
		r.Recorder.Eventf(controlPlane, corev1.EventTypeNormal, "ClusterPaused", "Cluster is paused")
		logger.Info("OCICluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	ociManagedCluster := &infrastructurev1beta1.OCIManagedCluster{}
	ociClusterName := client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, ociClusterName, ociManagedCluster); err != nil {
		logger.Info("Cluster is not available yet")
		r.Recorder.Eventf(controlPlane, corev1.EventTypeWarning, "ClusterNotAvailable", "Cluster is not available yet")
		return ctrl.Result{}, nil
	}

	if !ociManagedCluster.Status.Ready {
		logger.Info("Cluster infrastructure is not ready")
		r.Recorder.Eventf(controlPlane, corev1.EventTypeWarning, "ClusterInfrastructureNotReady", "Cluster infrastructure is not ready")
		return ctrl.Result{}, nil
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, ociManagedCluster) {
		r.Recorder.Eventf(controlPlane, corev1.EventTypeNormal, "ClusterPaused", "Cluster is paused")
		logger.Info("OCICluster or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	regionOverride := r.Region
	if len(ociManagedCluster.Spec.Region) > 0 {
		regionOverride = ociManagedCluster.Spec.Region
	}
	if len(regionOverride) <= 0 {
		return ctrl.Result{}, errors.New("OCIManagedControlPlane Region can't be nil")
	}

	clients, err := r.ClientProvider.GetOrBuildClient(regionOverride)
	if err != nil {
		logger.Error(err, "Couldn't get the clients for region")
	}

	helper, err := patch.NewHelper(controlPlane, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to init patch helper")
	}

	// Always close the scope when exiting this function so we can persist any OCICluster changes.
	defer func() {
		logger.Info("Closing cluster scope")
		conditions.SetSummary(controlPlane)

		if err := helper.Patch(ctx, controlPlane); err != nil && reterr == nil {
			reterr = err
		}
	}()

	var controlPlaneScope *scope.ControlPlaneScope

	clusterBase := scope.OCIManagedCluster{
		OCIManagedCluster: ociManagedCluster,
	}
	controlPlaneScope, err = scope.NewControlPlaneScope(scope.ControlPlaneScopeParams{
		Client:                 r.Client,
		Logger:                 &logger,
		Cluster:                cluster,
		OCIClusterBase:         clusterBase,
		ClientProvider:         r.ClientProvider,
		ContainerEngineClient:  clients.ContainerEngineClient,
		Region:                 regionOverride,
		OCIManagedControlPlane: controlPlane,
		BaseClient:             clients.BaseClient,
	})

	if err != nil {
		return ctrl.Result{}, err
	}

	// Handle deleted clusters
	if !controlPlane.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, logger, controlPlaneScope, controlPlane)
	}

	if err != nil {
		return reconcile.Result{}, errors.Errorf("failed to create scope: %+v", err)
	} else {
		return r.reconcile(ctx, logger, controlPlaneScope, controlPlane)
	}

}

func (r *OCIManagedClusterControlPlaneReconciler) reconcileComponent(ctx context.Context, controlPlaneScope *scope.ControlPlaneScope, controlPlane *v1beta1.OCIManagedControlPlane,
	reconciler func(context.Context) error,
	componentName string, failReason string, readyEventtype string) error {

	err := reconciler(ctx)
	if err != nil {
		r.Recorder.Event(controlPlane, corev1.EventTypeWarning, "ReconcileError", errors.Wrapf(err,
			fmt.Sprintf("failed to reconcile %s", componentName)).Error())
		conditions.MarkFalse(controlPlane, infrastructurev1beta1.ClusterReadyCondition, failReason, clusterv1.ConditionSeverityError, "")
		return errors.Wrapf(err, "failed to reconcile %s for OCICluster %s/%s", componentName, controlPlane.Namespace,
			controlPlane.Name)
	}

	trimmedComponentName := strings.ReplaceAll(componentName, " ", "")
	r.Recorder.Eventf(controlPlane, corev1.EventTypeNormal, readyEventtype,
		fmt.Sprintf("%s is in ready state", trimmedComponentName))

	return nil
}

func (r *OCIManagedClusterControlPlaneReconciler) reconcile(ctx context.Context, logger logr.Logger, controlPlaneScope *scope.ControlPlaneScope, cluster *infrastructurev1beta1.OCIManagedControlPlane) (ctrl.Result, error) {
	// If the OCICluster doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(cluster, infrastructurev1beta1.ControlPlaneFinalizer)

	controlPlane, err := controlPlaneScope.GetOrCreateControlPlane(ctx)
	if err != nil {
		r.Recorder.Event(cluster, corev1.EventTypeWarning, "ReconcileError", errors.Wrapf(err, "Failed to reconcile OCIManagedcontrolPlane").Error())
		return ctrl.Result{}, errors.Wrapf(err, "failed to reconcile OCI Managed Control Plane %s/%s", cluster.Namespace, controlPlaneScope.GetClusterName())
	}

	// Proceed to reconcile the DOMachine state.
	switch controlPlane.LifecycleState {
	case containerengine.ClusterLifecycleStateCreating:
		controlPlaneScope.Info("Control plane is pending")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	case containerengine.ClusterLifecycleStateUpdating:
		controlPlaneScope.Info("Control plane is updating")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	case containerengine.ClusterLifecycleStateActive:
		controlPlaneScope.Info("Instance is active")
		if controlPlaneScope.IsControlPlaneEndpointSubnetPrivate() {
			cluster.Spec.ControlPlaneEndpoint = &clusterv1.APIEndpoint{
				Host: *controlPlane.Endpoints.PrivateEndpoint,
				Port: 6443,
			}
		} else {
			cluster.Spec.ControlPlaneEndpoint = &clusterv1.APIEndpoint{
				Host: *controlPlane.Endpoints.PublicEndpoint,
				Port: 6443,
			}
		}
		controlPlaneScope.OCIManagedControlPlane.Status.Ready = true
		err := controlPlaneScope.ReconcileKubeconfig(ctx, controlPlane)
		if err != nil {
			return ctrl.Result{}, err
		}
	default:
		conditions.MarkFalse(cluster, infrastructurev1beta1.InstanceReadyCondition, infrastructurev1beta1.InstanceProvisionFailedReason, clusterv1.ConditionSeverityError, "")
		r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcileError",
			"Cluster has invalid lifecycle state %s", controlPlane.LifecycleState)
		return reconcile.Result{}, errors.New(fmt.Sprintf("Cluster  has invalid lifecycle state %s", controlPlane.LifecycleState))
	}
	return reconcile.Result{RequeueAfter: 300 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCIManagedClusterControlPlaneReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	ociManagedClusterMapper, err := OCIManagedClusterToOCIManagedControlPlaneMapper(ctx, r.Client, log)
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&infrastructurev1beta1.OCIManagedControlPlane{}).
		Watches(
			&source.Kind{Type: &infrastructurev1beta1.OCIManagedCluster{}},
			handler.EnqueueRequestsFromMapFunc(ociManagedClusterMapper),
		).
		Build(r)
	if err != nil {
		return errors.Wrapf(err, "error creating controller")
	}

	if err = c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(ClusterToOCIManagedControlPlaneMapper()),
		predicates.ResourceNotPausedAndHasFilterLabel(log, ""),
	); err != nil {
		return errors.Wrapf(err, "failed adding a watch for ready clusters")
	}

	return nil
}

// ClusterToInfrastructureMapFunc returns a handler.ToRequestsFunc that watches for
// Cluster events and returns reconciliation requests for an infrastructure provider object.
func (r *OCIManagedClusterControlPlaneReconciler) clusterToInfrastructureMapFunc(ctx context.Context, log logr.Logger) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		c, ok := o.(*clusterv1.Cluster)
		if !ok {
			return nil
		}

		// Make sure the ref is set
		if c.Spec.InfrastructureRef == nil {
			log.V(4).Info("Cluster does not have an InfrastructureRef, skipping mapping.")
			return nil
		}

		if c.Spec.InfrastructureRef.GroupVersionKind().Kind != "OCIManagedCluster" {
			log.V(4).Info("Cluster has an InfrastructureRef for a different type, skipping mapping.")
			return nil
		}

		ociCluster := &infrastructurev1beta1.OCIManagedCluster{}
		key := types.NamespacedName{Namespace: c.Spec.InfrastructureRef.Namespace, Name: c.Spec.InfrastructureRef.Name}

		if err := r.Get(ctx, key, ociCluster); err != nil {
			log.V(4).Error(err, "Failed to get OCI cluster")
			return nil
		}

		if annotations.IsExternallyManaged(ociCluster) {
			log.V(4).Info("OCICluster is externally managed, skipping mapping.")
			return nil
		}

		log.V(4).Info("Adding request.", "ociCluster", c.Spec.InfrastructureRef.Name)

		return []reconcile.Request{
			{
				NamespacedName: client.ObjectKey{
					Namespace: c.Namespace,
					Name:      c.Spec.InfrastructureRef.Name,
				},
			},
		}
	}
}

func (r *OCIManagedClusterControlPlaneReconciler) reconcileDelete(ctx context.Context, logger logr.Logger,
	controlPlaneScope *scope.ControlPlaneScope, controlPlane *infrastructurev1beta1.OCIManagedControlPlane) (ctrl.Result, error) {
	controlPlaneScope.Info("Handling deleted OCiManagedControlPlane")

	cluster, err := controlPlaneScope.GetOKECluster(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		controlPlaneScope.Info("Cluster is not found, may have been deleted")
		controllerutil.RemoveFinalizer(controlPlane, v1beta1.ControlPlaneFinalizer)
		return reconcile.Result{}, nil
	}

	switch cluster.LifecycleState {
	case containerengine.ClusterLifecycleStateDeleting:
		controlPlaneScope.Info("Cluster is terminating")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	case containerengine.ClusterLifecycleStateDeleted:
		controllerutil.RemoveFinalizer(controlPlane, v1beta1.ControlPlaneFinalizer)
		controlPlaneScope.Info("Cluster is deleted")
		return reconcile.Result{}, nil
	default:
		if err := controlPlaneScope.DeleteCluster(ctx); err != nil {
			controlPlaneScope.Error(err, "Error deleting cluster")
			return ctrl.Result{}, errors.Wrapf(err, "error deleting cluster %s", controlPlaneScope.GetClusterName())
		}
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}
}

func OCIManagedClusterToOCIManagedControlPlaneMapper(ctx context.Context, c client.Client, log logr.Logger) (handler.MapFunc, error) {
	return func(o client.Object) []ctrl.Request {
		ctx, cancel := context.WithTimeout(ctx, DefaultMappingTimeout)
		defer cancel()

		ociCluster, ok := o.(*infrastructurev1beta1.OCIManagedCluster)
		if !ok {
			log.Error(errors.Errorf("expected an OCIManagedCluster, got %T instead", o), "failed to map AzureManagedCluster")
			return nil
		}

		log = log.WithValues("OCIManagedCluster", ociCluster.Name, "Namespace", ociCluster.Namespace)

		// Don't handle deleted OCIManagedClusters
		if !ociCluster.ObjectMeta.DeletionTimestamp.IsZero() {
			log.V(4).Info("OCIManagedCluster has a deletion timestamp, skipping mapping.")
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, c, ociCluster.ObjectMeta)
		if err != nil {
			log.Error(err, "failed to get the owning cluster")
			return nil
		}

		if cluster == nil {
			log.Error(err, "cluster has not set owner ref yet")
			return nil
		}

		ref := cluster.Spec.ControlPlaneRef
		if ref == nil || ref.Name == "" {
			return nil
		}

		return []ctrl.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: ref.Namespace,
					Name:      ref.Name,
				},
			},
		}
	}, nil
}

func ClusterToOCIManagedControlPlaneMapper() handler.MapFunc {
	return func(o client.Object) []ctrl.Request {
		cluster, ok := o.(*clusterv1.Cluster)
		if !ok {
			return nil
		}

		ref := cluster.Spec.ControlPlaneRef
		if ref == nil || ref.Name == "" {
			return nil
		}

		return []ctrl.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: ref.Namespace,
					Name:      ref.Name,
				},
			},
		}
	}
}
