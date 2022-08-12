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

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/oracle/cluster-api-provider-oci/cloud/base"
	"github.com/oracle/cluster-api-provider-oci/cloud/services/vcn"

	"github.com/go-logr/logr"
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	"github.com/oracle/cluster-api-provider-oci/cloud/ociutil"
	identityClent "github.com/oracle/cluster-api-provider-oci/cloud/services/identity"
	nlb "github.com/oracle/cluster-api-provider-oci/cloud/services/networkloadbalancer"
	"github.com/oracle/oci-go-sdk/v63/common"
	"github.com/oracle/oci-go-sdk/v63/identity"
	"github.com/pkg/errors"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AvailabilityDomain    = "AvailabilityDomain"
	FaultDomain           = "FaultDomain"
	OCIClusterKind        = "OCICluster"
	OCIManagedClusterKind = "OCIManagedCluster"
)

// ClusterScopeParams defines the params need to create a new ClusterScope
type ClusterScopeParams struct {
	Client                client.Client
	Logger                *logr.Logger
	Cluster               *clusterv1.Cluster
	VCNClient             vcn.Client
	LoadBalancerClient    nlb.NetworkLoadBalancerClient
	IdentityClient        identityClent.Client
	Region                string
	OCIAuthConfigProvider common.ConfigurationProvider
	ClientProvider        *ClientProvider
	OCIClusterBase        base.OCIClusterBase
}

type ClusterScope struct {
	*logr.Logger
	client             client.Client
	patchHelper        *patch.Helper
	Cluster            *clusterv1.Cluster
	VCNClient          vcn.Client
	LoadBalancerClient nlb.NetworkLoadBalancerClient
	IdentityClient     identityClent.Client
	Region             string
	ClientProvider     *ClientProvider
	OCIClusterBase     base.OCIClusterBase
}

// NewClusterScope creates a ClusterScope given the ClusterScopeParams
func NewClusterScope(params ClusterScopeParams) (*ClusterScope, error) {
	// TODO add conditions everywhere properly and events as well
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.OCIClusterBase == nil {
		return nil, errors.New("failed to generate new scope from nil OCICluster")
	}

	if params.Logger == nil {
		log := klogr.New()
		params.Logger = &log
	}

	return &ClusterScope{
		Logger:             params.Logger,
		client:             params.Client,
		Cluster:            params.Cluster,
		VCNClient:          params.VCNClient,
		LoadBalancerClient: params.LoadBalancerClient,
		IdentityClient:     params.IdentityClient,
		Region:             params.Region,
		ClientProvider:     params.ClientProvider,
		OCIClusterBase:     params.OCIClusterBase,
	}, nil
}

func (s *ClusterScope) IsResourceCreatedByClusterAPI(resourceFreeFormTags map[string]string) bool {
	tagsAddedByClusterAPI := ociutil.BuildClusterTags(s.OCIClusterBase.GetOCIResourceIdentifier())
	for k, v := range tagsAddedByClusterAPI {
		if resourceFreeFormTags[k] != v {
			return false
		}
	}
	return true
}

func (s *ClusterScope) ReconcileFailureDomains(ctx context.Context) error {
	if s.OCIClusterBase.GetOCIClusterStatus().FailureDomains == nil {
		return s.setFailureDomains(ctx)
	}
	return nil
}

// setFailureDomains sets the failure domains of the environment based on whether it is single AD or multi AD regions
// in case of single AD regions, the failure domain will be fault domain, in case of multi Ad regions, it will
// be AD
func (s *ClusterScope) setFailureDomains(ctx context.Context) error {
	reqAd := identity.ListAvailabilityDomainsRequest{CompartmentId: common.String(s.GetCompartmentId())}

	respAd, err := s.IdentityClient.ListAvailabilityDomains(ctx, reqAd)
	if err != nil {
		s.Logger.Error(err, "failed to list identity domains")
		return err
	}

	// build the AD list for cluster
	err = s.setAvailabiltyDomainStatus(ctx, respAd.Items)
	if err != nil {
		return err
	}

	numOfAds := len(respAd.Items)
	if numOfAds != 1 && numOfAds != 3 {
		err := errors.New(fmt.Sprintf("invalid number of Availability Domains, should be either 1 or 3, but got %d", numOfAds))
		s.Logger.Error(err, "invalid number of Availability Domains")
		return err
	}

	if numOfAds == 3 {
		for i, ad := range respAd.Items {
			s.SetFailureDomain(strconv.Itoa(i+1), clusterv1.FailureDomainSpec{
				ControlPlane: true,
				Attributes:   map[string]string{AvailabilityDomain: *ad.Name},
			})
		}
	} else {
		adName := *respAd.Items[0].Name
		for i, fd := range s.OCIClusterBase.GetOCIClusterStatus().AvailabilityDomains[adName].FaultDomains {
			s.SetFailureDomain(strconv.Itoa(i+1), clusterv1.FailureDomainSpec{
				ControlPlane: true,
				Attributes: map[string]string{
					AvailabilityDomain: adName,
					FaultDomain:        fd,
				},
			})
		}
	}

	return nil
}

