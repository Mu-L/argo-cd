package controller

import (
	"strconv"
	"testing"

	"github.com/argoproj/gitops-engine/pkg/sync"
	synccommon "github.com/argoproj/gitops-engine/pkg/sync/common"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/argoproj/argo-cd/v3/common"
	"github.com/argoproj/argo-cd/v3/controller/testdata"
	"github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/argoproj/argo-cd/v3/reposerver/apiclient"
	"github.com/argoproj/argo-cd/v3/test"
	"github.com/argoproj/argo-cd/v3/util/argo/diff"
	"github.com/argoproj/argo-cd/v3/util/argo/normalizers"
)

func TestPersistRevisionHistory(t *testing.T) {
	app := newFakeApp()
	app.Status.OperationState = nil
	app.Status.History = nil

	defaultProject := &v1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.FakeArgoCDNamespace,
			Name:      "default",
		},
	}
	data := fakeData{
		apps: []runtime.Object{app, defaultProject},
		manifestResponse: &apiclient.ManifestResponse{
			Manifests: []string{},
			Namespace: test.FakeDestNamespace,
			Server:    test.FakeClusterURL,
			Revision:  "abc123",
		},
		managedLiveObjs: make(map[kube.ResourceKey]*unstructured.Unstructured),
	}
	ctrl := newFakeController(&data, nil)

	// Sync with source unspecified
	opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
		Sync: &v1alpha1.SyncOperation{},
	}}
	ctrl.appStateManager.SyncAppState(app, defaultProject, opState)
	// Ensure we record spec.source into sync result
	assert.Equal(t, app.Spec.GetSource(), opState.SyncResult.Source)

	updatedApp, err := ctrl.applicationClientset.ArgoprojV1alpha1().Applications(app.Namespace).Get(t.Context(), app.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, updatedApp.Status.History, 1)
	assert.Equal(t, app.Spec.GetSource(), updatedApp.Status.History[0].Source)
	assert.Equal(t, "abc123", updatedApp.Status.History[0].Revision)
}

func TestPersistManagedNamespaceMetadataState(t *testing.T) {
	app := newFakeApp()
	app.Status.OperationState = nil
	app.Status.History = nil
	app.Spec.SyncPolicy.ManagedNamespaceMetadata = &v1alpha1.ManagedNamespaceMetadata{
		Labels: map[string]string{
			"foo": "bar",
		},
		Annotations: map[string]string{
			"foo": "bar",
		},
	}

	defaultProject := &v1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.FakeArgoCDNamespace,
			Name:      "default",
		},
	}
	data := fakeData{
		apps: []runtime.Object{app, defaultProject},
		manifestResponse: &apiclient.ManifestResponse{
			Manifests: []string{},
			Namespace: test.FakeDestNamespace,
			Server:    test.FakeClusterURL,
			Revision:  "abc123",
		},
		managedLiveObjs: make(map[kube.ResourceKey]*unstructured.Unstructured),
	}
	ctrl := newFakeController(&data, nil)

	// Sync with source unspecified
	opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
		Sync: &v1alpha1.SyncOperation{},
	}}
	ctrl.appStateManager.SyncAppState(app, defaultProject, opState)
	// Ensure we record spec.syncPolicy.managedNamespaceMetadata into sync result
	assert.Equal(t, app.Spec.SyncPolicy.ManagedNamespaceMetadata, opState.SyncResult.ManagedNamespaceMetadata)
}

func TestPersistRevisionHistoryRollback(t *testing.T) {
	app := newFakeApp()
	app.Status.OperationState = nil
	app.Status.History = nil
	defaultProject := &v1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.FakeArgoCDNamespace,
			Name:      "default",
		},
	}
	data := fakeData{
		apps: []runtime.Object{app, defaultProject},
		manifestResponse: &apiclient.ManifestResponse{
			Manifests: []string{},
			Namespace: test.FakeDestNamespace,
			Server:    test.FakeClusterURL,
			Revision:  "abc123",
		},
		managedLiveObjs: make(map[kube.ResourceKey]*unstructured.Unstructured),
	}
	ctrl := newFakeController(&data, nil)

	// Sync with source specified
	source := v1alpha1.ApplicationSource{
		Helm: &v1alpha1.ApplicationSourceHelm{
			Parameters: []v1alpha1.HelmParameter{
				{
					Name:  "test",
					Value: "123",
				},
			},
		},
	}
	opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
		Sync: &v1alpha1.SyncOperation{
			Source: &source,
		},
	}}
	ctrl.appStateManager.SyncAppState(app, defaultProject, opState)
	// Ensure we record opState's source into sync result
	assert.Equal(t, source, opState.SyncResult.Source)

	updatedApp, err := ctrl.applicationClientset.ArgoprojV1alpha1().Applications(app.Namespace).Get(t.Context(), app.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Len(t, updatedApp.Status.History, 1)
	assert.Equal(t, source, updatedApp.Status.History[0].Source)
	assert.Equal(t, "abc123", updatedApp.Status.History[0].Revision)
}

