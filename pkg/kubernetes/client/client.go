/*
Copyright 2019 The OpenEBS Authors

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

package client

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/utkarshmani1997/jiva-operator/pkg/apis"
	jv "github.com/utkarshmani1997/jiva-operator/pkg/apis/openebs/v1alpha1"
	operr "github.com/utkarshmani1997/jiva-operator/pkg/errors/v1alpha1"
	"github.com/utkarshmani1997/jiva-operator/pkg/volume"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Client is the wrapper over the k8s client that will be used by
// NDM to interface with etcd
type Client struct {
	cfg    *rest.Config
	client client.Client
}

// New creates a new client object using the given config
func New(config *rest.Config) (*Client, error) {
	c := &Client{
		cfg: config,
	}
	err := c.Set()
	if err != nil {
		return c, err
	}
	return c, nil
}

// Set sets the client using the config
func (cl *Client) Set() error {
	c, err := client.New(cl.cfg, client.Options{})
	if err != nil {
		return err
	}
	cl.client = c
	return nil
}

// RegisterAPI registers the API scheme in the client using the manager.
// This function needs to be called only once a client object
func (cl *Client) RegisterAPI() error {
	mgr, err := manager.New(cl.cfg, manager.Options{})
	if err != nil {
		return err
	}

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	return nil
}

func (cl *Client) GetJivaVolume(name string) (*jv.JivaVolume, error) {
	ns := "openebs"
	instance := &jv.JivaVolume{}
	err := cl.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, instance)
	if err != nil {
		logrus.Errorf("Failed to get JivaVolume CR: %v", err)
		if errors.IsNotFound(err) {
			return instance, err
		}
		return instance, err
	}
	return instance, nil
}

func (cl *Client) UpdateJivaVolumeWithMountInfo(cr *jv.JivaVolume, minfo volume.MountInfo) error {
	cr.Spec.MountPath = minfo.Path
	cr.Spec.FSType = minfo.FSType
	err := cl.client.Update(context.TODO(), cr)
	if err != nil {
		logrus.Errorf("Failed to update JivaVolume CR: %v", err)
		return err
	}
	return nil
}

func (cl *Client) CreateJivaVolume(req *csi.CreateVolumeRequest) error {
	name := req.GetName()
	sc := req.GetParameters()["replicaSC"]
	ns := "openebs"
	jiva := new(Jiva).withKindAndAPIVersion("JivaVolume", "openebs.io/v1alpha1").
		withNameAndNamespace(name, ns).
		withPhase(jv.JivaVolumePhasePending).
		withSpec(jv.JivaVolumeSpec{
			PV:       name,
			Capacity: req.GetCapacityRange().GetRequiredBytes(),
			ReplicaSC: func(sc string) string {
				if sc == "" {
					return "openebs-hostpath"
				}
				return sc
			}(sc),
			ReplicaResource: func(req *csi.CreateVolumeRequest) v1.ResourceRequirements {
				return v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(HasResourceParameters(req)("replicaMinCPU")),
						v1.ResourceMemory: resource.MustParse(HasResourceParameters(req)("replicaMinMemory")),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(HasResourceParameters(req)("replicaMaxCPU")),
						v1.ResourceMemory: resource.MustParse(HasResourceParameters(req)("replicaMaxMemory")),
					},
				}
			}(req),

			TargetResource: func(req *csi.CreateVolumeRequest) v1.ResourceRequirements {
				return v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(HasResourceParameters(req)("targetMinCPU")),
						v1.ResourceMemory: resource.MustParse(HasResourceParameters(req)("targetMinMemory")),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(HasResourceParameters(req)("targetMaxCPU")),
						v1.ResourceMemory: resource.MustParse(HasResourceParameters(req)("targetMaxMemory")),
					},
				}
			}(req),
			ReplicationFactor: req.GetParameters()["replicaCount"],
			Iqn:               "iqn.2016-09.com.openebs.jiva:" + name,
		})

	if jiva.errs != nil {
		return operr.Errorf("failed to create JivaVolume CR, err: %v", jiva.errs)
	}

	obj := jiva.instance()
	err := cl.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: ns}, obj)
	if err != nil && errors.IsNotFound(err) {
		logrus.Infof("Creating a new JivaVolume CR, name: %v, namespace: %v", name, ns)
		err = cl.client.Create(context.TODO(), obj)
		if err != nil {
			return operr.Wrapf(err, "failed to create JivaVolume CR, name: %v, namespace: %v", name, ns)
		}
		return nil
	} else if err != nil {
		return operr.Wrapf(err, "failed to get the JivaVolume details: %v", obj.Name)
	}
	return nil
}

func (cl *Client) DeleteJivaVolume(req *csi.DeleteVolumeRequest) error {
	volumeID := req.GetVolumeId()
	obj := &jv.JivaVolume{}
	err := cl.client.Get(context.TODO(), types.NamespacedName{Name: volumeID, Namespace: "openebs"}, obj)
	if err != nil && errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return operr.Wrapf(err, "failed to get the JivaVolume CR details: %v", volumeID)
	}

	err = cl.client.Delete(context.TODO(), obj)
	if err != nil {
		return operr.Wrapf(err, "failed to delete the resource: %v", volumeID)
	}
	return nil
}