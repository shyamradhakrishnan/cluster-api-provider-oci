package scope

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	infrastructurev1beta1 "github.com/oracle/cluster-api-provider-oci/api/v1beta1"
	"github.com/oracle/cluster-api-provider-oci/cloud/ociutil"
	baseclient "github.com/oracle/cluster-api-provider-oci/cloud/services/base"
	"github.com/oracle/cluster-api-provider-oci/cloud/services/containerengine"
	"github.com/oracle/oci-go-sdk/v63/common"
	oke "github.com/oracle/oci-go-sdk/v63/containerengine"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2/klogr"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/kubeconfig"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/secret"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControlPlaneScopeParams defines the params need to create a new ClusterScope
type ControlPlaneScopeParams struct {
	Client                 client.Client
	Logger                 *logr.Logger
	Cluster                *clusterv1.Cluster
	ContainerEngineClient  containerengine.Client
	Region                 string
	OCIAuthConfigProvider  common.ConfigurationProvider
	ClientProvider         *ClientProvider
	OCIManagedControlPlane *infrastructurev1beta1.OCIManagedControlPlane
	OCIClusterBase         OCIClusterBase
	BaseClient             baseclient.BaseClient
}

type ControlPlaneScope struct {
	*logr.Logger
	client                 client.Client
	patchHelper            *patch.Helper
	Cluster                *clusterv1.Cluster
	ContainerEngineClient  containerengine.Client
	BaseClient             baseclient.BaseClient
	Region                 string
	ClientProvider         *ClientProvider
	OCIManagedControlPlane *infrastructurev1beta1.OCIManagedControlPlane
	OCIClusterBase         OCIClusterBase
}

// NewControlPlaneScope creates a ControlPlaneScope given the ControlPlaneScopeParams
func NewControlPlaneScope(params ControlPlaneScopeParams) (*ControlPlaneScope, error) {
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

	return &ControlPlaneScope{
		Logger:                 params.Logger,
		client:                 params.Client,
		Cluster:                params.Cluster,
		ContainerEngineClient:  params.ContainerEngineClient,
		Region:                 params.Region,
		ClientProvider:         params.ClientProvider,
		OCIClusterBase:         params.OCIClusterBase,
		OCIManagedControlPlane: params.OCIManagedControlPlane,
		BaseClient:             params.BaseClient,
	}, nil
}

func (s *ControlPlaneScope) GetOrCreateControlPlane(ctx context.Context) (*oke.Cluster, error) {
	cluster, err := s.GetOKECluster(ctx)
	if err != nil {
		return nil, err
	}
	if cluster != nil {
		s.Logger.Info("Found an existing instance")
		s.OCIManagedControlPlane.Spec.ID = cluster.Id
		return cluster, nil
	}
	endpointConfig := &oke.CreateClusterEndpointConfigDetails{
		SubnetId:          s.getControlPlaneEndpointSubnet(),
		NsgIds:            s.getControlPlaneEndpointNSGList(),
		IsPublicIpEnabled: common.Bool(s.IsControlPlaneEndpointSubnetPublic()),
	}
	details := oke.CreateClusterDetails{
		Name:              common.String(s.GetClusterName()),
		CompartmentId:     common.String(s.OCIClusterBase.GetCompartmentId()),
		VcnId:             s.OCIClusterBase.GetVCN().ID,
		KubernetesVersion: s.OCIManagedControlPlane.Spec.Version,
		FreeformTags:      s.GetFreeFormTags(),
		DefinedTags:       s.GetDefinedTags(),
		EndpointConfig:    endpointConfig,
	}
	createClusterRequest := oke.CreateClusterRequest{
		CreateClusterDetails: details,
	}
	response, err := s.ContainerEngineClient.CreateCluster(ctx, createClusterRequest)

	if err != nil {
		return nil, err
	}
	wrResponse, err := s.ContainerEngineClient.GetWorkRequest(ctx, oke.GetWorkRequestRequest{
		WorkRequestId: response.OpcWorkRequestId,
	})
	if err != nil {
		return nil, err
	}
	resources := wrResponse.Resources
	if len(resources) > 1 {
		return nil, errors.New("more than one resources are affected by the work request to create the cluster")
	}

	clusterId := resources[0].Identifier
	s.OCIManagedControlPlane.Spec.ID = clusterId
	return s.getOKEClusterFromOCID(ctx, clusterId)
}