func TestSyncComparisonError(t *testing.T) {
	app := newFakeApp()
	app.Status.OperationState = nil
	app.Status.History = nil

	defaultProject := &v1alpha1.AppProject{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: test.FakeArgoCDNamespace,
			Name:      "default",
		},
		Spec: v1alpha1.AppProjectSpec{
			SignatureKeys: []v1alpha1.SignatureKey{{KeyID: "test"}},
		},
	}
	data := fakeData{
		apps: []runtime.Object{app, defaultProject},
		manifestResponse: &apiclient.ManifestResponse{
			Manifests:    []string{},
			Namespace:    test.FakeDestNamespace,
			Server:       test.FakeClusterURL,
			Revision:     "abc123",
			VerifyResult: "something went wrong",
		},
		managedLiveObjs: make(map[kube.ResourceKey]*unstructured.Unstructured),
	}
	ctrl := newFakeController(&data, nil)

	// Sync with source unspecified
	opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
		Sync: &v1alpha1.SyncOperation{},
	}}
	t.Setenv("ARGOCD_GPG_ENABLED", "true")
	ctrl.appStateManager.SyncAppState(app, defaultProject, opState)

	conditions := app.Status.GetConditions(map[v1alpha1.ApplicationConditionType]bool{v1alpha1.ApplicationConditionComparisonError: true})
	assert.NotEmpty(t, conditions)
	assert.Equal(t, "abc123", opState.SyncResult.Revision)
}

func TestAppStateManager_SyncAppState(t *testing.T) {
	t.Parallel()

	type fixture struct {
		application *v1alpha1.Application
		project     *v1alpha1.AppProject
		controller  *ApplicationController
	}

	setup := func(liveObjects map[kube.ResourceKey]*unstructured.Unstructured) *fixture {
		app := newFakeApp()
		app.Status.OperationState = nil
		app.Status.History = nil

		if liveObjects == nil {
			liveObjects = make(map[kube.ResourceKey]*unstructured.Unstructured)
		}

		project := &v1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: test.FakeArgoCDNamespace,
				Name:      "default",
			},
			Spec: v1alpha1.AppProjectSpec{
				SignatureKeys: []v1alpha1.SignatureKey{{KeyID: "test"}},
				Destinations: []v1alpha1.ApplicationDestination{
					{
						Namespace: "*",
						Server:    "*",
					},
				},
			},
		}
		data := fakeData{
			apps: []runtime.Object{app, project},
			manifestResponse: &apiclient.ManifestResponse{
				Manifests: []string{},
				Namespace: test.FakeDestNamespace,
				Server:    test.FakeClusterURL,
				Revision:  "abc123",
			},
			managedLiveObjs: liveObjects,
		}
		ctrl := newFakeController(&data, nil)

		return &fixture{
			application: app,
			project:     project,
			controller:  ctrl,
		}
	}

	t.Run("will fail the sync if finds shared resources", func(t *testing.T) {
		// given
		t.Parallel()

		sharedObject := kube.MustToUnstructured(&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap1",
				Namespace: "default",
				Annotations: map[string]string{
					common.AnnotationKeyAppInstance: "guestbook:/ConfigMap:default/configmap1",
				},
			},
		})
		liveObjects := make(map[kube.ResourceKey]*unstructured.Unstructured)
		liveObjects[kube.GetResourceKey(sharedObject)] = sharedObject
		f := setup(liveObjects)

		// Sync with source unspecified
		opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
			Sync: &v1alpha1.SyncOperation{
				Source:      &v1alpha1.ApplicationSource{},
				SyncOptions: []string{"FailOnSharedResource=true"},
			},
		}}

		// when
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then
		assert.Equal(t, synccommon.OperationFailed, opState.Phase)
		assert.Contains(t, opState.Message, "ConfigMap/configmap1 is part of applications fake-argocd-ns/my-app and guestbook")
	})
}

func TestSyncWindowDeniesSync(t *testing.T) {
	t.Parallel()

	type fixture struct {
		application *v1alpha1.Application
		project     *v1alpha1.AppProject
		controller  *ApplicationController
	}

	setup := func() *fixture {
		app := newFakeApp()
		app.Status.OperationState = nil
		app.Status.History = nil

		project := &v1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: test.FakeArgoCDNamespace,
				Name:      "default",
			},
			Spec: v1alpha1.AppProjectSpec{
				SyncWindows: v1alpha1.SyncWindows{{
					Kind:         "deny",
					Schedule:     "0 0 * * *",
					Duration:     "24h",
					Clusters:     []string{"*"},
					Namespaces:   []string{"*"},
					Applications: []string{"*"},
				}},
			},
		}
		data := fakeData{
			apps: []runtime.Object{app, project},
			manifestResponse: &apiclient.ManifestResponse{
				Manifests: []string{},
				Namespace: test.FakeDestNamespace,
				Server:    test.FakeClusterURL,
				Revision:  "abc123",
			},
			managedLiveObjs: make(map[kube.ResourceKey]*unstructured.Unstructured),
		}
		ctrl := newFakeController(&data, nil)

		return &fixture{
			application: app,
			project:     project,
			controller:  ctrl,
		}
	}

	t.Run("will keep the sync progressing if a sync window prevents the sync", func(t *testing.T) {
		// given a project with an active deny sync window and an operation in progress
		t.Parallel()
		f := setup()
		opMessage := "Sync operation blocked by sync window"

		opState := &v1alpha1.OperationState{
			Operation: v1alpha1.Operation{
				Sync: &v1alpha1.SyncOperation{
					Source: &v1alpha1.ApplicationSource{},
				},
			},
			Phase: synccommon.OperationRunning,
		}
		// when
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then
		assert.Equal(t, synccommon.OperationRunning, opState.Phase)
		assert.Contains(t, opState.Message, opMessage)
	})
}

