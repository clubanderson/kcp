/*
Copyright 2022 The KCP Authors.

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

package apiexport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	kcpdynamic "github.com/kcp-dev/client-go/dynamic"
	kcpkubernetesclientset "github.com/kcp-dev/client-go/kubernetes"
	"github.com/kcp-dev/logicalcluster/v3"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apiextensions-apiserver/pkg/apihelpers"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extensionsapiserver "k8s.io/apiextensions-apiserver/pkg/apiserver"
	kcpapiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/kcp/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	"github.com/kcp-dev/kcp/config/helpers"
	"github.com/kcp-dev/kcp/pkg/apis/apis"
	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	"github.com/kcp-dev/kcp/pkg/apis/core"
	"github.com/kcp-dev/kcp/pkg/apis/scheduling"
	schedulingv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/scheduling/v1alpha1"
	"github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/util/conditions"
	workloadv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/workload/v1alpha1"
	kcpclientset "github.com/kcp-dev/kcp/pkg/client/clientset/versioned/cluster"
	"github.com/kcp-dev/kcp/test/e2e/fixtures/wildwest/apis/wildwest"
	"github.com/kcp-dev/kcp/test/e2e/framework"
)

func TestAPIExportAuthorizers(t *testing.T) {
	t.Parallel()
	framework.Suite(t, "control-plane")

	server := framework.SharedKcpServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	orgPath, _ := framework.NewOrganizationFixture(t, server)

	// see https://docs.google.com/drawings/d/1_sOiFZReAfypuUDyHS9rwpxbZgJNJuxdvbgXXgu2KAQ/edit for topology
	serviceProvider1Path, _ := framework.NewWorkspaceFixture(t, server, orgPath, framework.WithName("service-provider-1"))
	serviceProvider2Path, _ := framework.NewWorkspaceFixture(t, server, orgPath, framework.WithName("service-provider-2"))
	tenantPath, tenantWorkspace := framework.NewWorkspaceFixture(t, server, orgPath, framework.WithName("tenant"))
	tenantShadowCRDPath, tenantShadowCRDWorkspace := framework.NewWorkspaceFixture(t, server, orgPath, framework.WithName("tenant-shadowed-crd"))

	cfg := server.BaseConfig(t)

	serviceProvider1Admin := server.ClientCAUserConfig(t, rest.CopyConfig(cfg), "service-provider-1-admin")
	serviceProvider2Admin := server.ClientCAUserConfig(t, rest.CopyConfig(cfg), "service-provider-2-admin")
	tenantUser := server.ClientCAUserConfig(t, rest.CopyConfig(cfg), "tenant-user")

	kubeClient, err := kcpkubernetesclientset.NewForConfig(rest.CopyConfig(cfg))
	require.NoError(t, err)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, orgPath, []string{"service-provider-1-admin", "service-provider-2-admin", "tenant-user"}, nil, false)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, serviceProvider1Path, []string{"service-provider-1-admin"}, nil, true)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, serviceProvider2Path, []string{"service-provider-2-admin"}, nil, true)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, tenantPath, []string{"tenant-user"}, nil, true)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, tenantShadowCRDPath, []string{"tenant-user"}, nil, true)

	t.Logf("install sherriffs API resource schema, API export, permissions for tenant-user to be able to bind to the export in service provider workspace %q", serviceProvider1Path)
	require.NoError(t, apply(t, ctx, serviceProvider1Path, serviceProvider1Admin,
		&apisv1alpha1.APIResourceSchema{
			ObjectMeta: metav1.ObjectMeta{Name: "today.sheriffs.wild.wild.west"},
			Spec: apisv1alpha1.APIResourceSchemaSpec{
				Group: "wild.wild.west",
				Names: apiextensionsv1.CustomResourceDefinitionNames{Plural: "sheriffs", Singular: "sheriff", Kind: "Sheriff", ListKind: "SheriffList"},
				Scope: "Namespaced",
				Versions: []apisv1alpha1.APIResourceVersion{
					{Name: "v1alpha1", Served: true, Storage: true, Schema: runtime.RawExtension{Raw: []byte(`{"type":"object"}`)}},
				},
			},
		},
		&apisv1alpha1.APIExport{
			ObjectMeta: metav1.ObjectMeta{Name: "wild.wild.west"},
			Spec: apisv1alpha1.APIExportSpec{
				LatestResourceSchemas:   []string{"today.sheriffs.wild.wild.west"},
				MaximalPermissionPolicy: &apisv1alpha1.MaximalPermissionPolicy{Local: &apisv1alpha1.LocalAPIExportPolicy{}},
			},
		},

		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant-user-bind-apiexport"},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{"apis.kcp.io"}, ResourceNames: []string{"wild.wild.west"}, Resources: []string{"apiexports"}, Verbs: []string{"bind"}},
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant-user-bind-apiexport"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: "tenant-user"}},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.SchemeGroupVersion.Group, Kind: "ClusterRole", Name: "tenant-user-bind-apiexport"},
		},
	))

	t.Logf("get the sheriffs apiexport's generated identity hash")
	sherriffsIdentityHash := ""
	serviceProvider1AdminClient, err := kcpclientset.NewForConfig(serviceProvider1Admin)
	require.NoError(t, err)
	framework.Eventually(t, func() (done bool, str string) {
		sheriffExport, err := serviceProvider1AdminClient.Cluster(serviceProvider1Path).ApisV1alpha1().APIExports().Get(ctx, "wild.wild.west", metav1.GetOptions{})
		if err != nil {
			return false, fmt.Sprintf("error while waiting to get API export: %v", err)
		}
		if conditions.IsTrue(sheriffExport, apisv1alpha1.APIExportIdentityValid) {
			sherriffsIdentityHash = sheriffExport.Status.IdentityHash
			return true, ""
		}
		return false, fmt.Sprintf("waiting for API export identity to be valid: %+v", conditions.Get(sheriffExport, apisv1alpha1.APIExportIdentityValid))
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "could not wait for APIExport to be valid with identity hash")
	t.Logf("Found identity hash: %v", sherriffsIdentityHash)

	t.Logf("install cowboys API resource schema, API export, and permissions for tenant-user to be able to bind to the export in second service provider workspace %q", serviceProvider2Path)
	require.NoError(t, apply(t, ctx, serviceProvider2Path, serviceProvider2Admin,
		&apisv1alpha1.APIResourceSchema{
			ObjectMeta: metav1.ObjectMeta{Name: "today.cowboys.wildwest.dev"},
			Spec: apisv1alpha1.APIResourceSchemaSpec{
				Group: "wildwest.dev",
				Names: apiextensionsv1.CustomResourceDefinitionNames{Plural: "cowboys", Singular: "cowboy", Kind: "Cowboy", ListKind: "CowboyList"},
				Scope: "Namespaced",
				Versions: []apisv1alpha1.APIResourceVersion{
					{Name: "v1alpha1", Served: true, Storage: true, Schema: runtime.RawExtension{Raw: []byte(`{"type":"object"}`)}},
				},
			},
		},
		&apisv1alpha1.APIExport{
			ObjectMeta: metav1.ObjectMeta{Name: "today-cowboys"},
			Spec: apisv1alpha1.APIExportSpec{
				LatestResourceSchemas:   []string{"today.cowboys.wildwest.dev"},
				MaximalPermissionPolicy: &apisv1alpha1.MaximalPermissionPolicy{Local: &apisv1alpha1.LocalAPIExportPolicy{}},
				PermissionClaims: []apisv1alpha1.PermissionClaim{
					{
						GroupResource: apisv1alpha1.GroupResource{Resource: "configmaps"},
						All:           true,
					},
					{
						GroupResource: apisv1alpha1.GroupResource{Group: "wild.wild.west", Resource: "sheriffs"},
						IdentityHash:  sherriffsIdentityHash,
						All:           true,
					},
				},
			},
		},

		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant-user-bind"},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{"apis.kcp.io"}, ResourceNames: []string{"today-cowboys"}, Resources: []string{"apiexports"}, Verbs: []string{"bind"}},
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant-user-bind"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: "tenant-user"}},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.SchemeGroupVersion.Group, Kind: "ClusterRole", Name: "tenant-user-bind"},
		},

		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant-user-maximum-permission-policy"},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{"wildwest.dev"}, Resources: []string{"cowboys"}, Verbs: []string{"delete", "create", "list", "watch", "get", "patch"}},
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "tenant-user-maximum-permission-policy"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: "apis.kcp.io:binding:tenant-user"}},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.SchemeGroupVersion.Group, Kind: "ClusterRole", Name: "tenant-user-maximum-permission-policy"},
		},
	))

	t.Logf("bind cowboys and claimed sherriffs in the tenant workspace %q", tenantPath)
	require.NoError(t, apply(t, ctx, tenantPath, tenantUser,
		&apisv1alpha1.APIBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "wild.wild.west",
			},
			Spec: apisv1alpha1.APIBindingSpec{
				Reference: apisv1alpha1.BindingReference{
					Export: &apisv1alpha1.ExportBindingReference{
						Path: serviceProvider1Path.String(),
						Name: "wild.wild.west",
					},
				},
			},
		},
		&apisv1alpha1.APIBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cowboys",
			},
			Spec: apisv1alpha1.APIBindingSpec{
				PermissionClaims: []apisv1alpha1.AcceptablePermissionClaim{
					{
						PermissionClaim: apisv1alpha1.PermissionClaim{
							GroupResource: apisv1alpha1.GroupResource{Resource: "configmaps"},
							All:           true,
						},
						State: apisv1alpha1.ClaimAccepted,
					},
					{
						PermissionClaim: apisv1alpha1.PermissionClaim{
							GroupResource: apisv1alpha1.GroupResource{Group: "wild.wild.west", Resource: "sheriffs"},
							IdentityHash:  sherriffsIdentityHash,
							All:           true,
						},
						State: apisv1alpha1.ClaimAccepted,
					},
				},
				Reference: apisv1alpha1.BindingReference{
					Export: &apisv1alpha1.ExportBindingReference{
						Path: serviceProvider2Path.String(),
						Name: "today-cowboys",
					},
				},
			},
		},
	))

	t.Logf("Make sure [%q, %q] API groups shows up in consumer workspace %q group discovery", wildwest.GroupName, "wild.wild.west", tenantPath)
	tenantUserWorkspaceKcpClient, err := kcpclientset.NewForConfig(tenantUser)
	require.NoError(t, err)
	framework.Eventually(t, func() (success bool, reason string) {
		groups, err := tenantUserWorkspaceKcpClient.Cluster(tenantPath).Discovery().ServerGroups()
		if err != nil {
			return false, fmt.Sprintf("error retrieving consumer workspace %q group discovery: %v", tenantPath, err)
		}
		return groupExists(groups, wildwest.GroupName) && groupExists(groups, "wild.wild.west"), ""
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "discovery failed")

	t.Logf("Install cowboys CRD and also bind the conflicting cowboys API export in tenant workspace %q", tenantShadowCRDPath)
	require.NoError(t, apply(t, ctx, tenantShadowCRDPath, tenantUser,
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: "cowboys.wildwest.dev"},
			Spec: apiextensionsv1.CustomResourceDefinitionSpec{
				Group: "wildwest.dev",
				Names: apiextensionsv1.CustomResourceDefinitionNames{Plural: "cowboys", Singular: "cowboy", Kind: "Cowboy", ListKind: "CowboyList"},
				Scope: "Namespaced",
				Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
					{
						Name: "v1alpha1", Served: true, Storage: true,
						Schema: &apiextensionsv1.CustomResourceValidation{
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
								Type: "object",
							}},
					},
				},
			},
		},
	))

	t.Logf("Waiting for cowboys CRD to be ready in tenant workspace %q", tenantShadowCRDPath)
	tenantUserAPIExtensionsClient, err := kcpapiextensionsclientset.NewForConfig(tenantUser)
	require.NoError(t, err)
	framework.Eventually(t, func() (bool, string) {
		cowboysCRD, err := tenantUserAPIExtensionsClient.Cluster(tenantShadowCRDPath).ApiextensionsV1().CustomResourceDefinitions().Get(ctx, "cowboys.wildwest.dev", metav1.GetOptions{})
		if err != nil {
			return false, fmt.Sprintf("error creating API binding: %v", err)
		}
		if apihelpers.IsCRDConditionTrue(cowboysCRD, apiextensionsv1.Established) {
			return true, ""
		}
		return false, "waiting for cowboys CRD to become established"
	}, wait.ForeverTestTimeout, time.Millisecond*100, "waiting for cowboys CRD to become established failed")

	t.Logf("Create a cowboys APIBinding in consumer workspace %q that points to the today-cowboys export from %q but shadows a local cowboys CRD at the same time", tenantShadowCRDPath, serviceProvider2Path)
	require.NoError(t, apply(t, ctx, tenantShadowCRDPath, tenantUser,
		&apisv1alpha1.APIBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cowboys",
			},
			Spec: apisv1alpha1.APIBindingSpec{
				PermissionClaims: []apisv1alpha1.AcceptablePermissionClaim{
					{
						PermissionClaim: apisv1alpha1.PermissionClaim{
							GroupResource: apisv1alpha1.GroupResource{Resource: "configmaps"},
							All:           true,
						},
						State: apisv1alpha1.ClaimAccepted,
					},
					{
						PermissionClaim: apisv1alpha1.PermissionClaim{
							GroupResource: apisv1alpha1.GroupResource{Group: "wild.wild.west", Resource: "sheriffs"},
							IdentityHash:  sherriffsIdentityHash,
							All:           true,
						},
						State: apisv1alpha1.ClaimAccepted,
					},
				},
				Reference: apisv1alpha1.BindingReference{
					Export: &apisv1alpha1.ExportBindingReference{
						Path: serviceProvider2Path.String(),
						Name: "today-cowboys",
					},
				},
			},
		},
	))

	t.Logf("Waiting for cowboys APIBinding in consumer workspace %q to have the condition %q mentioning the conflict with the shadowing local cowboys CRD", tenantShadowCRDPath, apisv1alpha1.BindingUpToDate)
	tenantUserKcpClient, err := kcpclientset.NewForConfig(tenantUser)
	require.NoError(t, err)
	framework.Eventually(t, func() (bool, string) {
		binding, err := tenantUserKcpClient.Cluster(tenantShadowCRDPath).ApisV1alpha1().APIBindings().Get(ctx, "cowboys", metav1.GetOptions{})
		if err != nil {
			return false, fmt.Sprintf("error creating API binding: %v", err)
		}
		condition := conditions.Get(binding, apisv1alpha1.BindingUpToDate)
		if condition == nil {
			return false, "binding condition not found"
		}
		if strings.Contains(condition.Message, `overlaps with "cowboys.wildwest.dev" CustomResourceDefinition`) {
			return true, ""
		}
		return false, fmt.Sprintf("CRD conflict condition not yet met: %q", condition.Message)
	}, wait.ForeverTestTimeout, time.Millisecond*100, "api binding creation failed")

	require.NoError(t, apply(t, ctx, tenantPath, tenantUser, `
apiVersion: wildwest.dev/v1alpha1
kind: Cowboy
metadata:
  name: cowboy-via-api-binding
  namespace: default
`))

	require.NoError(t, apply(t, ctx, tenantShadowCRDPath, tenantUser, `
apiVersion: wildwest.dev/v1alpha1
kind: Cowboy
metadata:
  name: cowboy-via-crd
  namespace: default
`))

	t.Logf("get virtual workspace client for \"today-cowboys\" APIExport in workspace %q", serviceProvider2Path)
	var apiExport *apisv1alpha1.APIExport
	serviceProvider2AdminClient, err := kcpclientset.NewForConfig(serviceProvider2Admin)
	require.NoError(t, err)
	framework.Eventually(t, func() (bool, string) {
		var err error
		apiExport, err = serviceProvider2AdminClient.Cluster(serviceProvider2Path).ApisV1alpha1().APIExports().Get(ctx, "today-cowboys", metav1.GetOptions{})
		if err != nil {
			return false, fmt.Sprintf("waiting on apiexport to be available %v", err.Error())
		}
		//nolint:staticcheck // SA1019 VirtualWorkspaces is deprecated but not removed yet
		if len(apiExport.Status.VirtualWorkspaces) > 0 {
			return true, ""
		}
		return false, "waiting on virtual workspace to be ready"
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting on virtual workspace to be ready")

	serviceProvider2AdminApiExportVWCfg := rest.CopyConfig(serviceProvider2Admin)
	//nolint:staticcheck // SA1019 VirtualWorkspaces is deprecated but not removed yet
	serviceProvider2AdminApiExportVWCfg.Host = apiExport.Status.VirtualWorkspaces[0].URL
	user2DynamicVWClient, err := kcpdynamic.NewForConfig(serviceProvider2AdminApiExportVWCfg)
	require.NoError(t, err)

	t.Logf("verify that service-provider-2-admin cannot list sherrifs resources via virtual apiexport apiserver because we have no local maximal permissions yet granted")
	_, err = user2DynamicVWClient.Resource(schema.GroupVersionResource{Version: "v1", Resource: "sheriffs", Group: "wild.wild.west"}).List(ctx, metav1.ListOptions{})
	require.ErrorContains(
		t, err,
		`sheriffs.wild.wild.west is forbidden: User "service-provider-2-admin" cannot list resource "sheriffs" in API group "wild.wild.west" at the cluster scope: access denied`,
		"service-provider-2-admin must not be allowed to list sheriff resources")

	_, err = user2DynamicVWClient.Resource(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}).List(ctx, metav1.ListOptions{})
	require.NoError(t, err, "service-provider-2-admin must be allowed to list native types")

	require.NoError(t, apply(t, ctx, serviceProvider1Path, serviceProvider1Admin,
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{Name: "service-provider-2-admin-maximum-permission-policy"},
			Rules: []rbacv1.PolicyRule{
				{APIGroups: []string{"wild.wild.west"}, Resources: []string{"sheriffs"}, Verbs: []string{"delete", "create", "list", "watch", "get", "patch"}},
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "service-provider-2-admin-maximum-permission-policy"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: "apis.kcp.io:binding:service-provider-2-admin"}},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.SchemeGroupVersion.Group, Kind: "ClusterRole", Name: "service-provider-2-admin-maximum-permission-policy"},
		},
	))

	t.Logf("verify that service-provider-2-admin can lists all claimed resources using a wildcard request")
	claimedGVRs := []schema.GroupVersionResource{
		{Version: "v1", Resource: "configmaps"},
		{Version: "v1alpha1", Resource: "sheriffs", Group: "wild.wild.west"},
	}
	framework.Eventually(t, func() (success bool, reason string) {
		for _, gvr := range claimedGVRs {
			_, err := user2DynamicVWClient.Resource(gvr).List(ctx, metav1.ListOptions{})
			if err != nil {
				return false, fmt.Sprintf("error while waiting to list %q: %v", gvr, err)
			}
		}
		return true, ""
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "listing claimed resources failed")

	t.Logf("verify that service-provider-2-admin can lists sherriffs resources in the tenant workspace %q via the virtual apiexport apiserver", tenantPath)
	framework.Eventually(t, func() (success bool, reason string) {
		_, err = user2DynamicVWClient.Cluster(logicalcluster.Name(tenantWorkspace.Spec.Cluster).Path()).Resource(schema.GroupVersionResource{Version: "v1alpha1", Resource: "sheriffs", Group: "wild.wild.west"}).List(ctx, metav1.ListOptions{})
		if err != nil {
			return false, fmt.Sprintf("error while waiting to list sherriffs: %v", err)
		}
		return true, ""
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "listing claimed resources failed")

	t.Logf("verify that service-provider-2-admin cannot lists CRD shadowed sherriffs resources in the tenant workspace %q via the virtual apiexport apiserver", tenantShadowCRDPath)
	_, err = user2DynamicVWClient.Cluster(logicalcluster.Name(tenantShadowCRDWorkspace.Spec.Cluster).Path()).Resource(schema.GroupVersionResource{Version: "v1alpha1", Resource: "cowboys", Group: "wildwest.dev"}).List(ctx, metav1.ListOptions{})
	require.Error(t, err, "expected error, got none")
	require.True(t, errors.IsNotFound(err))
}

var scheme *runtime.Scheme

func init() {
	scheme = runtime.NewScheme()
	_ = apisv1alpha1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	_ = apiextensionsv1.AddToScheme(scheme)
}

func apply(t *testing.T, ctx context.Context, workspace logicalcluster.Path, cfg *rest.Config, manifests ...any) error {
	t.Helper()
	discoveryClient, err := kcpkubernetesclientset.NewForConfig(cfg)
	require.NoError(t, err)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient.Cluster(workspace).Discovery()))

	for _, manifest := range manifests {
		obj, gvk := func() (*unstructured.Unstructured, *schema.GroupVersionKind) {
			switch value := manifest.(type) {
			case string:
				result, gvk, err := extensionsapiserver.Codecs.UniversalDeserializer().Decode([]byte(value), nil, &unstructured.Unstructured{})
				require.NoError(t, err)

				obj, ok := result.(*unstructured.Unstructured)
				if !ok {
					t.Fatalf("decoded into incorrect type, got %T, wanted %T", obj, &unstructured.Unstructured{})
				}
				return obj, gvk
			case runtime.Object:
				ro := manifest.(runtime.Object)
				gvks, _, err := scheme.ObjectKinds(ro)
				require.NoError(t, err)
				ro.GetObjectKind().SetGroupVersionKind(gvks[0])
				o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ro)
				require.NoError(t, err)
				return &unstructured.Unstructured{Object: o}, &gvks[0]
			default:
				panic("unsupported type")
			}
		}()

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		require.NoError(t, err)

		dynamicClient, err := kcpdynamic.NewForConfig(cfg)
		require.NoError(t, err)
		var dynamicResource dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			t.Logf(`applying %q workspace %q namespace %q name %q`, gvk.String(), workspace.String(), obj.GetNamespace(), obj.GetName())
			dynamicResource = dynamicClient.Cluster(workspace).Resource(mapping.Resource).Namespace(obj.GetNamespace())
		} else {
			t.Logf(`applying %q workspace %q name %q`, gvk.String(), workspace.String(), obj.GetName())
			dynamicResource = dynamicClient.Cluster(workspace).Resource(mapping.Resource)
		}

		bytes, err := json.Marshal(obj)
		require.NoError(t, err)

		_, err = dynamicResource.Patch(ctx, obj.GetName(), types.ApplyPatchType, bytes, metav1.PatchOptions{
			FieldManager: t.Name(),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func TestRootAPIExportAuthorizers(t *testing.T) {
	t.Parallel()
	framework.Suite(t, "control-plane")

	server := framework.SharedKcpServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	orgPath, _ := framework.NewOrganizationFixture(t, server)

	servicePath, _ := framework.NewWorkspaceFixture(t, server, orgPath, framework.WithName("provider"))
	userPath, userWorkspace := framework.NewWorkspaceFixture(t, server, orgPath, framework.WithName("consumer"))
	userClusterName := logicalcluster.Name(userWorkspace.Spec.Cluster)

	cfg := server.BaseConfig(t)

	kubeClient, err := kcpkubernetesclientset.NewForConfig(rest.CopyConfig(cfg))
	require.NoError(t, err)
	kcpClient, err := kcpclientset.NewForConfig(rest.CopyConfig(cfg))
	require.NoError(t, err)

	providerUser := "user-1"
	consumerUser := "user-2"

	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, orgPath, []string{providerUser, consumerUser}, nil, false)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, servicePath, []string{providerUser}, nil, true)
	framework.AdmitWorkspaceAccess(ctx, t, kubeClient, userPath, []string{consumerUser}, nil, true)

	serviceKcpClient, err := kcpclientset.NewForConfig(framework.StaticTokenUserConfig(providerUser, rest.CopyConfig(cfg)))
	require.NoError(t, err)
	serviceDynamicClusterClient, err := kcpdynamic.NewForConfig(framework.StaticTokenUserConfig(providerUser, rest.CopyConfig(cfg)))
	require.NoError(t, err)

	userKcpClient, err := kcpclientset.NewForConfig(framework.StaticTokenUserConfig(consumerUser, rest.CopyConfig(cfg)))
	require.NoError(t, err)

	t.Logf("Install APIResourceSchema into service provider workspace %q", servicePath)
	serviceProviderKcpClient, err := kcpclientset.NewForConfig(framework.StaticTokenUserConfig(providerUser, rest.CopyConfig(cfg)))
	require.NoError(t, err)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(serviceProviderKcpClient.Cluster(servicePath).Discovery()))
	err = helpers.CreateResourceFromFS(ctx, serviceDynamicClusterClient.Cluster(servicePath), mapper, nil, "apiresourceschema_cowboys.yaml", testFiles)
	require.NoError(t, err)

	t.Logf("Get the root scheduling APIExport's identity hash")
	schedulingAPIExport, err := kcpClient.Cluster(core.RootCluster.Path()).ApisV1alpha1().APIExports().Get(ctx, "scheduling.kcp.io", metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, conditions.IsTrue(schedulingAPIExport, apisv1alpha1.APIExportIdentityValid))
	identityHash := schedulingAPIExport.Status.IdentityHash
	require.NotNil(t, identityHash)

	t.Logf("Create an APIExport for APIResourceSchema in service provider %q", servicePath)
	apiExport := &apisv1alpha1.APIExport{
		ObjectMeta: metav1.ObjectMeta{
			Name: "today-cowboys",
		},
		Spec: apisv1alpha1.APIExportSpec{
			LatestResourceSchemas: []string{"today.cowboys.wildwest.dev"},
			PermissionClaims: []apisv1alpha1.PermissionClaim{
				{
					GroupResource: apisv1alpha1.GroupResource{Group: scheduling.GroupName, Resource: "placements"},
					IdentityHash:  identityHash,
					All:           true,
				},
			},
		},
	}
	apiExport, err = serviceKcpClient.Cluster(servicePath).ApisV1alpha1().APIExports().Create(ctx, apiExport, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Logf("Grant user to be able to bind service API export from workspace %q", servicePath)
	cr, crb := createClusterRoleAndBindings(
		consumerUser,
		consumerUser, "User",
		[]string{"bind"},
		apis.GroupName, "apiexports", apiExport.Name,
	)
	_, err = kubeClient.Cluster(servicePath).RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = kubeClient.Cluster(servicePath).RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Logf("Create an APIBinding in consumer workspace %q that points to the service APIExport from %q", userPath, servicePath)
	apiBinding := &apisv1alpha1.APIBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cowboys",
		},
		Spec: apisv1alpha1.APIBindingSpec{
			Reference: apisv1alpha1.BindingReference{
				Export: &apisv1alpha1.ExportBindingReference{
					Path: servicePath.String(),
					Name: apiExport.Name,
				},
			},
			PermissionClaims: []apisv1alpha1.AcceptablePermissionClaim{
				{
					PermissionClaim: apisv1alpha1.PermissionClaim{
						GroupResource: apisv1alpha1.GroupResource{Group: scheduling.GroupName, Resource: "placements"},
						IdentityHash:  identityHash,
						All:           true,
					},
					State: apisv1alpha1.ClaimAccepted,
				},
			},
		},
	}
	framework.Eventually(t, func() (bool, string) {
		_, err := userKcpClient.Cluster(userPath).ApisV1alpha1().APIBindings().Create(ctx, apiBinding, metav1.CreateOptions{})
		if err != nil {
			return false, fmt.Sprintf("error creating API binding: %v", err)
		}
		return true, ""
	}, wait.ForeverTestTimeout, time.Millisecond*100, "api binding creation failed")

	t.Logf("Wait for the binding to be ready")
	framework.Eventually(t, func() (bool, string) {
		binding, err := userKcpClient.Cluster(userPath).ApisV1alpha1().APIBindings().Get(ctx, apiBinding.Name, metav1.GetOptions{})
		require.NoError(t, err, "error getting binding %s", binding.Name)
		condition := conditions.Get(binding, apisv1alpha1.InitialBindingCompleted)
		if condition == nil {
			return false, fmt.Sprintf("no %s condition exists", apisv1alpha1.InitialBindingCompleted)
		}
		if condition.Status == corev1.ConditionTrue {
			return true, ""
		}
		return false, fmt.Sprintf("not done waiting for the binding to be initially bound, reason: %v - message: %v", condition.Reason, condition.Message)
	}, wait.ForeverTestTimeout, time.Millisecond*100)

	t.Logf("Get virtual workspace client for service APIExport in workspace %q", servicePath)
	var export *apisv1alpha1.APIExport
	framework.Eventually(t, func() (bool, string) {
		var err error
		export, err = serviceKcpClient.Cluster(servicePath).ApisV1alpha1().APIExports().Get(ctx, apiExport.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Sprintf("waiting on APIExport to be available %v", err.Error())
		}
		//nolint:staticcheck // SA1019 VirtualWorkspaces is deprecated but not removed yet
		if len(export.Status.VirtualWorkspaces) > 0 {
			return true, ""
		}
		return false, "waiting on virtual workspace to be ready"
	}, wait.ForeverTestTimeout, 100*time.Millisecond, "waiting on virtual workspace to be ready")

	serviceAPIExportVWCfg := framework.StaticTokenUserConfig(providerUser, rest.CopyConfig(cfg))
	//nolint:staticcheck // SA1019 VirtualWorkspaces is deprecated but not removed yet
	serviceAPIExportVWCfg.Host = export.Status.VirtualWorkspaces[0].URL
	serviceDynamicVWClient, err := kcpdynamic.NewForConfig(serviceAPIExportVWCfg)
	require.NoError(t, err)

	t.Logf("Verify that service user can create a claimed resource in user workspace")
	placement := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": schedulingv1alpha1.SchemeGroupVersion.String(),
			"kind":       "Placement",
			"metadata": map[string]interface{}{
				"name": "default",
			},
			"spec": map[string]interface{}{
				"locationResource": map[string]interface{}{
					"group":    workloadv1alpha1.SchemeGroupVersion.Group,
					"resource": "synctargets",
					"version":  workloadv1alpha1.SchemeGroupVersion.Version,
				},
			},
		},
	}
	_, err = serviceDynamicVWClient.Cluster(userClusterName.Path()).
		Resource(schedulingv1alpha1.SchemeGroupVersion.WithResource("placements")).
		Create(ctx, placement, metav1.CreateOptions{})
	require.NoError(t, err)

	t.Logf("Verify that consumer user can get the created resource in user workspace")
	_, err = userKcpClient.Cluster(userClusterName.Path()).SchedulingV1alpha1().Placements().Get(ctx, placement.GetName(), metav1.GetOptions{})
	require.NoError(t, err)
}
