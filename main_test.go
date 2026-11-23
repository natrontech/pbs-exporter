package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestAggregateSnapshotsKeepsTypesApart(t *testing.T) {
	body := `{"data":[
		{"backup-type":"ct","backup-id":"101","backup-time":300,"comment":"Komodo","verification":{"state":"ok"}},
		{"backup-type":"ct","backup-id":"101","backup-time":200,"comment":"Komodo old","verification":{"state":"failed"}},
		{"backup-type":"vm","backup-id":"101","backup-time":100,"comment":"PDM","verification":{"state":"failed"}}
	]}`
	var response SnapshotResponse
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatal(err)
	}
	groups := aggregateSnapshots(response)
	if len(groups) != 2 {
		t.Fatalf("want 2 groups, got %d", len(groups))
	}
	ct := groups[backupGroup{"ct", "101"}]
	if ct.count != 2 || ct.lastTimeStamp != 300 || ct.lastVerify != "ok" || ct.name != "Komodo" {
		t.Errorf("ct/101 = %+v", *ct)
	}
	vm := groups[backupGroup{"vm", "101"}]
	if vm.count != 1 || vm.lastTimeStamp != 100 || vm.lastVerify != "failed" || vm.name != "PDM" {
		t.Errorf("vm/101 = %+v", *vm)
	}
}

// Fixtures recreated from real PBS 4.2.9 responses of the admin job
// endpoints (no secrets, example datastores, job ids within the 32 chars
// PBS allows); timestamps are unix seconds. The upids follow the PBS format
// UPID:node:pid:pstart:task_id:starttime:worker_type:worker_id:auth_id:
// with the start time as the sixth field in hex.
const verifyJobsBody = `{"data":[
	{"id":"v-2bf845d8-f2c5","store":"store1","schedule":"daily","last-run-state":"OK","last-run-endtime":1763011200,"next-run":1763097600,"last-run-upid":"UPID:pbs:00004F37:0039E469:0000002A:69155C70:verificationjob:store1:v-2bf845d8-f2c5:root@pam:"},
	{"id":"v-3d13cf0e-8b71","store":"store2","schedule":"sat 18:15","last-run-state":"verify failed - task unexpectedly no longer running","last-run-endtime":1762950000,"next-run":1763061300,"last-run-upid":"UPID:pbs:00004F37:0039E469:0000002B:69146D60:verificationjob:store2:v-3d13cf0e-8b71:root@pam:"},
	{"id":"v-9a1b7c2d-4e5f","store":"store1","schedule":"daily","next-run":1763097600},
	{"id":"v-5e6f7a8b-9c0d","store":"store2","schedule":"daily","last-run-state":"OK","last-run-endtime":1762951200,"next-run":1763061300,"last-run-upid":"not-a-upid"}
]}`

// The first sync job is currently running: PBS reports no last-run-state and
// no last-run-endtime until it finishes. The second one has no schedule (a
// manual job), so there is no next-run, and its task state is unknown.
const syncJobsBody = `{"data":[
	{"id":"s-0c9b01e0-2376","store":"store2","schedule":"daily","next-run":1763097600,"last-run-upid":"UPID:pbs:00004F37:0039E469:0000002C:6916ADF0:syncjob:s-0c9b01e0-2376:root@pam:"},
	{"id":"s-7f3a5b1c-9d2e","store":"store1","last-run-state":"unknown","last-run-endtime":1762956000,"last-run-upid":"UPID:pbs:00004F37:0039E469:0000002D:691484D0:syncjob:s-7f3a5b1c-9d2e:root@pam:"}
]}`

// The second prune job finished with warnings, which counts as success.
const pruneJobsBody = `{"data":[
	{"id":"default-store1-13b8b44b-1e45-435","store":"store1","schedule":"daily","last-run-state":"OK","last-run-endtime":1763010000,"next-run":1763097600,"last-run-upid":"UPID:pbs:00004F37:0039E469:0000002E:691557C0:prunejob:store1:default-store1-13b8b44b-1e45-435:root@pam:"},
	{"id":"default-store2-27c3d99e-91ab-2f4","store":"store2","schedule":"sat 18:15","last-run-state":"WARNINGS: 2","last-run-endtime":1762950000,"next-run":1763061300,"last-run-upid":"UPID:pbs:00004F37:0039E469:0000002F:69146D60:prunejob:store2:default-store2-27c3d99e-91ab-2f4:root@pam:"}
]}`

// gc jobs have no id (the datastore identifies them) and expose the task
// upid as "upid", not as "last-run-upid". The first job finished, the second
// one is currently running.
const gcJobsBody = `{"data":[
	{"store":"store2","schedule":"daily","last-run-state":"OK","last-run-endtime":1763000400,"next-run":1763097600,"upid":"UPID:pbs:00004F37:0039E469:00000030:69153240:garbage_collection:store2:root@pam:"},
	{"store":"store1","schedule":"daily","next-run":1763097600,"upid":"UPID:pbs:00004F37:0039E469:00000031:6916ADF0:garbage_collection:store1:root@pam:"}
]}`

