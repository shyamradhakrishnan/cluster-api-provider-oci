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

package v1beta1

import (
	"fmt"

	"github.com/oracle/oci-go-sdk/v63/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var managedclusterlogger = ctrl.Log.WithName("ocimanagedcluster-resource")

var (
	_ webhook.Defaulter = &OCIManagedCluster{}
	_ webhook.Validator = &OCIManagedCluster{}
)

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-ocimanagedcluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=ocimanagedclusters,versions=v1beta1,name=validation.ocimanagedcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1beta1-ocimanagedcluster,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=ocimanagedclusters,versions=v1beta1,name=default.ocimanagedcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

func (c *OCIManagedCluster) Default() {
	if c.Spec.OCIResourceIdentifier == "" {
		c.Spec.OCIResourceIdentifier = string(uuid.NewUUID())
	}
	if !c.Spec.NetworkSpec.SkipNetworkManagement {
		if len(c.Spec.NetworkSpec.Vcn.Subnets) == 0 {
			subnets := make([]*Subnet, 0)
			subnets = append(subnets, &Subnet{
				Role: ControlPlaneEndpointRole,
				Name: ControlPlaneEndpointDefaultName,
				CIDR: ControlPlaneEndpointSubnetDefaultCIDR,
				Type: Public,
			})

			subnets = append(subnets, &Subnet{
				Role: ServiceLoadBalancerRole,
				Name: ServiceLBDefaultName,
				CIDR: ServiceLoadBalancerDefaultCIDR,
				Type: Public,
			})
			subnets = append(subnets, &Subnet{
				Role: WorkerRole,
				Name: WorkerDefaultName,
				CIDR: WorkerSubnetDefaultCIDR,
				Type: Private,
			})
			subnets = append(subnets, &Subnet{
				Role: PodRole,
				Name: PodDefaultName,
				CIDR: PodDefaultCIDR,
				Type: Private,
			})
			c.Spec.NetworkSpec.Vcn.Subnets = subnets
		}
		if len(c.Spec.NetworkSpec.Vcn.NetworkSecurityGroups) == 0 {
			nsgs := make([]*NSG, 0)
			nsgs = append(nsgs, &NSG{
				Role:         ControlPlaneEndpointRole,
				Name:         ControlPlaneEndpointDefaultName,
				IngressRules: c.GetControlPlaneEndpointDefaultIngressRules(),
				EgressRules:  c.GetControlPlaneEndpointDefaultEgressRules(),
			})
			nsgs = append(nsgs, &NSG{
				Role:         WorkerRole,
				Name:         WorkerDefaultName,
				IngressRules: c.GetWorkerDefaultIngressRules(),
				EgressRules:  c.GetWorkerDefaultEgressRules(),
			})
			nsgs = append(nsgs, &NSG{
				Role:         ServiceLoadBalancerRole,
				Name:         ServiceLBDefaultName,
				IngressRules: c.GetPodDefaultIngressRules(),
				EgressRules:  c.GetPodDefaultEgressRules(),
			})
			nsgs = append(nsgs, &NSG{
				Role:         PodRole,
				Name:         PodDefaultName,
				IngressRules: c.GetLBServiceDefaultIngressRules(),
				EgressRules:  c.GetLBServiceDefaultEgressRules(),
			})
			c.Spec.NetworkSpec.Vcn.NetworkSecurityGroups = nsgs
		}
		if c.Spec.NetworkSpec.Vcn.CIDR == "" {
			c.Spec.NetworkSpec.Vcn.CIDR = VcnDefaultCidr
		}
	}
}

func (c *OCIManagedCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (c *OCIManagedCluster) ValidateCreate() error {
	clusterlogger.Info("validate update cluster", "name", c.Name)

	var allErrs field.ErrorList

	allErrs = append(allErrs, c.validate(nil)...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(c.GroupVersionKind().GroupKind(), c.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (c *OCIManagedCluster) ValidateDelete() error {
	managedclusterlogger.Info("validate delete cluster", "name", c.Name)

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (c *OCIManagedCluster) ValidateUpdate(old runtime.Object) error {
	clusterlogger.Info("validate update cluster", "name", c.Name)

	var allErrs field.ErrorList

	oldCluster, ok := old.(*OCICluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected an OCICluster but got a %T", old))
	}

	if c.Spec.Region != oldCluster.Spec.Region {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "region"), c.Spec.Region, "field is immutable"))
	}

	if c.Spec.OCIResourceIdentifier != oldCluster.Spec.OCIResourceIdentifier {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "ociResourceIdentifier"), c.Spec.OCIResourceIdentifier, "field is immutable"))
	}

	if c.Spec.CompartmentId != oldCluster.Spec.CompartmentId {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "compartmentId"), c.Spec.CompartmentId, "field is immutable"))
	}

	allErrs = append(allErrs, c.validate(oldCluster)...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(c.GroupVersionKind().GroupKind(), c.Name, allErrs)
}