func TestNormalizeTargetResources(t *testing.T) {
	type fixture struct {
		comparisonResult *comparisonResult
	}
	setup := func(t *testing.T, ignores []v1alpha1.ResourceIgnoreDifferences) *fixture {
		t.Helper()
		dc, err := diff.NewDiffConfigBuilder().
			WithDiffSettings(ignores, nil, true, normalizers.IgnoreNormalizerOpts{}).
			WithNoCache().
			Build()
		require.NoError(t, err)
		live := test.YamlToUnstructured(testdata.LiveDeploymentYaml)
		target := test.YamlToUnstructured(testdata.TargetDeploymentYaml)
		return &fixture{
			&comparisonResult{
				reconciliationResult: sync.ReconciliationResult{
					Live:   []*unstructured.Unstructured{live},
					Target: []*unstructured.Unstructured{target},
				},
				diffConfig: dc,
			},
		}
	}
	t.Run("will modify target resource adding live state in fields it should ignore", func(t *testing.T) {
		// given
		ignore := v1alpha1.ResourceIgnoreDifferences{
			Group:                 "*",
			Kind:                  "*",
			ManagedFieldsManagers: []string{"janitor"},
		}
		ignores := []v1alpha1.ResourceIgnoreDifferences{ignore}
		f := setup(t, ignores)

		// when
		targets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, targets, 1)
		iksmVersion := targets[0].GetAnnotations()["iksm-version"]
		assert.Equal(t, "2.0", iksmVersion)
	})
	t.Run("will not modify target resource if ignore difference is not configured", func(t *testing.T) {
		// given
		f := setup(t, []v1alpha1.ResourceIgnoreDifferences{})

		// when
		targets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, targets, 1)
		iksmVersion := targets[0].GetAnnotations()["iksm-version"]
		assert.Equal(t, "1.0", iksmVersion)
	})
	t.Run("will remove fields from target if not present in live", func(t *testing.T) {
		ignore := v1alpha1.ResourceIgnoreDifferences{
			Group:        "apps",
			Kind:         "Deployment",
			JSONPointers: []string{"/metadata/annotations/iksm-version"},
		}
		ignores := []v1alpha1.ResourceIgnoreDifferences{ignore}
		f := setup(t, ignores)
		live := f.comparisonResult.reconciliationResult.Live[0]
		unstructured.RemoveNestedField(live.Object, "metadata", "annotations", "iksm-version")

		// when
		targets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, targets, 1)
		_, ok := targets[0].GetAnnotations()["iksm-version"]
		assert.False(t, ok)
	})
	t.Run("will correctly normalize with multiple ignore configurations", func(t *testing.T) {
		// given
		ignores := []v1alpha1.ResourceIgnoreDifferences{
			{
				Group:        "apps",
				Kind:         "Deployment",
				JSONPointers: []string{"/spec/replicas"},
			},
			{
				Group:                 "*",
				Kind:                  "*",
				ManagedFieldsManagers: []string{"janitor"},
			},
		}
		f := setup(t, ignores)

		// when
		targets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, targets, 1)
		normalized := targets[0]
		iksmVersion, ok := normalized.GetAnnotations()["iksm-version"]
		require.True(t, ok)
		assert.Equal(t, "2.0", iksmVersion)
		replicas, ok, err := unstructured.NestedInt64(normalized.Object, "spec", "replicas")
		require.NoError(t, err)
		require.True(t, ok)
		assert.Equal(t, int64(4), replicas)
	})
	t.Run("will keep new array entries not found in live state if not ignored", func(t *testing.T) {
		t.Skip("limitation in the current implementation")
		// given
		ignores := []v1alpha1.ResourceIgnoreDifferences{
			{
				Group:             "apps",
				Kind:              "Deployment",
				JQPathExpressions: []string{".spec.template.spec.containers[] | select(.name == \"guestbook-ui\")"},
			},
		}
		f := setup(t, ignores)
		target := test.YamlToUnstructured(testdata.TargetDeploymentNewEntries)
		f.comparisonResult.reconciliationResult.Target = []*unstructured.Unstructured{target}

		// when
		targets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, targets, 1)
		containers, ok, err := unstructured.NestedSlice(targets[0].Object, "spec", "template", "spec", "containers")
		require.NoError(t, err)
		require.True(t, ok)
		assert.Len(t, containers, 2)
	})
}