// jobSample identifies one emitted job series by metric name and labels.
type jobSample struct {
	metric    string
	jobType   string
	id        string
	datastore string
}

func (s jobSample) String() string {
	return fmt.Sprintf("%s{type=%q,id=%q,datastore=%q}", s.metric, s.jobType, s.id, s.datastore)
}

// collectJobSamples scrapes the job endpoints of a fake PBS and returns the
// emitted job series with their values.
func collectJobSamples(t *testing.T, handler http.Handler) map[jobSample]float64 {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	exporter := NewExporter(server.URL, "root@pam", "token", "pbs-exporter")

	ch := make(chan prometheus.Metric, 256)
	exporter.getJobMetrics(ch)
	close(ch)

	names := map[*prometheus.Desc]string{
		job_running:                          "pbs_job_running",
		job_last_run_success:                 "pbs_job_last_run_success",
		job_last_run_start_timestamp_seconds: "pbs_job_last_run_start_timestamp_seconds",
		job_last_run_end_timestamp_seconds:   "pbs_job_last_run_end_timestamp_seconds",
		job_next_run_timestamp_seconds:       "pbs_job_next_run_timestamp_seconds",
	}
	samples := make(map[jobSample]float64)
	for m := range ch {
		var parsed dto.Metric
		if err := m.Write(&parsed); err != nil {
			t.Fatalf("failed to write metric: %v", err)
		}
		name, ok := names[m.Desc()]
		if !ok {
			t.Fatalf("unexpected metric %v", m.Desc())
		}
		labels := make(map[string]string)
		for _, label := range parsed.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}
		sample := jobSample{name, labels["type"], labels["id"], labels["datastore"]}
		if _, duplicate := samples[sample]; duplicate {
			t.Fatalf("duplicate series %v", sample)
		}
		samples[sample] = parsed.GetGauge().GetValue()
	}
	return samples
}

// syncPath is the path of syncJobsApi without its query string.
const syncPath = "/api2/json/admin/sync"

