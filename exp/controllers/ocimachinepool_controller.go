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
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	infrav1exp "github.com/oracle/cluster-api-provider-oci/exp/api/v1beta1"
	"github.com/oracle/oci-go-sdk/v63/common"
	"github.com/oracle/oci-go-sdk/v63/core"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	expclusterv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	"github.com/oracle/cluster-api-provider-oci/cloud/scope"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machinepools;machinepools/status,verbs=get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ocimachinepools/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the machinepool closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *OCIMachinePoolReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)
	logger.Info("Got reconciliation event for machine pool")

	// Fetch the OCIMachinePool.
	ociMachinePool := &infrav1exp.OCIMachinePool{}
	err := r.Get(ctx, req.NamespacedName, ociMachinePool)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the CAPI MachinePool
	machinePool, err := getOwnerMachinePool(ctx, r.Client, ociMachinePool.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machinePool == nil {
		logger.Info("MachinePool Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}
	logger = logger.WithValues("machinePool", machinePool.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machinePool.ObjectMeta)
	if err != nil {
		logger.Info("MachinePool is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}
	logger = logger.WithValues("cluster", cluster.Name)

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, ociMachinePool) {
		logger.Info("OCIMachinePool or linked Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	ociCluster := &infrastructurev1beta1.OCICluster{}
	ociClusterName := client.ObjectKey{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}

	if err := r.Client.Get(ctx, ociClusterName, ociCluster); err != nil {
		logger.Info("Cluster is not available yet")
		r.Recorder.Eventf(ociMachinePool, corev1.EventTypeWarning, "ClusterNotAvailable", "Cluster is not available yet")
		logger.V(2).Info("OCICluster is not available yet")
		return ctrl.Result{}, nil
	}

	regionOverride := r.Region
	if len(ociCluster.Spec.Region) > 0 {
		regionOverride = ociCluster.Spec.Region
	}
	if len(regionOverride) <= 0 {
		return ctrl.Result{}, errors.New("OCIMachinePoolReconciler Region can't be nil")
	}

	clients, err := r.ClientProvider.GetOrBuildClient(regionOverride)
	if err != nil {
		logger.Error(err, "Couldn't get the clients for region")
	}

	// Create the machine pool scope
	machinePoolScope, err := scope.NewMachinePoolScope(scope.MachinePoolScopeParams{
		Client:                    r.Client,
		ComputeClient:             clients.ComputeClient,
		ComputeManagementClient:   clients.ComputeManagementClient,
		Logger:                    &logger,
		Cluster:                   cluster,
		OCICluster:                ociCluster,
		MachinePool:               machinePool,
		OCIMachinePool:            ociMachinePool,
		VCNClient:                 clients.VCNClient,
		NetworkLoadBalancerClient: clients.LoadBalancerClient,
	})
	if err != nil {
		return ctrl.Result{}, errors.Errorf("failed to create scope: %+v", err)
	}

	// Always close the scope when exiting this function so we can persist any GCPMachine changes.
	defer func() {
		logger.Info("---- closing scope")
		if err := machinePoolScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Handle deleted machines
	if !ociMachinePool.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machinePoolScope)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, logger, machinePoolScope)
}

// SetupWithManager sets up the controller with the Manager.
func (r *OCIMachinePoolReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	logger := log.FromContext(ctx)
	fmt.Println("----  SetupWithManager for machine pool")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1exp.OCIMachinePool{}).
		Watches(
			&source.Kind{Type: &expclusterv1.MachinePool{}},
			handler.EnqueueRequestsFromMapFunc(machinePoolToInfrastructureMapFunc(expclusterv1.
				GroupVersion.WithKind(scope.OCIMachinePoolKind), logger)),
		).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Complete(r)
	//c, err := ctrl.NewControllerManagedBy(mgr).
	//	For(&infrav1exp.OCIMachinePool{}).
	//	Watches(
	//		&source.Kind{Type: &expclusterv1.MachinePool{}},
	//		handler.EnqueueRequestsFromMapFunc(machinePoolToInfrastructureMapFunc(infrav1exp.
	//			GroupVersion.WithKind(infrav1exp.OCIMachinePoolKind))),
	//	).
	//	//Watches(
	//	//	&source.Kind{Type: &infrastructurev1beta1.OCICluster{}},
	//	//	handler.EnqueueRequestsFromMapFunc(r.OCIClusterToOCIMachines(ctx)),
	//	//).
	//	WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))). // don't queue reconcile if resource is paused
	//	Build(r)
	//if err != nil {
	//	return errors.Wrapf(err, "error creating OCIMachinePool controller")
	//}
	//
	//clusterToObjectFunc, err := util.ClusterToObjectsMapper(r.Client, &infrav1exp.OCIMachinePoolList{}, mgr.GetScheme())
	//if err != nil {
	//	return errors.Wrapf(err, "failed to create mapper for Cluster to OCIMachinePool")
	//}
	//
	//// Add a watch on clusterv1.Cluster object for unpause & ready notifications.
	//if err := c.Watch(
	//	&source.Kind{Type: &clusterv1.Cluster{}},
	//	handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
	//	predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
	//); err != nil {
	//	return errors.Wrapf(err, "failed adding a watch for ready clusters")
	//}
	//
	//fmt.Println("------ SetupWithManager for machine pool done! -----")
	//return nil
}

func machinePoolToInfrastructureMapFunc(gvk schema.GroupVersionKind, logger logr.Logger) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		m, ok := o.(*expclusterv1.MachinePool)
		if !ok {
			panic(fmt.Sprintf("Expected a MachinePool but got a %T", o))
		}

		gk := gvk.GroupKind()
		// Return early if the GroupKind doesn't match what we expect
		infraGK := m.Spec.Template.Spec.InfrastructureRef.GroupVersionKind().GroupKind()
		if gk != infraGK {
			logger.V(4).Info("gk does not match", "gk", gk, "infraGK", infraGK)
			return nil
		}

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

// getOwnerMachinePool returns the MachinePool object owning the current resource.
func getOwnerMachinePool(ctx context.Context, c client.Client, obj metav1.ObjectMeta) (*expclusterv1.MachinePool, error) {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind != "MachinePool" {
			continue
		}
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		if gv.Group == expclusterv1.GroupVersion.Group {
			return getMachinePoolByName(ctx, c, obj.Namespace, ref.Name)
		}
	}
	return nil, nil
}

// getMachinePoolByName finds and return a Machine object using the specified params.
func getMachinePoolByName(ctx context.Context, c client.Client, namespace, name string) (*expclusterv1.MachinePool, error) {
	m := &expclusterv1.MachinePool{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := c.Get(ctx, key, m); err != nil {
		return nil, err
	}
	return m, nil
}

func (r *OCIMachinePoolReconciler) reconcileNormal(ctx context.Context, logger logr.Logger, machinePoolScope *scope.MachinePoolScope) (ctrl.Result, error) {
	machinePoolScope.Info("Handling reconcile OCIMachinePool")

	// If the OCIMachinePool is in an error state, return early.
	// If the OCIMachine is in an error state, return early.
	if machinePoolScope.HasFailed() {
		machinePoolScope.Info("Error state detected, skipping reconciliation")

		return ctrl.Result{}, nil
	}

	// If the OCIMachinePool doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machinePoolScope.OCIMachinePool, infrav1exp.MachinePoolFinalizer)
	// Register the finalizer immediately to avoid orphaning OCI resources on delete
	if err := machinePoolScope.PatchObject(ctx); err != nil {
		return reconcile.Result{}, err
	}

	// TODO: add back in after testing
	if !machinePoolScope.Cluster.Status.InfrastructureReady {
		logger.Info("Cluster infrastructure is not ready yet")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// TODO: add back in after testing
	//// Make sure bootstrap data is available and populated.
	if machinePoolScope.MachinePool.Spec.Template.Spec.Bootstrap.DataSecretName == nil {
		r.Recorder.Event(machinePoolScope.OCIMachinePool, corev1.EventTypeNormal, infrastructurev1beta1.WaitingForBootstrapDataReason, "Bootstrap data secret reference is not yet available")
		conditions.MarkFalse(machinePoolScope.OCIMachinePool, infrastructurev1beta1.InstanceReadyCondition, infrastructurev1beta1.WaitingForBootstrapDataReason, clusterv1.ConditionSeverityInfo, "")
		logger.Info("Bootstrap data secret reference is not yet available")
		return ctrl.Result{}, nil
	}

	// get or create the InstanceConfiguration
	// https://docs.oracle.com/en-us/iaas/api/#/en/iaas/20160918/InstanceConfiguration/
	if err := r.reconcileLaunchTemplate(ctx, machinePoolScope); err != nil {
		r.Recorder.Eventf(machinePoolScope.OCIMachinePool, corev1.EventTypeWarning, "FailedLaunchTemplateReconcile", "Failed to reconcile launch template: %v", err)
		machinePoolScope.Error(err, "failed to reconcile launch template")
		return ctrl.Result{}, err
	}

	logger.Info("---- after reconcileLaunchTemplate", "id", machinePoolScope.OCIMachinePool.Status.InstanceConfigurationId)

	// set the LaunchTemplateReady condition
	conditions.MarkTrue(machinePoolScope.OCIMachinePool, infrav1exp.LaunchTemplateReadyCondition)

	// Find existing Instance Pool
	instancePool, err := r.findInstancePool(ctx, machinePoolScope)
	if err != nil {
		conditions.MarkUnknown(machinePoolScope.OCIMachinePool, infrav1exp.InstancePoolReadyCondition, infrav1exp.InstancePoolNotFoundReason, err.Error())
		return ctrl.Result{}, err
	}

	if instancePool == nil {
		// Create new ASG
		if _, err := r.createInstancePool(ctx, machinePoolScope); err != nil {
			conditions.MarkFalse(machinePoolScope.OCIMachinePool, infrav1exp.InstancePoolReadyCondition, infrav1exp.InstancePoolProvisionFailedReason, clusterv1.ConditionSeverityError, err.Error())
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	machinePoolScope.Info("OCI Compute Instance Pool found", "InstancePoolID", *instancePool.Id)
	machinePoolScope.OCIMachinePool.Spec.ProviderID = common.String(fmt.Sprintf("oci://%s", *instancePool.Id))

	// TODO: set to true once infra is ready
	switch instancePool.LifecycleState {
	case core.InstancePoolLifecycleStateProvisioning, core.InstancePoolLifecycleStateStarting:
		machinePoolScope.Info("Instance Pool is pending")
		conditions.MarkFalse(machinePoolScope.OCIMachinePool, infrav1exp.InstancePoolReadyCondition, infrav1exp.InstancePoolNotReadyReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	case core.InstancePoolLifecycleStateRunning:
		machinePoolScope.Info("Instance pool is active")

		// record the event only when machine goes from not ready to ready state
		r.Recorder.Eventf(machinePoolScope.OCIMachinePool, corev1.EventTypeNormal, "InstancePoolReady",
			"Instance pool is in ready state")
		conditions.MarkTrue(machinePoolScope.OCIMachinePool, infrav1exp.InstancePoolReadyCondition)
		machinePoolScope.SetReady()
	default:
		conditions.MarkFalse(machinePoolScope.OCIMachinePool, infrav1exp.InstancePoolReadyCondition, infrav1exp.InstancePoolProvisionFailedReason, clusterv1.ConditionSeverityError, "")
		machinePoolScope.SetFailureReason(capierrors.CreateMachineError)
		machinePoolScope.SetFailureMessage(errors.Errorf("Instance Pool status %q is unexpected", instancePool.LifecycleState))
		r.Recorder.Eventf(machinePoolScope.OCIMachinePool, corev1.EventTypeWarning, "ReconcileError",
			"Instance pool has invalid lifecycle state %s", instancePool.LifecycleState)
		return reconcile.Result{}, errors.New(fmt.Sprintf("instance pool  has invalid lifecycle state %s", instancePool.LifecycleState))
	}

	return ctrl.Result{}, nil
}

func (r *OCIMachinePoolReconciler) reconcileDelete(ctx context.Context, machinePoolScope *scope.MachinePoolScope) (_ ctrl.Result, reterr error) {
	machinePoolScope.Info("Handling deleted OCIMachinePool")

	return ctrl.Result{}, nil
}

func (r *OCIMachinePoolReconciler) reconcileLaunchTemplate(ctx context.Context, machinePoolScope *scope.MachinePoolScope) error {
	var instanceConfiguration *core.InstanceConfiguration
	machinePoolScope.Info("---- machinePoolScope.OCIMachinePool.Status.InstanceConfigurationId", "id", machinePoolScope.OCIMachinePool.Status.InstanceConfigurationId)
	// If the IC exists try a get
	//get by name or tag I think
	instanceConfigurationId := machinePoolScope.GetInstanceConfigurationId()

	if len(instanceConfigurationId) > 0 {
		req := core.GetInstanceConfigurationRequest{InstanceConfigurationId: common.String(instanceConfigurationId)}
		instanceConfiguration, err := machinePoolScope.ComputeManagementClient.GetInstanceConfiguration(ctx, req)
		if err == nil {
			machinePoolScope.Info("instance configuration found", "InstanceConfigurationId", instanceConfiguration.Id)
			machinePoolScope.SetInstanceConfigurationIdStatus(instanceConfigurationId)
			return machinePoolScope.PatchObject(ctx)
		}
		//	TODO: handle get by ID 404
	}

	//else try to create
	tags := machinePoolScope.GetFreeFormTags(*machinePoolScope.OCICluster)

	cloudInitData, err := machinePoolScope.GetBootstrapData()
	if err != nil {
		return err
	}

	metadata := machinePoolScope.OCIMachinePool.Spec.Metadata
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadata["user_data"] = base64.StdEncoding.EncodeToString([]byte(cloudInitData))

	// if get fails maybe try to create
	if instanceConfiguration == nil {
		//poolName := fmt.Sprintf("%s-%s", machinePoolScope.OCICluster.Name, machinePoolScope.MachinePool.Name)
		//TODO: don't hard code :-)
		subnetId := machinePoolScope.GetWorkerMachineSubnet()
		nsgId := machinePoolScope.GetWorkerMachineNSG()
		req := core.CreateInstanceConfigurationRequest{
			CreateInstanceConfiguration: core.CreateInstanceConfigurationDetails{
				CompartmentId: common.String(machinePoolScope.OCICluster.Spec.CompartmentId),
				DisplayName:   common.String(machinePoolScope.OCIMachinePool.GetName()),
				FreeformTags:  tags,
				InstanceDetails: core.ComputeInstanceDetails{
					LaunchDetails: &core.InstanceConfigurationLaunchInstanceDetails{
						CompartmentId: common.String(machinePoolScope.OCICluster.Spec.CompartmentId),
						DisplayName:   common.String(machinePoolScope.OCIMachinePool.GetName()),
						Shape:         common.String("VM.Standard.E4.Flex"),
						// ShapeConfig is required for flex
						ShapeConfig: &core.InstanceConfigurationLaunchInstanceShapeConfigDetails{
							Ocpus: common.Float32(3),
						},
						SourceDetails: core.InstanceConfigurationInstanceSourceViaImageDetails{
							ImageId: common.String("ocid1.image.oc1.phx.aaaaaaaabalujkjojovptylwmgkh4ykxrzu47gkhd6nzxshj5n7f6jyocofa"),
						},
						Metadata: metadata,
						CreateVnicDetails: &core.InstanceConfigurationCreateVnicDetails{
							SubnetId:       subnetId,
							AssignPublicIp: common.Bool(true),
							NsgIds:         []string{*nsgId},
						},
					},
				},
			},
		}
		//OpcRetryToken: common.String("EXAMPLE-opcRetryToken-Value")}

		resp, err := machinePoolScope.ComputeManagementClient.CreateInstanceConfiguration(ctx, req)
		if err != nil {
			conditions.MarkFalse(machinePoolScope.MachinePool, infrav1exp.LaunchTemplateReadyCondition, infrav1exp.LaunchTemplateCreateFailedReason, clusterv1.ConditionSeverityError, err.Error())
			machinePoolScope.Info("failed to create instance configuration")
			return err
		}

		//TODO: handle update

		fmt.Println(resp)
		fmt.Println("----- id: ", *resp.Id)
		machinePoolScope.SetInstanceConfigurationIdStatus(*resp.Id)
		machinePoolScope.Info("--- gonna patch")
		return machinePoolScope.PatchObject(ctx)
	}

	return nil
}

func (r *OCIMachinePoolReconciler) findInstancePool(ctx context.Context, machinePoolScope *scope.MachinePoolScope) (*core.InstancePool, error) {
	// TODO: fix this I don't love having to list then get. There has to be a better way
	//poolName := fmt.Sprintf("%s-%s", machinePoolScope.OCICluster.Name, machinePoolScope.MachinePool.Name)

	// Query the instance using tags.
	req := core.ListInstancePoolsRequest{
		CompartmentId: common.String(machinePoolScope.OCICluster.Spec.CompartmentId),
		DisplayName:   common.String(machinePoolScope.OCIMachinePool.GetName()),
	}
	resp, err := machinePoolScope.ComputeManagementClient.ListInstancePools(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query OCIMachinePool by name")
	}

	if len(resp.Items) <= 0 {
		machinePoolScope.Info("No machine pool found", "machinepool-name", machinePoolScope.OCIMachinePool.GetName())
		return nil, nil
	}

	instancePool := resp.Items[0]

	reqGet := core.GetInstancePoolRequest{
		InstancePoolId: instancePool.Id,
	}
	respGet, err := machinePoolScope.ComputeManagementClient.GetInstancePool(ctx, reqGet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query OCIMachinePool by name")
	}

	if machinePoolScope.IsResourceCreatedByClusterAPI(respGet.InstancePool.FreeformTags) {
		return nil, errors.Wrapf(err, "failed to query OCIMachinePool by name")
	}

	return &respGet.InstancePool, nil
}

func (r *OCIMachinePoolReconciler) createInstancePool(ctx context.Context, machinePoolScope *scope.MachinePoolScope) (*core.InstancePool, error) {
	//poolName := fmt.Sprintf("%s-%s", machinePoolScope.OCICluster.Name, machinePoolScope.MachinePool.Name)

	availabilityDomain := machinePoolScope.OCICluster.Status.FailureDomains["1"].Attributes["AvailabilityDomain"]
	placement := []core.CreateInstancePoolPlacementConfigurationDetails{
		{
			//AvailabilityDomain: common.String("zkJl:PHX-AD-1"),
			AvailabilityDomain: common.String(availabilityDomain),
			PrimarySubnetId:    machinePoolScope.GetWorkerMachineSubnet(),
		},
	}

	machinePoolScope.Info("Creating Instance Pool")
	req := core.CreateInstancePoolRequest{
		CreateInstancePoolDetails: core.CreateInstancePoolDetails{
			CompartmentId:           common.String(machinePoolScope.OCICluster.Spec.CompartmentId),
			InstanceConfigurationId: common.String(machinePoolScope.GetInstanceConfigurationId()),
			Size:                    common.Int(1),
			DisplayName:             common.String(machinePoolScope.OCIMachinePool.GetName()),

			PlacementConfigurations: placement,
			//	FreeformTags:
		},
	}
	instncePool, err := machinePoolScope.ComputeManagementClient.CreateInstancePool(ctx, req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create OCIMachinePool")
	}

	return &instncePool.InstancePool, nil

}