func (c *OCIManagedCluster) validate(old *OCICluster) field.ErrorList {
	var allErrs field.ErrorList

	var oldNetworkSpec NetworkSpec
	if old != nil {
		oldNetworkSpec = old.Spec.NetworkSpec
	}

	allErrs = append(allErrs, validateNetworkSpec(c.Spec.NetworkSpec, oldNetworkSpec, field.NewPath("spec").Child("networkSpec"))...)

	if len(c.Spec.CompartmentId) <= 0 {
		allErrs = append(
			allErrs,
			field.Invalid(field.NewPath("spec", "compartmentId"), c.Spec.CompartmentId, "field is required"))
	}

	// Handle case where CompartmentId exists, but isn't valid
	// the separate "blank" check above is a more clear error for the user
	if len(c.Spec.CompartmentId) > 0 && !validOcid(c.Spec.CompartmentId) {
		allErrs = append(
			allErrs,
			field.Invalid(field.NewPath("spec", "compartmentId"), c.Spec.CompartmentId, "field is invalid"))
	}

	if len(c.Spec.OCIResourceIdentifier) <= 0 {
		allErrs = append(
			allErrs,
			field.Invalid(field.NewPath("spec", "ociResourceIdentifier"), c.Spec.OCIResourceIdentifier, "field is required"))
	}

	if !validRegion(c.Spec.Region) {
		allErrs = append(
			allErrs,
			field.Invalid(field.NewPath("spec", "region"), c.Spec.Region, "field is invalid. See https://docs.oracle.com/en-us/iaas/Content/General/Concepts/regions.htm"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return allErrs
}
func (c *OCIManagedCluster) GetControlPlaneEndpointSubnet() *Subnet {
	for _, subnet := range c.Spec.NetworkSpec.Vcn.Subnets {
		if subnet.Role == ControlPlaneEndpointRole {
			return subnet
		}
	}
	return nil
}

func (c *OCIManagedCluster) GetServiceLoadBalancerSubnet() *Subnet {
	for _, subnet := range c.Spec.NetworkSpec.Vcn.Subnets {
		if subnet.Role == ServiceLoadBalancerRole {
			return subnet
		}
	}
	return nil
}

func (c *OCIManagedCluster) GetNodeSubnet() *Subnet {
	for _, subnet := range c.Spec.NetworkSpec.Vcn.Subnets {
		if subnet.Role == WorkerRole {
			return subnet
		}
	}
	return nil
}

func (c *OCIManagedCluster) GetPodSubnet() *Subnet {
	for _, subnet := range c.Spec.NetworkSpec.Vcn.Subnets {
		if subnet.Role == PodRole {
			return subnet
		}
	}
	return nil
}

func (c *OCIManagedCluster) GetControlPlaneEndpointDefaultIngressRules() []IngressSecurityRuleForNSG {
	return []IngressSecurityRuleForNSG{
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Kubernetes worker to Kubernetes API endpoint communication."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(6443),
						Min: common.Int(6443),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String(WorkerSubnetDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Kubernetes worker to Kubernetes API endpoint communication."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(12250),
						Min: common.Int(12250),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String(WorkerSubnetDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Path Discovery."),
				Protocol:    common.String("1"),
				IcmpOptions: &IcmpOptions{
					Type: common.Int(3),
					Code: common.Int(4),
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String(WorkerSubnetDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Pod to Kubernetes API endpoint communication (when using VCN-native pod networking)."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(6443),
						Min: common.Int(6443),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String(PodDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Pod to Kubernetes API endpoint communication (when using VCN-native pod networking)."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(12250),
						Min: common.Int(12250),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String(PodDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("External access to Kubernetes API endpoint."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(6443),
						Min: common.Int(6443),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String("0.0.0.0/0"),
			},
		},
	}
}

func (c *OCIManagedCluster) GetControlPlaneEndpointDefaultEgressRules() []EgressSecurityRuleForNSG {
	// TODO add service gateway rules
	return []EgressSecurityRuleForNSG{
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Allow Kubernetes API endpoint to communicate with worker nodes."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(10250),
						Min: common.Int(10250),
					},
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(WorkerSubnetDefaultCIDR),
			},
		},
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Path Discovery."),
				Protocol:    common.String("1"),
				IcmpOptions: &IcmpOptions{
					Type: common.Int(3),
					Code: common.Int(4),
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(WorkerSubnetDefaultCIDR),
			},
		},
		{
			EgressSecurityRule: EgressSecurityRule{
				Description:     common.String("Allow Kubernetes API endpoint to communicate with pods (when using VCN-native pod networking)."),
				Protocol:        common.String("all"),
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(PodDefaultCIDR),
			},
		},
	}
}

func (c *OCIManagedCluster) GetWorkerDefaultIngressRules() []IngressSecurityRuleForNSG {
	return []IngressSecurityRuleForNSG{
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Allow Kubernetes API endpoint to communicate with worker nodes."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(10250),
						Min: common.Int(10250),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String(ControlPlaneEndpointSubnetDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Path Discovery."),
				Protocol:    common.String("1"),
				IcmpOptions: &IcmpOptions{
					Type: common.Int(3),
					Code: common.Int(4),
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String("0.0.0.0/0"),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Load Balancer to Worker nodes node ports."),
				Protocol:    common.String("all"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(32767),
						Min: common.Int(30000),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String(ServiceLoadBalancerDefaultCIDR),
			},
		},
	}
}

func (c *OCIManagedCluster) GetWorkerDefaultEgressRules() []EgressSecurityRuleForNSG {
	// TODO add service gateway rules
	return []EgressSecurityRuleForNSG{
		{
			EgressSecurityRule: EgressSecurityRule{
				Description:     common.String("Allow worker nodes to access pods."),
				Protocol:        common.String("all"),
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(PodDefaultCIDR),
			},
		},
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Path Discovery."),
				Protocol:    common.String("1"),
				IcmpOptions: &IcmpOptions{
					Type: common.Int(3),
					Code: common.Int(4),
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String("0.0.0.0/0"),
			},
		},
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Kubernetes worker to Kubernetes API endpoint communication."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(6443),
						Min: common.Int(6443),
					},
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(ControlPlaneEndpointSubnetDefaultCIDR),
			},
		},
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Kubernetes worker to Kubernetes API endpoint communication."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(12250),
						Min: common.Int(12250),
					},
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(ControlPlaneEndpointSubnetDefaultCIDR),
			},
		},
	}
}

