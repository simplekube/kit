package checks

import "github.com/simplekube/kit/pkg/k8s"

type RunOption = k8s.RunOption

type Job = k8s.Job
type Tasks = k8s.Tasks

type Task = k8s.Task
type Custom = k8s.CustomTask
type PodExec = k8s.PodExecTask
type AssertEquals = k8s.AssertIsEqualsTask
type CreateThenAssertEquals = k8s.CreateThenAssertIsEqualsTask
type UpsertThenAssertEquals = k8s.UpsertThenAssertIsEqualsTask
type AssertPodListCount = k8s.AssertPodListCountTask
type EventualTask = k8s.EventualTask
type ListingTask = k8s.ListingTask
type DeletingTask = k8s.DeletingTask
type FinalizersRemovalTask = k8s.FinalizersRemovalTask

var (
	Get           = k8s.ActionTypeGet
	Create        = k8s.ActionTypeCreate
	CreateOrMerge = k8s.ActionTypeCreateOrMerge
)

var (
	Equals = k8s.AssertTypeIsEquals
)
