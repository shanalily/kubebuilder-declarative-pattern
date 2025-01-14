package restmapper

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// NewControllerRESTMapper is the constructor for a ControllerRESTMapper
func NewControllerRESTMapper(cfg *rest.Config) (meta.RESTMapper, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &ControllerRESTMapper{
		uncached: discoveryClient,
		cache:    newCache(),
	}, nil
}

// ControllerRESTMapper is a meta.RESTMapper that is optimized for controllers.
// It caches results in memory, and minimizes discovery because we don't need shortnames etc in controllers.
// Controllers primarily need to map from GVK -> GVR.
type ControllerRESTMapper struct {
	uncached discovery.DiscoveryInterface
	cache    *cache
}

var _ meta.RESTMapper = &ControllerRESTMapper{}

// KindFor takes a partial resource and returns the single match.  Returns an error if there are multiple matches
func (m *ControllerRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, fmt.Errorf("ControllerRESTMaper does not support KindFor operation")
}

// KindsFor takes a partial resource and returns the list of potential kinds in priority order
func (m *ControllerRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	return nil, fmt.Errorf("ControllerRESTMaper does not support KindsFor operation")
}

// ResourceFor takes a partial resource and returns the single match.  Returns an error if there are multiple matches
func (m *ControllerRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return schema.GroupVersionResource{}, fmt.Errorf("ControllerRESTMaper does not support ResourceFor operation")
}

// ResourcesFor takes a partial resource and returns the list of potential resource in priority order
func (m *ControllerRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return nil, fmt.Errorf("ControllerRESTMaper does not support ResourcesFor operation")
}

// RESTMapping identifies a preferred resource mapping for the provided group kind.
func (m *ControllerRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	ctx := context.TODO()

	for _, version := range versions {
		gv := schema.GroupVersion{Group: gk.Group, Version: version}
		mapping, err := m.cache.findRESTMapping(ctx, m.uncached, gv, gk.Kind)
		if err != nil {
			return nil, err
		}
		if mapping != nil {
			return mapping, nil
		}
	}

	return nil, &meta.NoKindMatchError{GroupKind: gk, SearchedVersions: versions}
}

// RESTMappings returns all resource mappings for the provided group kind if no
// version search is provided. Otherwise identifies a preferred resource mapping for
// the provided version(s).
func (m *ControllerRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	return nil, fmt.Errorf("ControllerRESTMaper does not support RESTMappings operation")
}

func (m *ControllerRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return "", fmt.Errorf("ControllerRESTMaper does not support ResourceSingularizer operation")
}
