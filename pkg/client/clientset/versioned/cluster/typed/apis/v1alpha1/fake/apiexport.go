//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The KCP Authors.

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

// Code generated by kcp code-generator. DO NOT EDIT.

package v1alpha1

import (
	"context"

	"github.com/kcp-dev/logicalcluster/v2"

	kcptesting "github.com/kcp-dev/client-go/third_party/k8s.io/client-go/testing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"

	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
	apisv1alpha1client "github.com/kcp-dev/kcp/pkg/client/clientset/versioned/typed/apis/v1alpha1"
)

var aPIExportsResource = schema.GroupVersionResource{Group: "apis.kcp.dev", Version: "v1alpha1", Resource: "apiexports"}
var aPIExportsKind = schema.GroupVersionKind{Group: "apis.kcp.dev", Version: "v1alpha1", Kind: "APIExport"}

type aPIExportsClusterClient struct {
	*kcptesting.Fake
}

// Cluster scopes the client down to a particular cluster.
func (c *aPIExportsClusterClient) Cluster(cluster logicalcluster.Name) apisv1alpha1client.APIExportInterface {
	if cluster == logicalcluster.Wildcard {
		panic("A specific cluster must be provided when scoping, not the wildcard.")
	}

	return &aPIExportsClient{Fake: c.Fake, Cluster: cluster}
}

// List takes label and field selectors, and returns the list of APIExports that match those selectors across all clusters.
func (c *aPIExportsClusterClient) List(ctx context.Context, opts metav1.ListOptions) (*apisv1alpha1.APIExportList, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootListAction(aPIExportsResource, aPIExportsKind, logicalcluster.Wildcard, opts), &apisv1alpha1.APIExportList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &apisv1alpha1.APIExportList{ListMeta: obj.(*apisv1alpha1.APIExportList).ListMeta}
	for _, item := range obj.(*apisv1alpha1.APIExportList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested APIExports across all clusters.
func (c *aPIExportsClusterClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.InvokesWatch(kcptesting.NewRootWatchAction(aPIExportsResource, logicalcluster.Wildcard, opts))
}

type aPIExportsClient struct {
	*kcptesting.Fake
	Cluster logicalcluster.Name
}

func (c *aPIExportsClient) Create(ctx context.Context, aPIExport *apisv1alpha1.APIExport, opts metav1.CreateOptions) (*apisv1alpha1.APIExport, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootCreateAction(aPIExportsResource, c.Cluster, aPIExport), &apisv1alpha1.APIExport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apisv1alpha1.APIExport), err
}

func (c *aPIExportsClient) Update(ctx context.Context, aPIExport *apisv1alpha1.APIExport, opts metav1.UpdateOptions) (*apisv1alpha1.APIExport, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootUpdateAction(aPIExportsResource, c.Cluster, aPIExport), &apisv1alpha1.APIExport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apisv1alpha1.APIExport), err
}

func (c *aPIExportsClient) UpdateStatus(ctx context.Context, aPIExport *apisv1alpha1.APIExport, opts metav1.UpdateOptions) (*apisv1alpha1.APIExport, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootUpdateSubresourceAction(aPIExportsResource, c.Cluster, "status", aPIExport), &apisv1alpha1.APIExport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apisv1alpha1.APIExport), err
}

func (c *aPIExportsClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.Invokes(kcptesting.NewRootDeleteActionWithOptions(aPIExportsResource, c.Cluster, name, opts), &apisv1alpha1.APIExport{})
	return err
}

func (c *aPIExportsClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := kcptesting.NewRootDeleteCollectionAction(aPIExportsResource, c.Cluster, listOpts)

	_, err := c.Fake.Invokes(action, &apisv1alpha1.APIExportList{})
	return err
}

func (c *aPIExportsClient) Get(ctx context.Context, name string, options metav1.GetOptions) (*apisv1alpha1.APIExport, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootGetAction(aPIExportsResource, c.Cluster, name), &apisv1alpha1.APIExport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apisv1alpha1.APIExport), err
}

// List takes label and field selectors, and returns the list of APIExports that match those selectors.
func (c *aPIExportsClient) List(ctx context.Context, opts metav1.ListOptions) (*apisv1alpha1.APIExportList, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootListAction(aPIExportsResource, aPIExportsKind, c.Cluster, opts), &apisv1alpha1.APIExportList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &apisv1alpha1.APIExportList{ListMeta: obj.(*apisv1alpha1.APIExportList).ListMeta}
	for _, item := range obj.(*apisv1alpha1.APIExportList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

func (c *aPIExportsClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.InvokesWatch(kcptesting.NewRootWatchAction(aPIExportsResource, c.Cluster, opts))
}

func (c *aPIExportsClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*apisv1alpha1.APIExport, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootPatchSubresourceAction(aPIExportsResource, c.Cluster, name, pt, data, subresources...), &apisv1alpha1.APIExport{})
	if obj == nil {
		return nil, err
	}
	return obj.(*apisv1alpha1.APIExport), err
}
