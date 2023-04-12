/*
 Copyright (c) 2021, 2022 Oracle and/or its affiliates.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      https://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package controllers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/oracle/cluster-api-provider-oci/cloud/ociutil"
	"github.com/oracle/cluster-api-provider-oci/cloud/scope"
	"github.com/oracle/cluster-api-provider-oci/cloud/services/containerengine/mock_containerengine"
	infrav2exp "github.com/oracle/cluster-api-provider-oci/exp/api/v1beta2"
	"github.com/oracle/oci-go-sdk/v65/common"
	oke "github.com/oracle/oci-go-sdk/v65/containerengine"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	expclusterv1 "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestVirtualMachinePoolReconciliation(t *testing.T) {
	var (
		r        OCIVirtualMachinePoolReconciler
		mockCtrl *gomock.Controller
		req      reconcile.Request
		recorder *record.FakeRecorder
	)

	setup := func(t *testing.T, g *WithT) {
		mockCtrl = gomock.NewController(t)
		recorder = record.NewFakeRecorder(2)
	}
	teardown := func(t *testing.T, g *WithT) {
		mockCtrl.Finish()
	}
	tests := []struct {
		name             string
		errorExpected    bool
		objects          []client.Object
		expectedEvent    string
		eventNotExpected string
	}{
		{
			name:          "machine pool does not exist",
			errorExpected: false,
			objects:       []client.Object{getSecret()},
		},
		{
			name:          "no owner reference",
			errorExpected: false,
			objects:       []client.Object{getSecret(), getOCIVirtualMachinePoolWithNoOwner()},
			expectedEvent: "OwnerRefNotSet",
		},
		{
			name:          "cluster does not exist",
			errorExpected: false,
			objects:       []client.Object{getSecret(), getOCIVirtualMachinePool(), getMachinePool()},
			expectedEvent: "ClusterDoesNotExist",
		},
		{
			name:             "paused cluster",
			errorExpected:    false,
			objects:          []client.Object{getSecret(), getOCIVirtualMachinePool(), getMachinePool(), getPausedCluster()},
			eventNotExpected: "ClusterDoesNotExist",
		},
		{
			name:          "ocimanagedcluster does not exist",
			errorExpected: false,
			objects:       []client.Object{getSecret(), getOCIVirtualMachinePool(), getMachinePool(), getCluster()},
			expectedEvent: "ClusterNotAvailable",
		},
	}

	clientProvider, err := scope.MockNewClientProvider(scope.MockOCIClients{})
	if err != nil {
		t.Error(err)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			defer teardown(t, g)
			setup(t, g)

			client := fake.NewClientBuilder().WithObjects(tc.objects...).Build()
			r = OCIVirtualMachinePoolReconciler{
				Client:         client,
				Scheme:         runtime.NewScheme(),
				Recorder:       recorder,
				ClientProvider: clientProvider,
				Region:         MockTestRegion,
			}
			req = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "test",
					Name:      "test",
				},
			}

			_, err := r.Reconcile(context.Background(), req)
			if tc.errorExpected {
				g.Expect(err).To(Not(BeNil()))
			} else {
				g.Expect(err).To(BeNil())
			}
			if tc.expectedEvent != "" {
				g.Eventually(recorder.Events).Should(Receive(ContainSubstring(tc.expectedEvent)))
			}
			if tc.eventNotExpected != "" {
				g.Eventually(recorder.Events).ShouldNot(Receive(ContainSubstring(tc.eventNotExpected)))
			}
		})
	}
}

func TestNormalReconciliationFunctionForVirtualMP(t *testing.T) {
	var (
		r                     OCIVirtualMachinePoolReconciler
		mockCtrl              *gomock.Controller
		recorder              *record.FakeRecorder
		ociVirtualMachinePool *infrav2exp.OCIVirtualMachinePool
		okeClient             *mock_containerengine.MockClient
		ms                    *scope.VirtualMachinePoolScope
	)

	tags := make(map[string]string)
	tags[ociutil.CreatedBy] = ociutil.OCIClusterAPIProvider
	tags[ociutil.ClusterResourceIdentifier] = "resource_uid"

	setup := func(t *testing.T, g *WithT) {
		var err error
		mockCtrl = gomock.NewController(t)
		client := fake.NewClientBuilder().WithObjects(getSecret()).Build()
		okeClient = mock_containerengine.NewMockClient(mockCtrl)
		machinePool := getMachinePool()
		ociVirtualMachinePool = getOCIVirtualMachinePool()
		ociCluster := getOCIManagedClusterWithOwner()
		ociManagedControlPlane := infrav2exp.OCIManagedControlPlane{
			Spec: infrav2exp.OCIManagedControlPlaneSpec{
				ID: common.String("cluster-id"),
			},
			Status: infrav2exp.OCIManagedControlPlaneStatus{
				Ready: true,
			},
		}
		ms, err = scope.NewVirtualMachinePoolScope(scope.VirtualMachinePoolScopeParams{
			ContainerEngineClient:  okeClient,
			OCIManagedCluster:      ociCluster,
			Cluster:                getCluster(),
			Client:                 client,
			OCIVirtualMachinePool:  ociVirtualMachinePool,
			MachinePool:            machinePool,
			OCIManagedControlPlane: &ociManagedControlPlane,
		})

		recorder = record.NewFakeRecorder(2)
		r = OCIVirtualMachinePoolReconciler{
			Client:   client,
			Scheme:   runtime.NewScheme(),
			Recorder: recorder,
		}
		g.Expect(err).To(BeNil())
	}
	teardown := func(t *testing.T, g *WithT) {
		mockCtrl.Finish()
	}
	tests := []struct {
		name                    string
		errorExpected           bool
		expectedEvent           string
		eventNotExpected        string
		conditionAssertion      []conditionAssertion
		testSpecificSetup       func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient)
		expectedFailureMessages []string
	}{
		{
			name:               "virtual node pool in creating state",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav2exp.VirtualNodePoolNotReadyReason}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:             common.String("test"),
							LifecycleState: oke.VirtualNodePoolLifecycleStateCreating,
							FreeformTags:   tags,
						},
					}, nil)
			},
		},
		{
			name:               "virtual node pool create",
			errorExpected:      false,
			expectedEvent:      "Created new Virtual Node Pool: test",
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityInfo, infrav2exp.VirtualNodePoolNotReadyReason}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				ociVirtualMachinePool.Spec.ID = nil
				okeClient.EXPECT().ListVirtualNodePools(gomock.Any(), gomock.Any()).
					Return(oke.ListVirtualNodePoolsResponse{}, nil)
				okeClient.EXPECT().CreateVirtualNodePool(gomock.Any(), gomock.Any()).
					Return(oke.CreateVirtualNodePoolResponse{OpcWorkRequestId: common.String("wr-id")}, nil)
				okeClient.EXPECT().GetWorkRequest(gomock.Any(), gomock.Any()).
					Return(oke.GetWorkRequestResponse{WorkRequest: oke.WorkRequest{
						Resources: []oke.WorkRequestResource{
							{
								Identifier: common.String("virtual-node-pool"),
								EntityType: common.String("VirtualNodePool"),
							},
						},
					}}, nil)
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Any()).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:             common.String("test"),
							LifecycleState: oke.VirtualNodePoolLifecycleStateCreating,
							FreeformTags:   tags,
						},
					}, nil)
			},
		},
		{
			name:               "virtual node pool is created, no update",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionTrue, "", ""}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:                common.String("id"),
							LifecycleState:    oke.VirtualNodePoolLifecycleStateActive,
							ClusterId:         common.String("cluster-id"),
							DisplayName:       common.String("test"),
							CompartmentId:     common.String("test-compartment"),
							KubernetesVersion: common.String("v1.24.5"),
							InitialVirtualNodeLabels: []oke.InitialVirtualNodeLabel{{
								Key:   common.String("key"),
								Value: common.String("value"),
							}},
							Taints: []oke.Taint{{
								Key:    common.String("key"),
								Value:  common.String("value"),
								Effect: common.String("effect"),
							}},
							PlacementConfigurations: []oke.PlacementConfiguration{
								{
									AvailabilityDomain: common.String("test-ad"),
									SubnetId:           common.String("subnet-id"),
									FaultDomain:        []string{"fd-1", "fd-2"},
								},
							},
							NsgIds: []string{"nsg-id"},
							PodConfiguration: &oke.PodConfiguration{
								NsgIds:   []string{"pod-nsg-id"},
								Shape:    common.String("pod-shape"),
								SubnetId: common.String("pod-subnet-id"),
							},
							Size:         common.Int(3),
							FreeformTags: tags,
						},
					}, nil)

				okeClient.EXPECT().ListVirtualNodes(gomock.Any(), gomock.Eq(oke.ListVirtualNodesRequest{
					VirtualNodePoolId: common.String("id"),
				})).Return(oke.ListVirtualNodesResponse{}, nil)
			},
		},
		{
			name:               "virtual node pool in created, update",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionTrue, "", ""}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:                common.String("id"),
							LifecycleState:    oke.VirtualNodePoolLifecycleStateActive,
							ClusterId:         common.String("cluster-id"),
							DisplayName:       common.String("test"),
							CompartmentId:     common.String("test-compartment"),
							KubernetesVersion: common.String("v1.23.5"),
							InitialVirtualNodeLabels: []oke.InitialVirtualNodeLabel{{
								Key:   common.String("key"),
								Value: common.String("value"),
							}},
							Taints: []oke.Taint{{
								Key:    common.String("key"),
								Value:  common.String("value"),
								Effect: common.String("effect"),
							}},
							PlacementConfigurations: []oke.PlacementConfiguration{
								{
									AvailabilityDomain: common.String("test-ad"),
									SubnetId:           common.String("subnet-id"),
									FaultDomain:        []string{"fd-1", "fd-2"},
								},
							},
							PodConfiguration: &oke.PodConfiguration{
								NsgIds:   []string{"pod-nsg-id"},
								Shape:    common.String("pod-shape"),
								SubnetId: common.String("pod-subnet-id"),
							},
							Size:         common.Int(3),
							FreeformTags: tags,
						},
					}, nil)
				okeClient.EXPECT().ListVirtualNodes(gomock.Any(), gomock.Eq(oke.ListVirtualNodesRequest{
					VirtualNodePoolId: common.String("id"),
				})).Return(oke.ListVirtualNodesResponse{}, nil)
				okeClient.EXPECT().UpdateVirtualNodePool(gomock.Any(), gomock.Any()).
					Return(oke.UpdateVirtualNodePoolResponse{}, nil)
			},
		},
		{
			name:                    "virtual node pool in error state",
			errorExpected:           true,
			expectedFailureMessages: []string{"Virtual Node Pool status FAILED is unexpected"},
			conditionAssertion:      []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityError, infrav2exp.VirtualNodePoolProvisionFailedReason}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:             common.String("test"),
							LifecycleState: oke.VirtualNodePoolLifecycleStateFailed,
							FreeformTags:   tags,
						},
					}, nil)
			},
		},
		{
			name:          "virtual node pool in update state",
			errorExpected: false,
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:             common.String("test"),
							LifecycleState: oke.VirtualNodePoolLifecycleStateUpdating,
							FreeformTags:   tags,
						},
					}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			defer teardown(t, g)
			setup(t, g)
			tc.testSpecificSetup(ms, okeClient)
			ctx := context.Background()
			_, err := r.reconcileNormal(ctx, log.FromContext(ctx), ms)
			if len(tc.conditionAssertion) > 0 {
				expectVMPConditions(g, ociVirtualMachinePool, tc.conditionAssertion)
			}
			if tc.errorExpected {
				g.Expect(err).To(Not(BeNil()))
			} else {
				g.Expect(err).To(BeNil())
			}
			if tc.expectedEvent != "" {
				g.Eventually(recorder.Events).Should(Receive(ContainSubstring(tc.expectedEvent)))
			}
			if len(tc.expectedFailureMessages) > 0 {
				g.Expect(tc.expectedFailureMessages).To(Equal(ms.OCIVirtualMachinePool.Status.FailureMessages))
			}
		})
	}
}

func TestVMPDeletionFunction(t *testing.T) {
	var (
		r                     OCIVirtualMachinePoolReconciler
		mockCtrl              *gomock.Controller
		recorder              *record.FakeRecorder
		ociVirtualMachinePool *infrav2exp.OCIVirtualMachinePool
		okeClient             *mock_containerengine.MockClient
		ms                    *scope.VirtualMachinePoolScope
	)

	tags := make(map[string]string)
	tags[ociutil.CreatedBy] = ociutil.OCIClusterAPIProvider
	tags[ociutil.ClusterResourceIdentifier] = "resource_uid"

	setup := func(t *testing.T, g *WithT) {
		var err error
		mockCtrl = gomock.NewController(t)
		client := fake.NewClientBuilder().WithObjects(getSecret()).Build()
		okeClient = mock_containerengine.NewMockClient(mockCtrl)
		machinePool := getMachinePool()
		ociVirtualMachinePool = getOCIVirtualMachinePool()
		ociCluster := getOCIManagedClusterWithOwner()
		ociManagedControlPlane := infrav2exp.OCIManagedControlPlane{
			Spec: infrav2exp.OCIManagedControlPlaneSpec{
				ID: common.String("cluster-id"),
			},
			Status: infrav2exp.OCIManagedControlPlaneStatus{
				Ready: true,
			},
		}
		ms, err = scope.NewVirtualMachinePoolScope(scope.VirtualMachinePoolScopeParams{
			ContainerEngineClient:  okeClient,
			OCIManagedCluster:      ociCluster,
			Cluster:                getCluster(),
			Client:                 client,
			OCIVirtualMachinePool:  ociVirtualMachinePool,
			MachinePool:            machinePool,
			OCIManagedControlPlane: &ociManagedControlPlane,
		})

		recorder = record.NewFakeRecorder(2)
		r = OCIVirtualMachinePoolReconciler{
			Client:   client,
			Scheme:   runtime.NewScheme(),
			Recorder: recorder,
		}
		g.Expect(err).To(BeNil())
	}
	teardown := func(t *testing.T, g *WithT) {
		mockCtrl.Finish()
	}
	tests := []struct {
		name                    string
		errorExpected           bool
		expectedEvent           string
		eventNotExpected        string
		conditionAssertion      []conditionAssertion
		testSpecificSetup       func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient)
		expectedFailureMessages []string
	}{
		{
			name:               "virtual node pool to be deleted",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityWarning, infrav2exp.VirtualNodePoolDeletionInProgress}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:             common.String("test"),
							LifecycleState: oke.VirtualNodePoolLifecycleStateActive,
							FreeformTags:   tags,
						},
					}, nil)
				okeClient.EXPECT().DeleteVirtualNodePool(gomock.Any(), gomock.Eq(oke.DeleteVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.DeleteVirtualNodePoolResponse{}, nil)
			},
		},
		{
			name:               "virtual node pool not found",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolNotFoundReason, corev1.ConditionTrue, "", ""}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{}, ociutil.ErrNotFound)
			},
		},
		{
			name:               "virtual node pool",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolNotFoundReason, corev1.ConditionTrue, "", ""}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{}, ociutil.ErrNotFound)
			},
		},
		{
			name:               "virtual node pool deleting",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityWarning, infrav2exp.VirtualNodePoolDeletionInProgress}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:             common.String("test"),
							LifecycleState: oke.VirtualNodePoolLifecycleStateDeleting,
							FreeformTags:   tags,
						},
					}, nil)
			},
		},
		{
			name:               "virtual node pool deleted",
			errorExpected:      false,
			conditionAssertion: []conditionAssertion{{infrav2exp.VirtualNodePoolReadyCondition, corev1.ConditionFalse, clusterv1.ConditionSeverityWarning, infrav2exp.VirtualNodePoolDeletedReason}},
			testSpecificSetup: func(machinePoolScope *scope.VirtualMachinePoolScope, okeClient *mock_containerengine.MockClient) {
				okeClient.EXPECT().GetVirtualNodePool(gomock.Any(), gomock.Eq(oke.GetVirtualNodePoolRequest{
					VirtualNodePoolId: common.String("test"),
				})).
					Return(oke.GetVirtualNodePoolResponse{
						VirtualNodePool: oke.VirtualNodePool{
							Id:             common.String("test"),
							LifecycleState: oke.VirtualNodePoolLifecycleStateDeleted,
							FreeformTags:   tags,
						},
					}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			defer teardown(t, g)
			setup(t, g)
			tc.testSpecificSetup(ms, okeClient)
			ctx := context.Background()
			_, err := r.reconcileDelete(ctx, ms)
			if len(tc.conditionAssertion) > 0 {
				expectVMPConditions(g, ociVirtualMachinePool, tc.conditionAssertion)
			}
			if tc.errorExpected {
				g.Expect(err).To(Not(BeNil()))
			} else {
				g.Expect(err).To(BeNil())
			}
			if tc.expectedEvent != "" {
				g.Eventually(recorder.Events).Should(Receive(ContainSubstring(tc.expectedEvent)))
			}
			if len(tc.expectedFailureMessages) > 0 {
				g.Expect(tc.expectedFailureMessages).To(Equal(ms.OCIVirtualMachinePool.Status.FailureMessages))
			}
		})
	}
}

func getOCIVirtualMachinePoolWithNoOwner() *infrav2exp.OCIVirtualMachinePool {
	ociMachine := getOCIVirtualMachinePool()
	ociMachine.OwnerReferences = []metav1.OwnerReference{}
	return ociMachine
}

func getOCIVirtualMachinePool() *infrav2exp.OCIVirtualMachinePool {
	return &infrav2exp.OCIVirtualMachinePool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			UID:       "uid",
			Labels: map[string]string{
				clusterv1.ClusterLabelName: "test-cluster",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       "test-cluster",
					Kind:       "Cluster",
					APIVersion: clusterv1.GroupVersion.String(),
				},
				{
					Name:       "test",
					Kind:       "MachinePool",
					APIVersion: expclusterv1.GroupVersion.String(),
				},
			},
		},
		Spec: infrav2exp.OCIVirtualMachinePoolSpec{
			ID:       common.String("test"),
			NsgNames: []string{"worker-nsg"},
			InitialVirtualNodeLabels: []infrav2exp.KeyValue{{
				Key:   common.String("key"),
				Value: common.String("value"),
			}},
			Taints: []infrav2exp.Taint{{
				Key:    common.String("key"),
				Value:  common.String("value"),
				Effect: common.String("effect"),
			}},
			PodConfiguration: infrav2exp.PodConfig{
				NsgNames:   []string{"pod-nsg"},
				Shape:      common.String("pod-shape"),
				SubnetName: common.String("pod-subnet"),
			},
			PlacementConfigs: []infrav2exp.VirtualNodepoolPlacementConfig{
				{
					AvailabilityDomain: common.String("test-ad"),
					SubnetName:         common.String("worker-subnet"),
					FaultDomains:       []string{"fd-1", "fd-2"},
				},
			},
		},
	}
}

func expectVMPConditions(g *WithT, m *infrav2exp.OCIVirtualMachinePool, expected []conditionAssertion) {
	g.Expect(len(m.Status.Conditions)).To(BeNumerically(">=", len(expected)), "number of conditions")
	for _, c := range expected {
		actual := conditions.Get(m, c.conditionType)
		g.Expect(actual).To(Not(BeNil()))
		g.Expect(actual.Type).To(Equal(c.conditionType))
		g.Expect(actual.Status).To(Equal(c.status))
		g.Expect(actual.Severity).To(Equal(c.severity))
		g.Expect(actual.Reason).To(Equal(c.reason))
	}
}