// SetFailureDomain sets the cluster's failure domain in the status
func (s *ClusterScope) SetFailureDomain(id string, spec clusterv1.FailureDomainSpec) {
	status := s.OCIClusterBase.GetOCIClusterStatus()
	if status.FailureDomains == nil {
		status.FailureDomains = make(clusterv1.FailureDomains)
	}
	status.FailureDomains[id] = spec
}

// setAvailabiltyDomainStatus builds the OCIAvailabilityDomain list and sets the OCICluster's status with this list
// so that other parts of the provider have access to ADs and FDs without having to make multiple calls to identity.
func (s *ClusterScope) setAvailabiltyDomainStatus(ctx context.Context, ads []identity.AvailabilityDomain) error {
	clusterAds := make(map[string]infrastructurev1beta1.OCIAvailabilityDomain)
	for _, ad := range ads {
		reqFd := identity.ListFaultDomainsRequest{
			CompartmentId:      common.String(s.GetCompartmentId()),
			AvailabilityDomain: ad.Name,
		}
		respFd, err := s.IdentityClient.ListFaultDomains(ctx, reqFd)
		if err != nil {
			s.Logger.Error(err, "failed to list fault domains")
			return err
		}

		var faultDomains []string
		for _, fd := range respFd.Items {
			faultDomains = append(faultDomains, *fd.Name)
		}

		adName := *ad.Name
		clusterAds[adName] = infrastructurev1beta1.OCIAvailabilityDomain{
			Name:         adName,
			FaultDomains: faultDomains,
		}
	}

	status := s.OCIClusterBase.GetOCIClusterStatus()
	status.AvailabilityDomains = clusterAds

	return nil
}

func (s *ClusterScope) IsTagsEqual(freeFromTags map[string]string, definedTags map[string]map[string]interface{}) bool {
	if reflect.DeepEqual(freeFromTags, s.GetFreeFormTags()) && reflect.DeepEqual(definedTags, s.GetDefinedTags()) {
		return true
	}
	return false
}

// GetRegionCodeFromRegion pulls all OCI regions available and returns the passed in region's code if contained in
// the list.
//
// example: "ca-toronto-1" -> "YYZ"
func (s *ClusterScope) GetRegionCodeFromRegion(ctx context.Context, region string) (string, error) {
	regionCodes, err := s.IdentityClient.ListRegions(ctx)
	if err != nil {
		s.Logger.Error(err, "failed to list oci regions")
		return "", errors.Wrap(err, "failed to list oci regions")
	}
	for _, regionCode := range regionCodes.Items {
		if *regionCode.Name == region {
			return *regionCode.Key, nil
		}
	}
	return "", errors.Errorf("unable to get region code from region name")
}

// GetDefinedTags returns a map of DefinedTags defined in the OCICluster's spec
func (s *ClusterScope) GetDefinedTags() map[string]map[string]interface{} {
	tags := s.OCIClusterBase.GetDefinedTags()
	if tags == nil {
		return make(map[string]map[string]interface{})
	}
	definedTags := make(map[string]map[string]interface{})
	for ns, mapNs := range tags {
		mapValues := make(map[string]interface{})
		for k, v := range mapNs {
			mapValues[k] = v
		}
		definedTags[ns] = mapValues
	}
	return definedTags
}

// GetCompartmentId returns the CompartmentId defined in OCICluster's spec
func (s *ClusterScope) GetCompartmentId() string {
	return s.OCIClusterBase.GetCompartmentId()
}

// APIServerPort returns the APIServerPort to use when creating the load balancer.
func (s *ClusterScope) APIServerPort() int32 {
	if s.Cluster.Spec.ClusterNetwork != nil && s.Cluster.Spec.ClusterNetwork.APIServerPort != nil {
		return *s.Cluster.Spec.ClusterNetwork.APIServerPort
	}
	return ApiServerPort
}

// GetFreeFormTags returns a map of FreeformTags defined in the OCICluster's spec
func (s *ClusterScope) GetFreeFormTags() map[string]string {
	tags := s.OCIClusterBase.GetFreeformTags()
	if tags == nil {
		tags = make(map[string]string)
	}
	tagsAddedByClusterAPI := ociutil.BuildClusterTags(string(s.OCIClusterBase.GetOCIResourceIdentifier()))
	for k, v := range tagsAddedByClusterAPI {
		tags[k] = v
	}
	return tags
}

func (s *ClusterScope) GetOCIClusterBase() base.OCIClusterBase {
	return s.OCIClusterBase
}

func (s *ClusterScope) getDRG() *infrastructurev1beta1.DRG {
	return s.OCIClusterBase.GetDRG()
}

func (s *ClusterScope) getDrgID() *string {
	return s.getDRG().ID
}

func (s *ClusterScope) isPeeringEnabled() bool {
	return s.OCIClusterBase.GetVCNPeering() != nil
}