func jobsHandler(t *testing.T, verify, sync, prune, gc string) http.Handler {
	t.Helper()
	jobs := map[string]string{
		verifyJobsApi: verify,
		syncPath:      sync,
		pruneJobsApi:  prune,
		gcJobsApi:     gc,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == syncPath && r.URL.Query().Get("sync-direction") != "all" {
			t.Errorf("sync jobs requested without sync-direction=all: %s", r.URL)
		}
		body, ok := jobs[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(body)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// TestGetJobMetrics covers the job states PBS reports: OK, warnings, an
// error string, unknown, never-run and running (where the last-run series
// must be absent), jobs without a schedule (no next-run), the last-run
// start time parsed from the upid, and the gc specifics (no id, "upid"
// key, no running state).
func TestGetJobMetrics(t *testing.T) {
	samples := collectJobSamples(t, jobsHandler(t, verifyJobsBody, syncJobsBody, pruneJobsBody, gcJobsBody))

	want := map[jobSample]float64{
		// verify: OK
		{"pbs_job_running", "verify", "v-2bf845d8-f2c5", "store1"}:                          0,
		{"pbs_job_last_run_success", "verify", "v-2bf845d8-f2c5", "store1"}:                 1,
		{"pbs_job_last_run_start_timestamp_seconds", "verify", "v-2bf845d8-f2c5", "store1"}: 1763007600,
		{"pbs_job_last_run_end_timestamp_seconds", "verify", "v-2bf845d8-f2c5", "store1"}:   1763011200,
		{"pbs_job_next_run_timestamp_seconds", "verify", "v-2bf845d8-f2c5", "store1"}:       1763097600,
		// verify: error string counts as failed
		{"pbs_job_running", "verify", "v-3d13cf0e-8b71", "store2"}:                          0,
		{"pbs_job_last_run_success", "verify", "v-3d13cf0e-8b71", "store2"}:                 0,
		{"pbs_job_last_run_start_timestamp_seconds", "verify", "v-3d13cf0e-8b71", "store2"}: 1762946400,
		{"pbs_job_last_run_end_timestamp_seconds", "verify", "v-3d13cf0e-8b71", "store2"}:   1762950000,
		// verify: never ran (no last-run series, but a next-run)
		{"pbs_job_next_run_timestamp_seconds", "verify", "v-9a1b7c2d-4e5f", "store1"}: 1763097600,
		// verify: malformed upid, only the start time series is skipped
		{"pbs_job_running", "verify", "v-5e6f7a8b-9c0d", "store2"}:                        0,
		{"pbs_job_last_run_success", "verify", "v-5e6f7a8b-9c0d", "store2"}:               1,
		{"pbs_job_last_run_end_timestamp_seconds", "verify", "v-5e6f7a8b-9c0d", "store2"}: 1762951200,
		{"pbs_job_next_run_timestamp_seconds", "verify", "v-5e6f7a8b-9c0d", "store2"}:     1763061300,
		// sync: running (upid without endtime), start time from the upid
		{"pbs_job_running", "sync", "s-0c9b01e0-2376", "store2"}:                          1,
		{"pbs_job_last_run_start_timestamp_seconds", "sync", "s-0c9b01e0-2376", "store2"}: 1763094000,
		{"pbs_job_next_run_timestamp_seconds", "sync", "s-0c9b01e0-2376", "store2"}:       1763097600,
		// sync: no schedule (manual job, no next-run), unknown task state
		{"pbs_job_running", "sync", "s-7f3a5b1c-9d2e", "store1"}:                          0,
		{"pbs_job_last_run_success", "sync", "s-7f3a5b1c-9d2e", "store1"}:                 0,
		{"pbs_job_last_run_start_timestamp_seconds", "sync", "s-7f3a5b1c-9d2e", "store1"}: 1762952400,
		{"pbs_job_last_run_end_timestamp_seconds", "sync", "s-7f3a5b1c-9d2e", "store1"}:   1762956000,
		// prune: OK and WARNINGS both count as success
		{"pbs_job_last_run_success", "prune", "default-store1-13b8b44b-1e45-435", "store1"}: 1,
		{"pbs_job_last_run_success", "prune", "default-store2-27c3d99e-91ab-2f4", "store2"}: 1,
		// gc: no id, the datastore is used as id; the start time comes from
		// the "upid" field
		{"pbs_job_last_run_success", "gc", "store2", "store2"}:                 1,
		{"pbs_job_last_run_start_timestamp_seconds", "gc", "store2", "store2"}: 1762996800,
		{"pbs_job_last_run_end_timestamp_seconds", "gc", "store2", "store2"}:   1763000400,
		{"pbs_job_next_run_timestamp_seconds", "gc", "store2", "store2"}:       1763097600,
		{"pbs_job_last_run_start_timestamp_seconds", "gc", "store1", "store1"}: 1763094000,
		{"pbs_job_next_run_timestamp_seconds", "gc", "store1", "store1"}:       1763097600,
	}
	for sample, value := range want {
		if got, ok := samples[sample]; !ok {
			t.Errorf("missing series %v", sample)
		} else if got != value {
			t.Errorf("series %v = %v, want %v", sample, got, value)
		}
	}

	absent := []jobSample{
		// running sync job: no last-run-state and no last-run-endtime
		{"pbs_job_last_run_success", "sync", "s-0c9b01e0-2376", "store2"},
		{"pbs_job_last_run_end_timestamp_seconds", "sync", "s-0c9b01e0-2376", "store2"},
		// never-run verify job: no last-run series at all
		{"pbs_job_running", "verify", "v-9a1b7c2d-4e5f", "store1"},
		{"pbs_job_last_run_success", "verify", "v-9a1b7c2d-4e5f", "store1"},
		{"pbs_job_last_run_start_timestamp_seconds", "verify", "v-9a1b7c2d-4e5f", "store1"},
		{"pbs_job_last_run_end_timestamp_seconds", "verify", "v-9a1b7c2d-4e5f", "store1"},
		// malformed upid: start time skipped, other series unaffected
		{"pbs_job_last_run_start_timestamp_seconds", "verify", "v-5e6f7a8b-9c0d", "store2"},
		// sync job without schedule: no next-run
		{"pbs_job_next_run_timestamp_seconds", "sync", "s-7f3a5b1c-9d2e", "store1"},
		// gc jobs expose no last-run-upid and get no running state, even
		// while one is running (their "upid" can be a stale fallback)
		{"pbs_job_running", "gc", "store2", "store2"},
		{"pbs_job_running", "gc", "store1", "store1"},
		{"pbs_job_last_run_success", "gc", "store1", "store1"},
		{"pbs_job_last_run_end_timestamp_seconds", "gc", "store1", "store1"},
	}
	for _, sample := range absent {
		if _, ok := samples[sample]; ok {
			t.Errorf("unexpected series %v, want it absent", sample)
		}
	}
}

// TestGetJobMetricsPartialFailure ensures a failing job endpoint in the
// middle of the collection loop (here a 400 for sync on PBS older than 3.3)
// is skipped while the endpoints collected after it are still reported.
func TestGetJobMetricsPartialFailure(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == syncPath {
			http.Error(w, "schema does not allow additional properties.", http.StatusBadRequest)
			return
		}
		jobsHandler(t, verifyJobsBody, syncJobsBody, pruneJobsBody, gcJobsBody).ServeHTTP(w, r)
	})

	samples := collectJobSamples(t, handler)

	if _, ok := samples[jobSample{"pbs_job_running", "sync", "s-0c9b01e0-2376", "store2"}]; ok {
		t.Error("sync series reported although the sync endpoint failed")
	}
	for _, sample := range []jobSample{
		{"pbs_job_last_run_success", "verify", "v-2bf845d8-f2c5", "store1"},
		{"pbs_job_last_run_success", "prune", "default-store1-13b8b44b-1e45-435", "store1"},
		{"pbs_job_last_run_success", "gc", "store2", "store2"},
	} {
		if _, ok := samples[sample]; !ok {
			t.Errorf("missing series %v although only the sync endpoint failed", sample)
		}
	}
}
