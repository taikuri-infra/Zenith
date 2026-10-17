// Package e2e is the end-to-end suite for a fresh Zenith install.
//
// The harness (infra/e2e) installs Zenith on a fresh server and describes it
// through ZENITH_E2E_* environment variables (see Config). The tests carry the
// build tag e2e and talk to the server only through its API, the zen binary
// and the harness hooks (server exec, reboot, copy); they never run docker.
//
//	go test -tags e2e -count=1 -timeout 40m ./e2e/ -run 'TestInstall|TestDeployWithDatabaseAndDomain|TestReboot|TestBackup$'
//
// Run order is part of the contract: go test runs tests in source order with files
// taken alphabetically, so the files are numbered in the order the tests must run:
// 01 TestInstall, 02 TestDeployWithDatabaseAndDomain (writes <artifacts>/state.json),
// 03 TestReboot and 04 TestBackup (both need that state), 05 TestBackupRestore
// (restore run on another server), 06 TestUpgrade (its own server).
//
// Only the configuration parsing is built without the tag, so a plain
// go test ./... covers it.
package e2e
