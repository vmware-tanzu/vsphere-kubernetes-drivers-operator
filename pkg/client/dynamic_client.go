/*
Copyright 2021.

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
	"bufio"
	"bytes"
	"encoding/json"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/models"
	"strings"

	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	ApplyYamlFunc = applyYamlSpec
)

func applyYamlSpec(ctx vdocontext.VDOContext, c client.Client, specObj *unstructured.Unstructured, namespace string, action string) error {
	if specObj == nil {
		return nil
	}

	if namespace != "" {
		specObj.SetNamespace(namespace)
	}
	if specObj.GetKind() == "ClusterRoleBinding" && namespace == "" {
		specObj.SetNamespace("kube-system")
	}

	ctx.Logger.V(4).Info("will create object with", "name", specObj.GetName(), "kind", specObj.GetKind())
	switch action {
	case "CREATE":
		err := c.Create(ctx, specObj)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "Error when creating object with %s name, %s kind",
				specObj.GetName(), specObj.GetKind())
		}
	case "DELETE":
		err := c.Delete(ctx, specObj)
		if err != nil {
			return errors.Wrapf(err, "Error when deleting object with %s name, %s kind",
				specObj.GetName(), specObj.GetKind())
		}

	case "UPDATE":
		err := c.Update(ctx, specObj)
		if err != nil {
			return errors.Wrapf(err, "Error when updating object with %s name, %s kind",
				specObj.GetName(), specObj.GetKind())
		}
	}

	return nil
}

// ParseAndProcessK8sObjects executes ApplyYamlFunc for each object in the provided YAML.
// If an error is returned then no further objects are processed.
// The data may be a single YAML document or multidoc YAML.
// When a non-empty namespace is provided then all objects are assigned the
// the namespace prior to any other actions being performed with or to the
// object.
func ParseAndProcessK8sObjects(ctx vdocontext.VDOContext, c client.Client, data []byte, namespace, action string) (appliedSpec bool, err error) {
	var (
		multidocReader = utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	)
	// Iterate over the data until Read returns io.EOF. Every successful
	// read returns a complete YAML document.
	for {
		buf, err := multidocReader.Read()
		if err != nil {
			if err == io.EOF {
				return appliedSpec, nil
			}
			return false, errors.Wrap(err, "failed to read yaml data")
		}
		// Do not use this YAML doc if it is unkind.
		var typeMeta runtime.TypeMeta
		if err := yaml.Unmarshal(buf, &typeMeta); err != nil {
			continue
		}
		if typeMeta.Kind == "" {
			continue
		}

		if typeMeta.Kind == "List" {
			listObject := new(corev1.List)

			if err := yaml.Unmarshal(buf, &listObject); err != nil {
				return false, errors.Wrap(err, "failed to unmarshal yaml data")
			}
			for _, item := range listObject.Items {
				// Define the unstructured object into which the YAML document will be
				// unmarshaled.
				obj := &unstructured.Unstructured{
					Object: map[string]interface{}{},
				}

				if err := yaml.Unmarshal(item.Raw, &obj.Object); err != nil {
					return false, errors.Wrap(err, "failed to unmarshal yaml data")
				}

				if err := ApplyYamlFunc(ctx, c, obj, namespace, action); err != nil {
					if !apierrors.IsAlreadyExists(err) {
						return false, err
					}
				} else {
					appliedSpec = true
				}
			}
		} else {
			// Define the unstructured object into which the YAML document will be
			// unmarshaled.
			obj := &unstructured.Unstructured{
				Object: map[string]interface{}{},
			}

			if err := yaml.Unmarshal(buf, &obj.Object); err != nil {
				return false, errors.Wrap(err, "failed to unmarshal yaml data")
			}

			if err := ApplyYamlFunc(ctx, c, obj, namespace, action); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					return false, err
				}
			} else {
				appliedSpec = true
			}
		}
	}
}

func GenerateYamlFromUrl(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("Recieved response code %d reading from url %s", resp.StatusCode, url)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bodyBytes, nil
}

func GenerateYamlFromFilePath(path string) ([]byte, error) {
	filePath := strings.Replace(path, "file:/", "", 1)
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return fileBytes, nil
}

func ParseMatrixYaml(config string) (models.CompatMatrix, error) {
	var fileBytes []byte
	var err error

	if strings.Contains(config, "file://") {
		fileBytes, err = GenerateYamlFromFilePath(config)
		if err != nil {
			return models.CompatMatrix{}, err
		}
	} else {
		fileBytes, err = GenerateYamlFromUrl(config)
		if err != nil {
			return models.CompatMatrix{}, err
		}
	}
	var matrix models.CompatMatrix

	err = json.Unmarshal(fileBytes, &matrix)
	if err != nil {
		return models.CompatMatrix{}, err
	}

	return matrix, err
}
