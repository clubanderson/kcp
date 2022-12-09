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
	kcpcache "github.com/kcp-dev/apimachinery/pkg/cache"
	"github.com/kcp-dev/logicalcluster/v2"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	tenancyv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/tenancy/v1alpha1"
)

// ClusterWorkspaceTypeClusterLister can list ClusterWorkspaceTypes across all workspaces, or scope down to a ClusterWorkspaceTypeLister for one workspace.
// All objects returned here must be treated as read-only.
type ClusterWorkspaceTypeClusterLister interface {
	// List lists all ClusterWorkspaceTypes in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*tenancyv1alpha1.ClusterWorkspaceType, err error)
	// Cluster returns a lister that can list and get ClusterWorkspaceTypes in one workspace.
	Cluster(cluster logicalcluster.Name) ClusterWorkspaceTypeLister
	ClusterWorkspaceTypeClusterListerExpansion
}

type clusterWorkspaceTypeClusterLister struct {
	indexer cache.Indexer
}

// NewClusterWorkspaceTypeClusterLister returns a new ClusterWorkspaceTypeClusterLister.
// We assume that the indexer:
// - is fed by a cross-workspace LIST+WATCH
// - uses kcpcache.MetaClusterNamespaceKeyFunc as the key function
// - has the kcpcache.ClusterIndex as an index
func NewClusterWorkspaceTypeClusterLister(indexer cache.Indexer) *clusterWorkspaceTypeClusterLister {
	return &clusterWorkspaceTypeClusterLister{indexer: indexer}
}

// List lists all ClusterWorkspaceTypes in the indexer across all workspaces.
func (s *clusterWorkspaceTypeClusterLister) List(selector labels.Selector) (ret []*tenancyv1alpha1.ClusterWorkspaceType, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*tenancyv1alpha1.ClusterWorkspaceType))
	})
	return ret, err
}

// Cluster scopes the lister to one workspace, allowing users to list and get ClusterWorkspaceTypes.
func (s *clusterWorkspaceTypeClusterLister) Cluster(cluster logicalcluster.Name) ClusterWorkspaceTypeLister {
	return &clusterWorkspaceTypeLister{indexer: s.indexer, cluster: cluster}
}

// ClusterWorkspaceTypeLister can list all ClusterWorkspaceTypes, or get one in particular.
// All objects returned here must be treated as read-only.
type ClusterWorkspaceTypeLister interface {
	// List lists all ClusterWorkspaceTypes in the workspace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*tenancyv1alpha1.ClusterWorkspaceType, err error)
	// Get retrieves the ClusterWorkspaceType from the indexer for a given workspace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*tenancyv1alpha1.ClusterWorkspaceType, error)
	ClusterWorkspaceTypeListerExpansion
}

// clusterWorkspaceTypeLister can list all ClusterWorkspaceTypes inside a workspace.
type clusterWorkspaceTypeLister struct {
	indexer cache.Indexer
	cluster logicalcluster.Name
}

// List lists all ClusterWorkspaceTypes in the indexer for a workspace.
func (s *clusterWorkspaceTypeLister) List(selector labels.Selector) (ret []*tenancyv1alpha1.ClusterWorkspaceType, err error) {
	err = kcpcache.ListAllByCluster(s.indexer, s.cluster, selector, func(i interface{}) {
		ret = append(ret, i.(*tenancyv1alpha1.ClusterWorkspaceType))
	})
	return ret, err
}

// Get retrieves the ClusterWorkspaceType from the indexer for a given workspace and name.
func (s *clusterWorkspaceTypeLister) Get(name string) (*tenancyv1alpha1.ClusterWorkspaceType, error) {
	key := kcpcache.ToClusterAwareKey(s.cluster.String(), "", name)
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(tenancyv1alpha1.Resource("ClusterWorkspaceType"), name)
	}
	return obj.(*tenancyv1alpha1.ClusterWorkspaceType), nil
}

// NewClusterWorkspaceTypeLister returns a new ClusterWorkspaceTypeLister.
// We assume that the indexer:
// - is fed by a workspace-scoped LIST+WATCH
// - uses cache.MetaNamespaceKeyFunc as the key function
func NewClusterWorkspaceTypeLister(indexer cache.Indexer) *clusterWorkspaceTypeScopedLister {
	return &clusterWorkspaceTypeScopedLister{indexer: indexer}
}

// clusterWorkspaceTypeScopedLister can list all ClusterWorkspaceTypes inside a workspace.
type clusterWorkspaceTypeScopedLister struct {
	indexer cache.Indexer
}

// List lists all ClusterWorkspaceTypes in the indexer for a workspace.
func (s *clusterWorkspaceTypeScopedLister) List(selector labels.Selector) (ret []*tenancyv1alpha1.ClusterWorkspaceType, err error) {
	err = cache.ListAll(s.indexer, selector, func(i interface{}) {
		ret = append(ret, i.(*tenancyv1alpha1.ClusterWorkspaceType))
	})
	return ret, err
}

// Get retrieves the ClusterWorkspaceType from the indexer for a given workspace and name.
func (s *clusterWorkspaceTypeScopedLister) Get(name string) (*tenancyv1alpha1.ClusterWorkspaceType, error) {
	key := name
	obj, exists, err := s.indexer.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(tenancyv1alpha1.Resource("ClusterWorkspaceType"), name)
	}
	return obj.(*tenancyv1alpha1.ClusterWorkspaceType), nil
}