func TestNormalizeTargetResourcesWithList(t *testing.T) {
	type fixture struct {
		comparisonResult *comparisonResult
	}
	setupHTTPProxy := func(t *testing.T, ignores []v1alpha1.ResourceIgnoreDifferences) *fixture {
		t.Helper()
		dc, err := diff.NewDiffConfigBuilder().
			WithDiffSettings(ignores, nil, true, normalizers.IgnoreNormalizerOpts{}).
			WithNoCache().
			Build()
		require.NoError(t, err)
		live := test.YamlToUnstructured(testdata.LiveHTTPProxy)
		target := test.YamlToUnstructured(testdata.TargetHTTPProxy)
		return &fixture{
			&comparisonResult{
				reconciliationResult: sync.ReconciliationResult{
					Live:   []*unstructured.Unstructured{live},
					Target: []*unstructured.Unstructured{target},
				},
				diffConfig: dc,
			},
		}
	}

	t.Run("will properly ignore nested fields within arrays", func(t *testing.T) {
		// given
		ignores := []v1alpha1.ResourceIgnoreDifferences{
			{
				Group:             "projectcontour.io",
				Kind:              "HTTPProxy",
				JQPathExpressions: []string{".spec.routes[]"},
				// JSONPointers: []string{"/spec/routes"},
			},
		}
		f := setupHTTPProxy(t, ignores)
		target := test.YamlToUnstructured(testdata.TargetHTTPProxy)
		f.comparisonResult.reconciliationResult.Target = []*unstructured.Unstructured{target}

		// when
		patchedTargets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, f.comparisonResult.reconciliationResult.Live, 1)
		require.Len(t, f.comparisonResult.reconciliationResult.Target, 1)
		require.Len(t, patchedTargets, 1)

		// live should have 1 entry
		require.Len(t, dig(f.comparisonResult.reconciliationResult.Live[0].Object, "spec", "routes", 0, "rateLimitPolicy", "global", "descriptors"), 1)
		// assert some arbitrary field to show `entries[0]` is not an empty object
		require.Equal(t, "sample-header", dig(f.comparisonResult.reconciliationResult.Live[0].Object, "spec", "routes", 0, "rateLimitPolicy", "global", "descriptors", 0, "entries", 0, "requestHeader", "headerName"))

		// target has 2 entries
		require.Len(t, dig(f.comparisonResult.reconciliationResult.Target[0].Object, "spec", "routes", 0, "rateLimitPolicy", "global", "descriptors", 0, "entries"), 2)
		// assert some arbitrary field to show `entries[0]` is not an empty object
		require.Equal(t, "sample-header", dig(f.comparisonResult.reconciliationResult.Target[0].Object, "spec", "routes", 0, "rateLimitPolicy", "global", "descriptors", 0, "entries", 0, "requestHeaderValueMatch", "headers", 0, "name"))

		// It should be *1* entries in the array
		require.Len(t, dig(patchedTargets[0].Object, "spec", "routes", 0, "rateLimitPolicy", "global", "descriptors"), 1)
		// and it should NOT equal an empty object
		require.Len(t, dig(patchedTargets[0].Object, "spec", "routes", 0, "rateLimitPolicy", "global", "descriptors", 0, "entries", 0), 1)
	})
	t.Run("will correctly set array entries if new entries have been added", func(t *testing.T) {
		// given
		ignores := []v1alpha1.ResourceIgnoreDifferences{
			{
				Group:             "apps",
				Kind:              "Deployment",
				JQPathExpressions: []string{".spec.template.spec.containers[].env[] | select(.name == \"SOME_ENV_VAR\")"},
			},
		}
		f := setupHTTPProxy(t, ignores)
		live := test.YamlToUnstructured(testdata.LiveDeploymentEnvVarsYaml)
		target := test.YamlToUnstructured(testdata.TargetDeploymentEnvVarsYaml)
		f.comparisonResult.reconciliationResult.Live = []*unstructured.Unstructured{live}
		f.comparisonResult.reconciliationResult.Target = []*unstructured.Unstructured{target}

		// when
		targets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, targets, 1)
		containers, ok, err := unstructured.NestedSlice(targets[0].Object, "spec", "template", "spec", "containers")
		require.NoError(t, err)
		require.True(t, ok)
		assert.Len(t, containers, 1)

		ports := containers[0].(map[string]any)["ports"].([]any)
		assert.Len(t, ports, 1)

		env := containers[0].(map[string]any)["env"].([]any)
		assert.Len(t, env, 3)

		first := env[0]
		second := env[1]
		third := env[2]

		// Currently the defined order at this time is the insertion order of the target manifest.
		assert.Equal(t, "SOME_ENV_VAR", first.(map[string]any)["name"])
		assert.Equal(t, "some_value", first.(map[string]any)["value"])

		assert.Equal(t, "SOME_OTHER_ENV_VAR", second.(map[string]any)["name"])
		assert.Equal(t, "some_other_value", second.(map[string]any)["value"])

		assert.Equal(t, "YET_ANOTHER_ENV_VAR", third.(map[string]any)["name"])
		assert.Equal(t, "yet_another_value", third.(map[string]any)["value"])
	})

	t.Run("ignore-deployment-image-replicas-changes-additive", func(t *testing.T) {
		// given

		ignores := []v1alpha1.ResourceIgnoreDifferences{
			{
				Group:        "apps",
				Kind:         "Deployment",
				JSONPointers: []string{"/spec/replicas"},
			}, {
				Group:             "apps",
				Kind:              "Deployment",
				JQPathExpressions: []string{".spec.template.spec.containers[].image"},
			},
		}
		f := setupHTTPProxy(t, ignores)
		live := test.YamlToUnstructured(testdata.MinimalImageReplicaDeploymentYaml)
		target := test.YamlToUnstructured(testdata.AdditionalImageReplicaDeploymentYaml)
		f.comparisonResult.reconciliationResult.Live = []*unstructured.Unstructured{live}
		f.comparisonResult.reconciliationResult.Target = []*unstructured.Unstructured{target}

		// when
		targets, err := normalizeTargetResources(f.comparisonResult)

		// then
		require.NoError(t, err)
		require.Len(t, targets, 1)
		metadata, ok, err := unstructured.NestedMap(targets[0].Object, "metadata")
		require.NoError(t, err)
		require.True(t, ok)
		labels, ok := metadata["labels"].(map[string]any)
		require.True(t, ok)
		assert.Len(t, labels, 2)
		assert.Equal(t, "web", labels["appProcess"])

		spec, ok, err := unstructured.NestedMap(targets[0].Object, "spec")
		require.NoError(t, err)
		require.True(t, ok)

		assert.Equal(t, int64(1), spec["replicas"])

		template, ok := spec["template"].(map[string]any)
		require.True(t, ok)

		tMetadata, ok := template["metadata"].(map[string]any)
		require.True(t, ok)
		tLabels, ok := tMetadata["labels"].(map[string]any)
		require.True(t, ok)
		assert.Len(t, tLabels, 2)
		assert.Equal(t, "web", tLabels["appProcess"])

		tSpec, ok := template["spec"].(map[string]any)
		require.True(t, ok)
		containers, ok, err := unstructured.NestedSlice(tSpec, "containers")
		require.NoError(t, err)
		require.True(t, ok)
		assert.Len(t, containers, 1)

		first := containers[0].(map[string]any)
		assert.Equal(t, "alpine:3", first["image"])

		resources, ok := first["resources"].(map[string]any)
		require.True(t, ok)
		requests, ok := resources["requests"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "400m", requests["cpu"])

		env, ok, err := unstructured.NestedSlice(first, "env")
		require.NoError(t, err)
		require.True(t, ok)
		assert.Len(t, env, 1)

		env0 := env[0].(map[string]any)
		assert.Equal(t, "EV", env0["name"])
		assert.Equal(t, "here", env0["value"])
	})
}

