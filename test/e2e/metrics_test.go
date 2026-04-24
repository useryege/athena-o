package e2e

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// . "github.com/useryege/athena/pkg/apis/application/v1alpha1"
	// . "github.com/useryege/athena/test/e2e/fixture/app"
)

func TestKubectlMetrics(t *testing.T) {
	// Sync an app so that there are metrics to scrape.
	// ctx := Given(t)
	// ctx.When()
	// ctx.
	// 	Path(guestbookPath).
	// 	When().
	// 	CreateApp().
	// 	Then().
	// 	Expect(SyncStatusIs(SyncStatusCodeOutOfSync)).
	// 	And(func(app *Application) {
	// 		assert.Equal(t, ctx.GetName(), app.Name)
	// 		assert.Equal(t, fixture.RepoURL(fixture.RepoURLTypeFile), app.Spec.GetSource().RepoURL)
	// 		assert.Equal(t, guestbookPath, app.Spec.GetSource().Path)
	// 		assert.Equal(t, ctx.DeploymentNamespace(), app.Spec.Destination.Namespace)
	// 		assert.Equal(t, KubernetesInternalAPIServerAddr, app.Spec.Destination.Server)
	// 	}).
	// 	Expect(Event(argo.EventReasonResourceCreated, "create")).
	// 	And(func(_ *Application) {
	// 		// app should be listed
	// 		output, err := fixture.RunCli("app", "list")
	// 		require.NoError(t, err)
	// 		assert.Contains(t, output, ctx.GetName())
	// 	}).
	// 	When().
	// 	// ensure that create is idempotent
	// 	CreateApp().
	// 	Then().
	// 	Given().
	// 	Revision("master").
	// 	When().
	// 	// ensure that update replaces spec and merge labels and annotations
	// 	And(func() {
	// 		errors.NewHandler(t).FailOnErr(fixture.AppClientset.ArgoprojV1alpha1().Applications(fixture.TestNamespace()).Patch(t.Context(),
	// 			ctx.GetName(), types.MergePatchType, []byte(`{"metadata": {"labels": { "test": "label" }, "annotations": { "test": "annotation" }}}`), metav1.PatchOptions{}))
	// 	}).
	// 	CreateApp("--upsert").
	// 	Then().
	// 	And(func(app *Application) {
	// 		assert.Equal(t, "label", app.Labels["test"])
	// 		assert.Equal(t, "annotation", app.Annotations["test"])
	// 		assert.Equal(t, "master", app.Spec.GetSource().TargetRevision)
	// 	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://127.0.0.1:8083/metrics", http.NoBody)
	require.NoError(t, err)

	// Repeat the test for port 8083, i.e. the API server.
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() {
		err = resp.Body.Close()
		require.NoError(t, err)
	}()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// t.Logf("metrics: %s", string(body))

	assert.Contains(t, string(body), "athena_kubectl_request_duration_seconds", "metrics should have contained athena_kubectl_request_duration_seconds")
	assert.Contains(t, string(body), "athena_kubectl_request_size_bytes", "metrics should have contained athena_kubectl_request_size_bytes")
	assert.Contains(t, string(body), "athena_kubectl_response_size_bytes", "metrics should have contained athena_kubectl_response_size_bytes")
	assert.Contains(t, string(body), "athena_kubectl_rate_limiter_duration_seconds", "metrics should have contained athena_kubectl_rate_limiter_duration_seconds")
	assert.Contains(t, string(body), "athena_kubectl_requests_total", "metrics should have contained athena_kubectl_requests_total")
	assert.Contains(t, string(body), "grpc_server_handled_total", "metrics should have contained grpc_server_handled_total for all the reflected methods")
	assert.Contains(t, string(body), "grpc_server_msg_received_total", "metrics should have contained grpc_server_msg_received_total for all the reflected methods")
}
