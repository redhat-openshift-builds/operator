/*
Copyright 2020 The Tekton Authors

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

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	scheme "github.com/tektoncd/operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// TektonChainsGetter has a method to return a TektonChainInterface.
// A group's client should implement this interface.
type TektonChainsGetter interface {
	TektonChains() TektonChainInterface
}

// TektonChainInterface has methods to work with TektonChain resources.
type TektonChainInterface interface {
	Create(ctx context.Context, tektonChain *v1alpha1.TektonChain, opts v1.CreateOptions) (*v1alpha1.TektonChain, error)
	Update(ctx context.Context, tektonChain *v1alpha1.TektonChain, opts v1.UpdateOptions) (*v1alpha1.TektonChain, error)
	UpdateStatus(ctx context.Context, tektonChain *v1alpha1.TektonChain, opts v1.UpdateOptions) (*v1alpha1.TektonChain, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.TektonChain, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.TektonChainList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.TektonChain, err error)
	TektonChainExpansion
}

// tektonChains implements TektonChainInterface
type tektonChains struct {
	client rest.Interface
}

// newTektonChains returns a TektonChains
func newTektonChains(c *OperatorV1alpha1Client) *tektonChains {
	return &tektonChains{
		client: c.RESTClient(),
	}
}

// Get takes name of the tektonChain, and returns the corresponding tektonChain object, and an error if there is any.
func (c *tektonChains) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.TektonChain, err error) {
	result = &v1alpha1.TektonChain{}
	err = c.client.Get().
		Resource("tektonchains").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of TektonChains that match those selectors.
func (c *tektonChains) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.TektonChainList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.TektonChainList{}
	err = c.client.Get().
		Resource("tektonchains").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested tektonChains.
func (c *tektonChains) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("tektonchains").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a tektonChain and creates it.  Returns the server's representation of the tektonChain, and an error, if there is any.
func (c *tektonChains) Create(ctx context.Context, tektonChain *v1alpha1.TektonChain, opts v1.CreateOptions) (result *v1alpha1.TektonChain, err error) {
	result = &v1alpha1.TektonChain{}
	err = c.client.Post().
		Resource("tektonchains").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tektonChain).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a tektonChain and updates it. Returns the server's representation of the tektonChain, and an error, if there is any.
func (c *tektonChains) Update(ctx context.Context, tektonChain *v1alpha1.TektonChain, opts v1.UpdateOptions) (result *v1alpha1.TektonChain, err error) {
	result = &v1alpha1.TektonChain{}
	err = c.client.Put().
		Resource("tektonchains").
		Name(tektonChain.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tektonChain).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *tektonChains) UpdateStatus(ctx context.Context, tektonChain *v1alpha1.TektonChain, opts v1.UpdateOptions) (result *v1alpha1.TektonChain, err error) {
	result = &v1alpha1.TektonChain{}
	err = c.client.Put().
		Resource("tektonchains").
		Name(tektonChain.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(tektonChain).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the tektonChain and deletes it. Returns an error if one occurs.
func (c *tektonChains) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("tektonchains").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *tektonChains) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("tektonchains").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched tektonChain.
func (c *tektonChains) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.TektonChain, err error) {
	result = &v1alpha1.TektonChain{}
	err = c.client.Patch(pt).
		Resource("tektonchains").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
