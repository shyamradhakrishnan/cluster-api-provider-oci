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

package scope

import "github.com/oracle/oci-go-sdk/v65/networkloadbalancer"

const (
	VcnDefaultCidr                        = "10.0.0.0/16"
	ControlPlaneEndpointSubnetDefaultCIDR = "10.0.0.8/29"
	ControlPlaneMachineSubnetDefaultCIDR  = "10.0.0.0/29"
	WorkerSubnetDefaultCIDR               = "10.0.64.0/20"
	ServiceLoadBalancerDefaultCIDR        = "10.0.0.32/27"
	ApiServerPort                         = 6443
	APIServerLBBackendSetName             = "apiserver-lb-backendset"
	APIServerLBListener                   = "apiserver-lb-listener"
	SGWServiceSuffix                      = "-services-in-oracle-services-network"
	ServiceGatewayName                    = "service-gateway"
	PublicRouteTableName                  = "public-route-table"
	PrivateRouteTableName                 = "private-route-table"
	NatGatewayName                        = "nat-gateway"
	InternetGatewayName                   = "internet-gateway"
	ControlPlaneEndpointDefaultName       = "control-plane-endpoint"
	ControlPlaneDefaultName               = "control-plane"
	WorkerDefaultName                     = "worker"
	ServiceLBDefaultName                  = "service-lb"
)

var (
	LoadBalancerPolicy = networkloadbalancer.NetworkLoadBalancingPolicyFiveTuple
)