func (s *ControlPlaneScope) GetOKECluster(ctx context.Context) (*oke.Cluster, error) {
	okeClusterID := s.OCIManagedControlPlane.Spec.ID
	if okeClusterID != nil {
		return s.getOKEClusterFromOCID(ctx, okeClusterID)
	}
	instance, err := s.GetOKEClusterByDisplayName(ctx, s.GetClusterName())
	if err != nil {
		return nil, err
	}
	return instance, err
}

func (s *ControlPlaneScope) getOKEClusterFromOCID(ctx context.Context, clusterID *string) (*oke.Cluster, error) {
	req := oke.GetClusterRequest{ClusterId: clusterID}

	// Send the request using the service client
	resp, err := s.ContainerEngineClient.GetCluster(ctx, req)
	if err != nil {
		return nil, err
	}
	return &resp.Cluster, nil
}

func (s *ControlPlaneScope) GetOKEClusterByDisplayName(ctx context.Context, name string) (*oke.Cluster, error) {
	req := oke.ListClustersRequest{Name: common.String(name),
		CompartmentId: common.String(s.OCIClusterBase.GetCompartmentId())}
	resp, err := s.ContainerEngineClient.ListClusters(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp.Items) == 0 {
		return nil, nil
	}
	for _, cluster := range resp.Items {
		if s.IsResourceCreatedByClusterAPI(cluster.FreeformTags) {
			return s.getOKEClusterFromOCID(ctx, cluster.Id)
		}
	}
	return nil, nil
}

func (s *ControlPlaneScope) IsResourceCreatedByClusterAPI(resourceFreeFormTags map[string]string) bool {
	tagsAddedByClusterAPI := ociutil.BuildClusterTags(s.OCIClusterBase.GetOCIResourceIdentifier())
	for k, v := range tagsAddedByClusterAPI {
		if resourceFreeFormTags[k] != v {
			return false
		}
	}
	return true
}

func (s *ControlPlaneScope) GetClusterName() string {
	if s.OCIManagedControlPlane.Name == "" {
		return s.OCIManagedControlPlane.Name
	}
	return s.OCIClusterBase.GetName()
}

