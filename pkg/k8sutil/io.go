package k8sutil

import (
	"io"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

// ReadKubernetesObjects decodes the YAML or JSON documents from the provided
// reader into unstructured Kubernetes API objects
func ReadKubernetesObjects(r io.Reader) ([]*unstructured.Unstructured, error) {
	reader := yamlutil.NewYAMLOrJSONDecoder(r, 2048)
	objects := make([]*unstructured.Unstructured, 0)

	for {
		obj := &unstructured.Unstructured{}
		err := reader.Decode(obj)
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return objects, errors.Wrap(err, "decode to unstructured")
		}

		if obj.IsList() {
			_ = obj.EachListItem(func(item runtime.Object) (_ error) {
				obj, _ := item.(*unstructured.Unstructured)
				if IsKubernetesObject(obj) && !IsKustomizeObject(obj) {
					objects = append(objects, obj)
				}
				return nil
			})
			continue
		}

		if IsKubernetesObject(obj) && !IsKustomizeObject(obj) {
			objects = append(objects, obj)
		}
	}

	return objects, nil
}