func TestDeriveServiceAccountMatchingNamespaces(t *testing.T) {
	t.Parallel()

	type fixture struct {
		project     *v1alpha1.AppProject
		application *v1alpha1.Application
		cluster     *v1alpha1.Cluster
	}

	setup := func(destinationServiceAccounts []v1alpha1.ApplicationDestinationServiceAccount, destinationNamespace, destinationServerURL, applicationNamespace string) *fixture {
		project := &v1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "argocd-ns",
				Name:      "testProj",
			},
			Spec: v1alpha1.AppProjectSpec{
				DestinationServiceAccounts: destinationServiceAccounts,
			},
		}
		app := &v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: applicationNamespace,
				Name:      "testApp",
			},
			Spec: v1alpha1.ApplicationSpec{
				Project: "testProj",
				Destination: v1alpha1.ApplicationDestination{
					Server:    destinationServerURL,
					Namespace: destinationNamespace,
				},
			},
		}
		cluster := &v1alpha1.Cluster{
			Server: "https://kubernetes.svc.local",
			Name:   "test-cluster",
		}
		return &fixture{
			project:     project,
			application: app,
			cluster:     cluster,
		}
	}

	t.Run("empty destination service accounts", func(t *testing.T) {
		// given an application referring a project with no destination service accounts
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := ""
		expectedErrMsg := "no matching service account found for destination server https://kubernetes.svc.local and namespace testns"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)
		assert.Equal(t, expectedSA, sa)

		// then, there should be an error saying no valid match was found
		assert.EqualError(t, err, expectedErrMsg)
	})

	t.Run("exact match of destination namespace", func(t *testing.T) {
		// given an application referring a project with exactly one destination service account that matches the application destination,
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should be no error and should use the right service account for impersonation
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("exact one match with multiple destination service accounts", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts having one exact match for application destination
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "guestbook",
				DefaultServiceAccount: "guestbook-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "guestbook-test",
				DefaultServiceAccount: "guestbook-test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should be no error and should use the right service account for impersonation
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("first match to be used when multiple matches are available", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts having multiple match for application destination
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa-3",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "guestbook",
				DefaultServiceAccount: "guestbook-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should be no error and it should use the first matching service account for impersonation
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("first match to be used when glob pattern is used", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts with glob patterns matching the application destination
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "test*",
				DefaultServiceAccount: "test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should not be any error and should use the first matching glob pattern service account for impersonation
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("no match among a valid list", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts with no matches for application destination
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "test1",
				DefaultServiceAccount: "test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "test2",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := ""
		expectedErrMsg := "no matching service account found for destination server https://kubernetes.svc.local and namespace testns"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should be an error saying no match was found
		require.EqualError(t, err, expectedErrMsg)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("app destination namespace is empty", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts with empty application destination namespace
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				DefaultServiceAccount: "test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "*",
				DefaultServiceAccount: "test-sa-2",
			},
		}
		destinationNamespace := ""
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:argocd-ns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should not be any error and the service account configured for with empty namespace should be used.
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("match done via catch all glob pattern", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts having a catch all glob pattern
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns1",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "*",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should not be any error and the catch all service account should be returned
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("match done via invalid glob pattern", func(t *testing.T) {
		// given an application referring a project with a destination service account having an invalid glob pattern for namespace
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "e[[a*",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := ""

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there must be an error as the glob pattern is invalid.
		require.ErrorContains(t, err, "invalid glob pattern for destination namespace")
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("sa specified with a namespace", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts having a matching service account specified with its namespace
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "myns:test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "*",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:myns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)
		assert.Equal(t, expectedSA, sa)

		// then, there should not be any error and the service account with its namespace should be returned.
		require.NoError(t, err)
	})

	t.Run("app destination name instead of server URL", func(t *testing.T) {
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "*",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)

		// Use destination name instead of server URL
		f.application.Spec.Destination.Server = ""
		f.application.Spec.Destination.Name = f.cluster.Name

		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)
		assert.Equal(t, expectedSA, sa)

		// then, there should not be any error and the service account with its namespace should be returned.
		require.NoError(t, err)
	})
}

func TestDeriveServiceAccountMatchingServers(t *testing.T) {
	t.Parallel()

	type fixture struct {
		project     *v1alpha1.AppProject
		application *v1alpha1.Application
		cluster     *v1alpha1.Cluster
	}

	setup := func(destinationServiceAccounts []v1alpha1.ApplicationDestinationServiceAccount, destinationNamespace, destinationServerURL, applicationNamespace string) *fixture {
		project := &v1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "argocd-ns",
				Name:      "testProj",
			},
			Spec: v1alpha1.AppProjectSpec{
				DestinationServiceAccounts: destinationServiceAccounts,
			},
		}
		app := &v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: applicationNamespace,
				Name:      "testApp",
			},
			Spec: v1alpha1.ApplicationSpec{
				Project: "testProj",
				Destination: v1alpha1.ApplicationDestination{
					Server:    destinationServerURL,
					Namespace: destinationNamespace,
				},
			},
		}
		cluster := &v1alpha1.Cluster{
			Server: "https://kubernetes.svc.local",
			Name:   "test-cluster",
		}
		return &fixture{
			project:     project,
			application: app,
			cluster:     cluster,
		}
	}

	t.Run("exact one match with multiple destination service accounts", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts and one exact match for application destination
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "guestbook",
				DefaultServiceAccount: "guestbook-sa",
			},
			{
				Server:                "https://abc.svc.local",
				Namespace:             "guestbook",
				DefaultServiceAccount: "guestbook-test-sa",
			},
			{
				Server:                "https://cde.svc.local",
				Namespace:             "guestbook",
				DefaultServiceAccount: "default-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should not be any error and the right service account must be returned.
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("first match to be used when multiple matches are available", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts and multiple matches for application destination
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "guestbook",
				DefaultServiceAccount: "guestbook-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should not be any error and first matching service account should be used
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("first match to be used when glob pattern is used", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts with a matching glob pattern and exact match
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "test*",
				DefaultServiceAccount: "test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)
		assert.Equal(t, expectedSA, sa)

		// then, there should not be any error and the service account of the glob pattern, being the first match should be returned.
		require.NoError(t, err)
	})

	t.Run("no match among a valid list", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts with no match
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa",
			},
			{
				Server:                "https://abc.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://cde.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://xyz.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := ""
		expectedErr := "no matching service account found for destination server https://xyz.svc.local and namespace testns"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, &v1alpha1.Cluster{Server: destinationServerURL})

		// then, there an error with appropriate message must be returned
		require.EqualError(t, err, expectedErr)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("match done via catch all glob pattern", func(t *testing.T) {
		// given an application referring a project with multiple destination service accounts with matching catch all glob pattern
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "testns1",
				DefaultServiceAccount: "test-sa-2",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
			{
				Server:                "*",
				Namespace:             "*",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://localhost:6443"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there should not be any error and the service account of the glob pattern match must be returned.
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("match done via invalid glob pattern", func(t *testing.T) {
		// given an application referring a project with a destination service account having an invalid glob pattern for server
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "e[[a*",
				Namespace:             "test-ns",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := ""

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)

		// then, there must be an error as the glob pattern is invalid.
		require.ErrorContains(t, err, "invalid glob pattern for destination server")
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("sa specified with a namespace", func(t *testing.T) {
		// given app sync impersonation feature is enabled and matching service account is prefixed with a namespace
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://abc.svc.local",
				Namespace:             "testns",
				DefaultServiceAccount: "myns:test-sa",
			},
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "default",
				DefaultServiceAccount: "default-sa",
			},
			{
				Server:                "*",
				Namespace:             "*",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://abc.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:myns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)
		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, &v1alpha1.Cluster{Server: destinationServerURL})

		// then, there should not be any error and the service account with the given namespace prefix must be returned.
		require.NoError(t, err)
		assert.Equal(t, expectedSA, sa)
	})

	t.Run("app destination name instead of server URL", func(t *testing.T) {
		t.Parallel()
		destinationServiceAccounts := []v1alpha1.ApplicationDestinationServiceAccount{
			{
				Server:                "https://kubernetes.svc.local",
				Namespace:             "*",
				DefaultServiceAccount: "test-sa",
			},
		}
		destinationNamespace := "testns"
		destinationServerURL := "https://kubernetes.svc.local"
		applicationNamespace := "argocd-ns"
		expectedSA := "system:serviceaccount:testns:test-sa"

		f := setup(destinationServiceAccounts, destinationNamespace, destinationServerURL, applicationNamespace)

		// Use destination name instead of server URL
		f.application.Spec.Destination.Server = ""
		f.application.Spec.Destination.Name = f.cluster.Name

		// when
		sa, err := deriveServiceAccountToImpersonate(f.project, f.application, f.cluster)
		assert.Equal(t, expectedSA, sa)

		// then, there should not be any error and the service account with its namespace should be returned.
		require.NoError(t, err)
	})
}

func TestSyncWithImpersonate(t *testing.T) {
	type fixture struct {
		application *v1alpha1.Application
		project     *v1alpha1.AppProject
		controller  *ApplicationController
	}

	setup := func(impersonationEnabled bool, destinationNamespace, serviceAccountName string) *fixture {
		app := newFakeApp()
		app.Status.OperationState = nil
		app.Status.History = nil
		project := &v1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: test.FakeArgoCDNamespace,
				Name:      "default",
			},
			Spec: v1alpha1.AppProjectSpec{
				DestinationServiceAccounts: []v1alpha1.ApplicationDestinationServiceAccount{
					{
						Server:                "https://localhost:6443",
						Namespace:             destinationNamespace,
						DefaultServiceAccount: serviceAccountName,
					},
				},
			},
		}
		additionalObjs := []runtime.Object{}
		if serviceAccountName != "" {
			syncServiceAccount := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceAccountName,
					Namespace: test.FakeDestNamespace,
				},
			}
			additionalObjs = append(additionalObjs, syncServiceAccount)
		}
		data := fakeData{
			apps: []runtime.Object{app, project},
			manifestResponse: &apiclient.ManifestResponse{
				Manifests: []string{},
				Namespace: test.FakeDestNamespace,
				Server:    "https://localhost:6443",
				Revision:  "abc123",
			},
			managedLiveObjs: map[kube.ResourceKey]*unstructured.Unstructured{},
			configMapData: map[string]string{
				"application.sync.impersonation.enabled": strconv.FormatBool(impersonationEnabled),
			},
			additionalObjs: additionalObjs,
		}
		ctrl := newFakeController(&data, nil)
		return &fixture{
			application: app,
			project:     project,
			controller:  ctrl,
		}
	}

	t.Run("sync with impersonation and no matching service account", func(t *testing.T) {
		// given app sync impersonation feature is enabled with an application referring a project no matching service account
		f := setup(true, test.FakeArgoCDNamespace, "")
		opMessage := "failed to find a matching service account to impersonate: no matching service account found for destination server https://localhost:6443 and namespace fake-dest-ns"

		opState := &v1alpha1.OperationState{
			Operation: v1alpha1.Operation{
				Sync: &v1alpha1.SyncOperation{
					Source: &v1alpha1.ApplicationSource{},
				},
			},
			Phase: synccommon.OperationRunning,
		}
		// when
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then, app sync should fail with expected error message in operation state
		assert.Equal(t, synccommon.OperationError, opState.Phase)
		assert.Contains(t, opState.Message, opMessage)
	})

	t.Run("sync with impersonation and empty service account match", func(t *testing.T) {
		// given app sync impersonation feature is enabled with an application referring a project matching service account that is an empty string
		f := setup(true, test.FakeDestNamespace, "")
		opMessage := "failed to find a matching service account to impersonate: default service account contains invalid chars ''"

		opState := &v1alpha1.OperationState{
			Operation: v1alpha1.Operation{
				Sync: &v1alpha1.SyncOperation{
					Source: &v1alpha1.ApplicationSource{},
				},
			},
			Phase: synccommon.OperationRunning,
		}
		// when
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then app sync should fail with expected error message in operation state
		assert.Equal(t, synccommon.OperationError, opState.Phase)
		assert.Contains(t, opState.Message, opMessage)
	})

	t.Run("sync with impersonation and matching sa", func(t *testing.T) {
		// given app sync impersonation feature is enabled with an application referring a project matching service account
		f := setup(true, test.FakeDestNamespace, "test-sa")
		opMessage := "successfully synced (no more tasks)"

		opState := &v1alpha1.OperationState{
			Operation: v1alpha1.Operation{
				Sync: &v1alpha1.SyncOperation{
					Source: &v1alpha1.ApplicationSource{},
				},
			},
			Phase: synccommon.OperationRunning,
		}
		// when
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then app sync should not fail
		assert.Equal(t, synccommon.OperationSucceeded, opState.Phase)
		assert.Contains(t, opState.Message, opMessage)
	})

	t.Run("sync without impersonation", func(t *testing.T) {
		// given app sync impersonation feature is disabled with an application referring a project matching service account
		f := setup(false, test.FakeDestNamespace, "")
		opMessage := "successfully synced (no more tasks)"

		opState := &v1alpha1.OperationState{
			Operation: v1alpha1.Operation{
				Sync: &v1alpha1.SyncOperation{
					Source: &v1alpha1.ApplicationSource{},
				},
			},
			Phase: synccommon.OperationRunning,
		}
		// when
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then application sync should pass using the control plane service account
		assert.Equal(t, synccommon.OperationSucceeded, opState.Phase)
		assert.Contains(t, opState.Message, opMessage)
	})

	t.Run("app destination name instead of server URL", func(t *testing.T) {
		// given app sync impersonation feature is enabled with an application referring a project matching service account
		f := setup(true, test.FakeDestNamespace, "test-sa")
		opMessage := "successfully synced (no more tasks)"

		opState := &v1alpha1.OperationState{
			Operation: v1alpha1.Operation{
				Sync: &v1alpha1.SyncOperation{
					Source: &v1alpha1.ApplicationSource{},
				},
			},
			Phase: synccommon.OperationRunning,
		}

		f.application.Spec.Destination.Server = ""
		f.application.Spec.Destination.Name = "minikube"

		// when
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then app sync should not fail
		assert.Equal(t, synccommon.OperationSucceeded, opState.Phase)
		assert.Contains(t, opState.Message, opMessage)
	})
}