// GetDefinedTags returns a map of DefinedTags defined in the OCICluster's spec
func (s *ControlPlaneScope) GetDefinedTags() map[string]map[string]interface{} {
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

// GetFreeFormTags returns a map of FreeformTags defined in the OCICluster's spec
func (s *ControlPlaneScope) GetFreeFormTags() map[string]string {
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

func (s *ControlPlaneScope) getControlPlaneEndpointSubnet() *string {
	for _, subnet := range s.OCIClusterBase.GetVCN().Subnets {
		if subnet.Role == infrastructurev1beta1.ControlPlaneEndpointRole {
			return subnet.ID
		}
	}
	return nil
}

func (s *ControlPlaneScope) getControlPlaneEndpointNSGList() []string {
	nsgs := make([]string, 0)
	for _, nsg := range s.OCIClusterBase.GetVCN().NetworkSecurityGroups {
		if nsg.Role == infrastructurev1beta1.ControlPlaneEndpointRole {
			nsgs = append(nsgs, *nsg.ID)
		}
	}
	return nsgs
}

func (s *ControlPlaneScope) IsControlPlaneEndpointSubnetPublic() bool {
	for _, subnet := range s.OCIClusterBase.GetVCN().Subnets {
		if subnet.Role == infrastructurev1beta1.ControlPlaneEndpointRole && subnet.Type == infrastructurev1beta1.Public {
			return true
		}
	}
	return false
}

func (s *ControlPlaneScope) DeleteCluster(ctx context.Context) error {
	req := oke.DeleteClusterRequest{ClusterId: s.OCIManagedControlPlane.Spec.ID}
	_, err := s.ContainerEngineClient.DeleteCluster(ctx, req)
	return err
}

func (s *ControlPlaneScope) createCAPIKubeconfigSecret(ctx context.Context, okeCluster *oke.Cluster, clusterRef types.NamespacedName) error {
	controllerOwnerRef := *metav1.NewControllerRef(s.OCIManagedControlPlane, infrastructurev1beta1.GroupVersion.WithKind("OCIManagedControlPlane"))
	req := oke.CreateKubeconfigRequest{ClusterId: s.OCIManagedControlPlane.Spec.ID}
	response, err := s.ContainerEngineClient.CreateKubeconfig(ctx, req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(response.Content)
	config, err := clientcmd.NewClientConfigFromBytes(body)
	if err != nil {
		return err
	}
	rawConfig, err := config.RawConfig()
	if err != nil {
		return err
	}
	userName := getKubeConfigUserName(*okeCluster.Name, false)
	currentCluster := rawConfig.Clusters[rawConfig.CurrentContext]
	currentContext := rawConfig.Contexts[rawConfig.CurrentContext]

	cfg, err := createBaseKubeConfig(userName, currentCluster, currentContext.Cluster, rawConfig.CurrentContext)

	token, err := s.BaseClient.GenerateToken(ctx, *okeCluster.Id)
	if err != nil {
		return fmt.Errorf("generating presigned token: %w", err)
	}

	cfg.AuthInfos = map[string]*api.AuthInfo{
		userName: {
			Token: token,
		},
	}

	out, err := clientcmd.Write(*cfg)
	s.Logger.Info(fmt.Sprintf("kubeconfig is %s", string(out)))
	if err != nil {
		return errors.Wrap(err, "failed to serialize config to yaml")
	}

	kubeconfigSecret := kubeconfig.GenerateSecretWithOwner(clusterRef, out, controllerOwnerRef)
	if err := s.client.Create(ctx, kubeconfigSecret); err != nil {
		return errors.Wrap(err, "failed to create kubeconfig secret")
	}
	return err

}

func createBaseKubeConfig(userName string, kubeconfigCluster *api.Cluster, clusterName string, contextName string) (*api.Config, error) {

	cfg := &api.Config{
		APIVersion: api.SchemeGroupVersion.Version,
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   kubeconfigCluster.Server,
				CertificateAuthorityData: kubeconfigCluster.CertificateAuthorityData,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		CurrentContext: contextName,
	}

	return cfg, nil
}

func (s *ControlPlaneScope) ReconcileKubeconfig(ctx context.Context, okeCluster *oke.Cluster) error {
	clusterRef := types.NamespacedName{
		Name:      s.Cluster.Name,
		Namespace: s.Cluster.Namespace,
	}

	// Create the kubeconfig used by CAPI
	configSecret, err := secret.GetFromNamespacedName(ctx, s.client, clusterRef, secret.Kubeconfig)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get kubeconfig secret")
		}

		if createErr := s.createCAPIKubeconfigSecret(
			ctx,
			okeCluster,
			clusterRef,
		); createErr != nil {
			return fmt.Errorf("creating kubeconfig secret: %w", err)
		}
	} else if updateErr := s.updateCAPIKubeconfigSecret(ctx, configSecret, okeCluster); updateErr != nil {
		return fmt.Errorf("updating kubeconfig secret: %w", err)
	}

	// Set initialized to true to indicate the kubconfig has been created
	s.OCIManagedControlPlane.Status.Initialized = true

	return nil
}

func (s *ControlPlaneScope) updateCAPIKubeconfigSecret(ctx context.Context, configSecret *corev1.Secret, okeCluster *oke.Cluster) error {
	data, ok := configSecret.Data[secret.KubeconfigDataName]
	if !ok {
		return errors.Errorf("missing key %q in secret data", secret.KubeconfigDataName)
	}

	config, err := clientcmd.Load(data)
	if err != nil {
		return errors.Wrap(err, "failed to convert kubeconfig Secret into a clientcmdapi.Config")
	}

	token, err := s.BaseClient.GenerateToken(ctx, *okeCluster.Id)
	if err != nil {
		return fmt.Errorf("generating presigned token: %w", err)
	}

	if err != nil {
		return fmt.Errorf("generating presigned token: %w", err)
	}

	userName := getKubeConfigUserName(*okeCluster.Name, false)
	config.AuthInfos[userName].Token = token

	out, err := clientcmd.Write(*config)
	if err != nil {
		return errors.Wrap(err, "failed to serialize config to yaml")
	}

	configSecret.Data[secret.KubeconfigDataName] = out

	err = s.client.Update(ctx, configSecret)
	if err != nil {
		return fmt.Errorf("updating kubeconfig secret: %w", err)
	}

	return nil
}

func getKubeConfigUserName(clusterName string, isUser bool) string {
	if isUser {
		return fmt.Sprintf("%s-user", clusterName)
	}

	return fmt.Sprintf("%s-capi-admin", clusterName)
}
