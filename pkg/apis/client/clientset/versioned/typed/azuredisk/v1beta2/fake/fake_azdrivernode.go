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
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1beta2 "sigs.k8s.io/azuredisk-csi-driver/pkg/apis/azuredisk/v1beta2"
)

// FakeAzDriverNodes implements AzDriverNodeInterface
type FakeAzDriverNodes struct {
	Fake *FakeDiskV1beta2
	ns   string
}

var azdrivernodesResource = schema.GroupVersionResource{Group: "disk.csi.azure.com", Version: "v1beta2", Resource: "azdrivernodes"}

var azdrivernodesKind = schema.GroupVersionKind{Group: "disk.csi.azure.com", Version: "v1beta2", Kind: "AzDriverNode"}

// Get takes name of the azDriverNode, and returns the corresponding azDriverNode object, and an error if there is any.
func (c *FakeAzDriverNodes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1beta2.AzDriverNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(azdrivernodesResource, c.ns, name), &v1beta2.AzDriverNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.AzDriverNode), err
}

// List takes label and field selectors, and returns the list of AzDriverNodes that match those selectors.
func (c *FakeAzDriverNodes) List(ctx context.Context, opts v1.ListOptions) (result *v1beta2.AzDriverNodeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(azdrivernodesResource, azdrivernodesKind, c.ns, opts), &v1beta2.AzDriverNodeList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta2.AzDriverNodeList{ListMeta: obj.(*v1beta2.AzDriverNodeList).ListMeta}
	for _, item := range obj.(*v1beta2.AzDriverNodeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested azDriverNodes.
func (c *FakeAzDriverNodes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(azdrivernodesResource, c.ns, opts))

}

// Create takes the representation of a azDriverNode and creates it.  Returns the server's representation of the azDriverNode, and an error, if there is any.
func (c *FakeAzDriverNodes) Create(ctx context.Context, azDriverNode *v1beta2.AzDriverNode, opts v1.CreateOptions) (result *v1beta2.AzDriverNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(azdrivernodesResource, c.ns, azDriverNode), &v1beta2.AzDriverNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.AzDriverNode), err
}

// Update takes the representation of a azDriverNode and updates it. Returns the server's representation of the azDriverNode, and an error, if there is any.
func (c *FakeAzDriverNodes) Update(ctx context.Context, azDriverNode *v1beta2.AzDriverNode, opts v1.UpdateOptions) (result *v1beta2.AzDriverNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(azdrivernodesResource, c.ns, azDriverNode), &v1beta2.AzDriverNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.AzDriverNode), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAzDriverNodes) UpdateStatus(ctx context.Context, azDriverNode *v1beta2.AzDriverNode, opts v1.UpdateOptions) (*v1beta2.AzDriverNode, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(azdrivernodesResource, "status", c.ns, azDriverNode), &v1beta2.AzDriverNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.AzDriverNode), err
}

// Delete takes name of the azDriverNode and deletes it. Returns an error if one occurs.
func (c *FakeAzDriverNodes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(azdrivernodesResource, c.ns, name, opts), &v1beta2.AzDriverNode{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAzDriverNodes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(azdrivernodesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1beta2.AzDriverNodeList{})
	return err
}

// Patch applies the patch and returns the patched azDriverNode.
func (c *FakeAzDriverNodes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1beta2.AzDriverNode, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(azdrivernodesResource, c.ns, name, pt, data, subresources...), &v1beta2.AzDriverNode{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta2.AzDriverNode), err
}
