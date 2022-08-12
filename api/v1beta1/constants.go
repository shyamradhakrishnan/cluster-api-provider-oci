package v1beta1

const (
	VcnDefaultCidr                        = "10.0.0.0/16"
	ControlPlaneEndpointSubnetDefaultCIDR = "10.0.0.8/29"
	ControlPlaneMachineSubnetDefaultCIDR  = "10.0.0.0/29"
	WorkerSubnetDefaultCIDR               = "10.0.64.0/20"
	ServiceLoadBalancerDefaultCIDR        = "10.0.0.32/27"
	APIServerLBBackendSetName             = "apiserver-lb-backendset"
	APIServerLBListener                   = "apiserver-lb-listener"
	ControlPlaneEndpointDefaultName       = "control-plane-endpoint"
	ControlPlaneDefaultName               = "control-plane"
	WorkerDefaultName                     = "worker"
	ServiceLBDefaultName                  = "service-lb"
	PodDefaultName                        = "pod"
	PodDefaultCIDR                        = "10.0.4.0/24"
)
