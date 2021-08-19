/*
Copyright 2021 The Openshift Evangelists

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

package v1

import (
	v1 "github.com/hmsayem/server-controller/pkg/apis/hmsayem.com/v1"
	"github.com/hmsayem/server-controller/pkg/client/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type HmsayemV1Interface interface {
	RESTClient() rest.Interface
	ServersGetter
}

// HmsayemV1Client is used to interact with features provided by the hmsayem.com group.
type HmsayemV1Client struct {
	restClient rest.Interface
}

func (c *HmsayemV1Client) Servers(namespace string) ServerInterface {
	return newServers(c, namespace)
}

// NewForConfig creates a new HmsayemV1Client for the given config.
func NewForConfig(c *rest.Config) (*HmsayemV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &HmsayemV1Client{client}, nil
}

// NewForConfigOrDie creates a new HmsayemV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *HmsayemV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new HmsayemV1Client for the given RESTClient.
func New(c rest.Interface) *HmsayemV1Client {
	return &HmsayemV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *HmsayemV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}