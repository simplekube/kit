package k8s

import (
	"fmt"
	"os"
	"testing"

	"github.com/simplekube/kit/pkg/pointer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var klient client.Client
var rscheme *runtime.Scheme

// runMain helps to return exit code along with use of defer statements
func runMain(m *testing.M) int {
	var err error
	var cfg *rest.Config

	testEnv := &envtest.Environment{
		UseExistingCluster: pointer.Bool(false), // use local binaries i.e. etcd & apiserver
		// AttachControlPlaneOutput: true, // this is too verbose
	}
	cfg, err = testEnv.Start()
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer func() {
		sErr := testEnv.Stop()
		if sErr != nil {
			fmt.Println(sErr)
		}
	}()

	// initialise the Kubernetes client needed to invoke K8s APIs
	// Note: This is a global variable
	klient, err = client.New(cfg, client.Options{})
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// initialise Kubernetes scheme that has all native schemas registered
	// Note: This is a global variable
	rscheme = scheme.Scheme

	err = RegisterBaseRunOptions(&RunOptions{
		Client: klient,
		Scheme: rscheme,
	})
	if err != nil {
		fmt.Println(err)
		return 1
	}

	// Trigger to execute all functions starting with Test*
	// found in ./*_test.go files
	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}
