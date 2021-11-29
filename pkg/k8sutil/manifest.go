package k8sutil

import (
	"bufio"
	"os"
	"path"
	"sort"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func BuildSortableObjectsFromYMLs(filePaths []string) ([]*unstructured.Unstructured, error) {
	objs, err := BuildObjectsFromYMLs(filePaths)
	if err != nil {
		return nil, err
	}
	sort.Sort(SortableUnstructureds(objs))
	return objs, nil
}

func BuildObjectsFromYMLs(filePaths []string) ([]*unstructured.Unstructured, error) {
	if len(filePaths) == 0 {
		return nil, errors.New("no file paths provided")
	}

	var objects = make([]*unstructured.Unstructured, 0)
	manifests, err := ScanForYMLsFromPaths(filePaths)
	if err != nil {
		return nil, err
	}
	for _, manifest := range manifests {
		ms, err := os.Open(manifest)
		if err != nil {
			return nil, err
		}

		objs, err := ReadKubernetesObjects(bufio.NewReader(ms))
		ms.Close()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read %q", manifest)
		}
		objects = append(objects, objs...)
	}
	return objects, nil
}

func ScanForYMLsFromPaths(paths []string) ([]string, error) {
	var manifests []string

	for _, in := range paths {
		fi, err := os.Stat(in)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get file info")
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			m, err := ScanForYMLsFromDir(in)
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, m...)
		case mode.IsRegular():
			if IsExtYML(fi.Name()) {
				manifests = append(manifests, in)
			}
		}
	}

	return manifests, nil
}

// ScanForYMLsFromDir scans for files present in the provided directory &
// its sub-directories if any
func ScanForYMLsFromDir(dir string) ([]string, error) {
	var manifests []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read %q", dir)
	}

	for _, file := range files {
		if file.IsDir() {
			m, err := ScanForYMLsFromDir(path.Join(dir, file.Name()))
			if err != nil {
				return nil, err
			}
			manifests = append(manifests, m...)
		}
		if IsExtYML(file.Name()) {
			manifests = append(manifests, path.Join(dir, file.Name()))
		}
	}
	return manifests, err
}

// IsExtYML returns true if provided file has yaml extension
func IsExtYML(f string) bool {
	ext := path.Ext(f)
	return ext == ".yaml" || ext == ".yml"
}
