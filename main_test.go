package main

import (
	"encoding/json"
	"testing"
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
