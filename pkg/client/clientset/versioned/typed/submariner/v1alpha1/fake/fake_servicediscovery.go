/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeServiceDiscoveries implements ServiceDiscoveryInterface
type FakeServiceDiscoveries struct {
	Fake *FakeSubmarinerV1alpha1
	ns   string
}

var servicediscoveriesResource = schema.GroupVersionResource{Group: "submariner.io", Version: "v1alpha1", Resource: "servicediscoveries"}

var servicediscoveriesKind = schema.GroupVersionKind{Group: "submariner.io", Version: "v1alpha1", Kind: "ServiceDiscovery"}

// Get takes name of the serviceDiscovery, and returns the corresponding serviceDiscovery object, and an error if there is any.
func (c *FakeServiceDiscoveries) Get(name string, options v1.GetOptions) (result *v1alpha1.ServiceDiscovery, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(servicediscoveriesResource, c.ns, name), &v1alpha1.ServiceDiscovery{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceDiscovery), err
}

// List takes label and field selectors, and returns the list of ServiceDiscoveries that match those selectors.
func (c *FakeServiceDiscoveries) List(opts v1.ListOptions) (result *v1alpha1.ServiceDiscoveryList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(servicediscoveriesResource, servicediscoveriesKind, c.ns, opts), &v1alpha1.ServiceDiscoveryList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ServiceDiscoveryList{ListMeta: obj.(*v1alpha1.ServiceDiscoveryList).ListMeta}
	for _, item := range obj.(*v1alpha1.ServiceDiscoveryList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested serviceDiscoveries.
func (c *FakeServiceDiscoveries) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(servicediscoveriesResource, c.ns, opts))

}

// Create takes the representation of a serviceDiscovery and creates it.  Returns the server's representation of the serviceDiscovery, and an error, if there is any.
func (c *FakeServiceDiscoveries) Create(serviceDiscovery *v1alpha1.ServiceDiscovery) (result *v1alpha1.ServiceDiscovery, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(servicediscoveriesResource, c.ns, serviceDiscovery), &v1alpha1.ServiceDiscovery{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceDiscovery), err
}

// Update takes the representation of a serviceDiscovery and updates it. Returns the server's representation of the serviceDiscovery, and an error, if there is any.
func (c *FakeServiceDiscoveries) Update(serviceDiscovery *v1alpha1.ServiceDiscovery) (result *v1alpha1.ServiceDiscovery, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(servicediscoveriesResource, c.ns, serviceDiscovery), &v1alpha1.ServiceDiscovery{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceDiscovery), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeServiceDiscoveries) UpdateStatus(serviceDiscovery *v1alpha1.ServiceDiscovery) (*v1alpha1.ServiceDiscovery, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(servicediscoveriesResource, "status", c.ns, serviceDiscovery), &v1alpha1.ServiceDiscovery{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceDiscovery), err
}

// Delete takes name of the serviceDiscovery and deletes it. Returns an error if one occurs.
func (c *FakeServiceDiscoveries) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(servicediscoveriesResource, c.ns, name), &v1alpha1.ServiceDiscovery{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeServiceDiscoveries) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(servicediscoveriesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ServiceDiscoveryList{})
	return err
}

// Patch applies the patch and returns the patched serviceDiscovery.
func (c *FakeServiceDiscoveries) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ServiceDiscovery, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(servicediscoveriesResource, c.ns, name, pt, data, subresources...), &v1alpha1.ServiceDiscovery{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ServiceDiscovery), err
}