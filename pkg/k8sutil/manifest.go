package k8sutil

import (
	"bufio"
	"os"
	"path"
	"sort"

	"github.com/hashicorp/go-multierror"

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

	var errs = make([]error, 0, len(manifests))
	for _, manifest := range manifests {
		ms, err := os.Open(manifest)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "yaml %q", manifest))
			continue
		}

		objs, err := ReadKubernetesObjects(bufio.NewReader(ms))
		ms.Close()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		objects = MaybeAppendUnstructuredList(objects, objs)
	}
	return objects, (&multierror.Error{Errors: errs}).ErrorOrNil()
}

func ScanForYMLsFromPaths(paths []string) ([]string, error) {
	var manifests []string

	var errs = make([]error, 0, len(paths))
	for _, path := range paths {
		fi, err := os.Stat(path)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "path %q", path))
			continue
		}

		switch mode := fi.Mode(); {
		case mode.IsDir():
			m, err := ScanForYMLsFromDir(path)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "path %q", path))
				continue
			}
			manifests = append(manifests, m...)
		case mode.IsRegular():
			if IsExtensionYML(fi.Name()) {
				manifests = append(manifests, path)
			}
		}
	}

	return manifests, (&multierror.Error{Errors: errs}).ErrorOrNil()
}

// ScanForYMLsFromDir scans for files present in the provided directory &
// its sub-directories if any
func ScanForYMLsFromDir(dir string) ([]string, error) {
	var manifests []string
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "dir %q", dir)
	}

	var errs = make([]error, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			m, err := ScanForYMLsFromDir(path.Join(dir, file.Name()))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			manifests = append(manifests, m...)
		}
		if IsExtensionYML(file.Name()) {
			manifests = append(manifests, path.Join(dir, file.Name()))
		}
	}
	return manifests, (&multierror.Error{Errors: errs}).ErrorOrNil()
}

// IsExtensionYML returns true if provided file has yaml extension
func IsExtensionYML(f string) bool {
	ext := path.Ext(f)
	return ext == ".yaml" || ext == ".yml"
}
