// Package e2e is the end-to-end suite for a fresh Zenith install.
//
// The harness (infra/e2e) installs Zenith on a fresh k3d (k3s in Docker) cluster and describes it
// through ZENITH_E2E_* environment variables (see Config). The tests carry the
// build tag e2e and talk to the server only through its API, the zen binary
// and the harness hooks (server exec, reboot, copy); they never run docker or kubectl.
//
//	go test -tags e2e -count=1 -timeout 40m ./e2e/ -run 'TestInstall|TestDeployWithDatabaseAndDomain|TestReboot|TestBackup$|TestUpgrade'
//
// Run order is part of the contract: go test runs tests in source order with files
// taken alphabetically, so the files are numbered in the order the tests must run:
// 01 TestInstall, 02 TestDeployWithDatabaseAndDomain (writes <artifacts>/state.json),
// 03 TestReboot and 04 TestBackup (both need that state), 05 TestBackupRestore
// (restore run on another cluster), 06 TestUpgrade (needs the state too, runs last on the first cluster).
//
// Only the configuration parsing is built without the tag, so a plain
// go test ./... covers it.
package e2e
