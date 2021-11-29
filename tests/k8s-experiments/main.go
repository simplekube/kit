package main

import (
	"context"
	"fmt"
	"os"

	"k8s.tests/checks"
	"k8s.tests/setup"

	"github.com/simplekube/kit/pkg/envutil"
	"github.com/simplekube/kit/pkg/k8s"
)

// have a separate function so we can return an exit code w/o skipping defers
func run() int {
	fmt.Println(os.Args)

	// set up test environment
	env := setup.New("e2e-testing")
	ctx := context.Background()
	options := &k8s.RunOptions{}

	err := env.Setup(ctx)
	// we should defer the teardown first & then handle the error if any
	defer func() {
		terr := env.Teardown(ctx)
		if terr != nil {
			fmt.Printf("%s\n", terr)
		}
	}()
	if err != nil {
		fmt.Printf("%s\n", err)
		return 1
	}

	// optionally set above setup as e2e suite's namespace
	// via an environment variable
	envutil.MayBeSet(checks.EnvKeyE2eSuiteNamespace, env.GetNamespace())

	// run the check(s)
	checkFns := []func(ctx2 context.Context, opts ...k8s.RunOption) error{
		checks.IsK8sDeploymentIdempotent,
		checks.DoesK8sDeploymentPropagate,
		checks.DoesK8sDNSWork,
		checks.DoesHPAWork,
	}
	for _, fn := range checkFns {
		err := fn(ctx, options)
		if err != nil {
			fmt.Printf("%s\n", err)
			return 1
		}
	}

	return 0
}

func main() {
	// TODO (@amit.das)
	//  handle termination signals & use the handler to invoke Teardown
	os.Exit(run())
}