func (c *OCIManagedCluster) GetPodDefaultIngressRules() []IngressSecurityRuleForNSG {
	return []IngressSecurityRuleForNSG{
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Allow worker nodes to access pods."),
				Protocol:    common.String("all"),
				SourceType:  IngressSecurityRuleSourceTypeCidrBlock,
				Source:      common.String(WorkerSubnetDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Allow Kubernetes API endpoint to communicate with pods."),
				Protocol:    common.String("all"),
				SourceType:  IngressSecurityRuleSourceTypeCidrBlock,
				Source:      common.String(ControlPlaneEndpointSubnetDefaultCIDR),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Allow pods to communicate with other pods."),
				Protocol:    common.String("all"),
				SourceType:  IngressSecurityRuleSourceTypeCidrBlock,
				Source:      common.String(PodDefaultCIDR),
			},
		},
	}
}

func (c *OCIManagedCluster) GetPodDefaultEgressRules() []EgressSecurityRuleForNSG {
	// TODO add service gateway rules
	return []EgressSecurityRuleForNSG{
		{
			EgressSecurityRule: EgressSecurityRule{
				Description:     common.String("Allow pods to communicate with other pods."),
				Protocol:        common.String("all"),
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(PodDefaultCIDR),
			},
		},
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Pod to Kubernetes API endpoint communication (when using VCN-native pod networking)."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(6443),
						Min: common.Int(6443),
					},
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(ControlPlaneEndpointSubnetDefaultCIDR),
			},
		},
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Pod to Kubernetes API endpoint communication (when using VCN-native pod networking)."),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(12250),
						Min: common.Int(12250),
					},
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(ControlPlaneEndpointSubnetDefaultCIDR),
			},
		},
	}
}

func (c *OCIManagedCluster) GetLBServiceDefaultIngressRules() []IngressSecurityRuleForNSG {
	return []IngressSecurityRuleForNSG{
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Accept http traffic on port 80"),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(80),
						Min: common.Int(80),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String("0.0.0.0/0"),
			},
		},
		{
			IngressSecurityRule: IngressSecurityRule{
				Description: common.String("Accept https traffic on port 443"),
				Protocol:    common.String("6"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(443),
						Min: common.Int(443),
					},
				},
				SourceType: IngressSecurityRuleSourceTypeCidrBlock,
				Source:     common.String("0.0.0.0/0"),
			},
		},
	}
}

func (c *OCIManagedCluster) GetLBServiceDefaultEgressRules() []EgressSecurityRuleForNSG {
	// TODO add service gateway rules
	return []EgressSecurityRuleForNSG{
		{
			EgressSecurityRule: EgressSecurityRule{
				Description: common.String("Pod to Kubernetes API endpoint communication (when using VCN-native pod networking)."),
				Protocol:    common.String("all"),
				TcpOptions: &TcpOptions{
					DestinationPortRange: &PortRange{
						Max: common.Int(32767),
						Min: common.Int(30000),
					},
				},
				DestinationType: EgressSecurityRuleDestinationTypeCidrBlock,
				Destination:     common.String(WorkerSubnetDefaultCIDR),
			},
		},
	}
}