func TestClientSideApplyMigration(t *testing.T) {
	t.Parallel()

	type fixture struct {
		application *v1alpha1.Application
		project     *v1alpha1.AppProject
		controller  *ApplicationController
	}

	setup := func(disableMigration bool, customManager string) *fixture {
		app := newFakeApp()
		app.Status.OperationState = nil
		app.Status.History = nil

		// Add sync options
		if disableMigration {
			app.Spec.SyncPolicy.SyncOptions = append(app.Spec.SyncPolicy.SyncOptions, "DisableClientSideApplyMigration=true")
		}

		// Add custom manager annotation if specified
		if customManager != "" {
			app.Annotations = map[string]string{
				"argocd.argoproj.io/client-side-apply-migration-manager": customManager,
			}
		}

		project := &v1alpha1.AppProject{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: test.FakeArgoCDNamespace,
				Name:      "default",
			},
		}
		data := fakeData{
			apps: []runtime.Object{app, project},
			manifestResponse: &apiclient.ManifestResponse{
				Manifests: []string{},
				Namespace: test.FakeDestNamespace,
				Server:    test.FakeClusterURL,
				Revision:  "abc123",
			},
			managedLiveObjs: make(map[kube.ResourceKey]*unstructured.Unstructured),
		}
		ctrl := newFakeController(&data, nil)

		return &fixture{
			application: app,
			project:     project,
			controller:  ctrl,
		}
	}

	t.Run("client-side apply migration enabled by default", func(t *testing.T) {
		// given
		t.Parallel()
		f := setup(false, "")

		// when
		opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
			Sync: &v1alpha1.SyncOperation{
				Source: &v1alpha1.ApplicationSource{},
			},
		}}
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then
		assert.Equal(t, synccommon.OperationSucceeded, opState.Phase)
		assert.Contains(t, opState.Message, "successfully synced")
	})

	t.Run("client-side apply migration disabled", func(t *testing.T) {
		// given
		t.Parallel()
		f := setup(true, "")

		// when
		opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
			Sync: &v1alpha1.SyncOperation{
				Source: &v1alpha1.ApplicationSource{},
			},
		}}
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then
		assert.Equal(t, synccommon.OperationSucceeded, opState.Phase)
		assert.Contains(t, opState.Message, "successfully synced")
	})

	t.Run("client-side apply migration with custom manager", func(t *testing.T) {
		// given
		t.Parallel()
		f := setup(false, "my-custom-manager")

		// when
		opState := &v1alpha1.OperationState{Operation: v1alpha1.Operation{
			Sync: &v1alpha1.SyncOperation{
				Source: &v1alpha1.ApplicationSource{},
			},
		}}
		f.controller.appStateManager.SyncAppState(f.application, f.project, opState)

		// then
		assert.Equal(t, synccommon.OperationSucceeded, opState.Phase)
		assert.Contains(t, opState.Message, "successfully synced")
	})
}

func dig(obj any, path ...any) any {
	i := obj

	for _, segment := range path {
		switch segment := segment.(type) {
		case int:
			i = i.([]any)[segment]
		case string:
			i = i.(map[string]any)[segment]
		default:
			panic("invalid path for object")
		}
	}

	return i
}
