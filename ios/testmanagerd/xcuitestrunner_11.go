package testmanagerd

import (
	"context"
	"strings"

	"github.com/danielpaulus/go-ios/ios"
	dtx "github.com/danielpaulus/go-ios/ios/dtx_codec"
	"github.com/danielpaulus/go-ios/ios/instruments"
	log "github.com/sirupsen/logrus"
)

func RunXCUIWithBundleIds11Ctx(
	ctx context.Context,
	bundleID string,
	testRunnerBundleID string,
	xctestConfigFileName string,
	device ios.DeviceEntry,
	args []string,
	env []string,
) error {
	log.Debugf("set up xcuitest")
	testSessionId, xctestConfigPath, testConfig, testInfo, err := setupXcuiTest(device, bundleID, testRunnerBundleID, xctestConfigFileName)
	if err != nil {
		return err
	}
	log.Debugf("test session setup ok")
	ctx, cancelCtx := context.WithCancel(ctx)
	conn, err := dtx.NewConnection(device, testmanagerd, dtx.WithConnectionBreakdownCallback(cancelCtx))
	if err != nil {
		return err
	}
	defer conn.Close()
	ideDaemonProxy := newDtxProxyWithConfig(conn, testConfig)

	conn2, err := dtx.NewConnection(device, testmanagerd, dtx.WithConnectionBreakdownCallback(cancelCtx))
	if err != nil {
		return err
	}
	defer conn2.Close()
	log.Debug("connections ready")
	ideDaemonProxy2 := newDtxProxyWithConfig(conn2, testConfig)
	ideDaemonProxy2.ideInterface.testConfig = testConfig
	// TODO: fixme
	protocolVersion := uint64(25)
	_, err = ideDaemonProxy.daemonConnection.initiateSessionWithIdentifier(testSessionId, protocolVersion)
	if err != nil {
		return err
	}

	pControl, err := instruments.NewProcessControl(device)
	if err != nil {
		return err
	}
	defer pControl.Close()

	pid, err := startTestRunner11(pControl, xctestConfigPath, testRunnerBundleID, testSessionId.String(), testInfo.testrunnerAppPath+"/PlugIns/"+xctestConfigFileName, args, env)
	if err != nil {
		return err
	}
	log.Debugf("Runner started with pid:%d, waiting for testBundleReady", pid)

	err = ideDaemonProxy2.daemonConnection.initiateControlSession(pid, protocolVersion)
	if err != nil {
		return err
	}
	log.Debugf("control session initiated")
	ideInterfaceChannel := ideDaemonProxy.dtxConnection.ForChannelRequest(ProxyDispatcher{id: "emty"})

	log.Debug("start executing testplan")
	err = ideDaemonProxy2.daemonConnection.startExecutingTestPlanWithProtocolVersion(ideInterfaceChannel, 25)
	if err != nil {
		log.Error(err)
	}

	closeChanUsed := false
	select {
	case <-ctx.Done():
	case <-closeChan:
		closeChanUsed = true
	}
	log.Infof("Killing WebDriverAgent with pid %d ...", pid)
	err = pControl.KillProcess(pid)
	if err != nil {
		return err
	}
	log.Info("WDA killed successfully")

	if closeChanUsed {
		var signal interface{}
		closedChan <- signal
	}
	return nil
}

func startTestRunner11(pControl *instruments.ProcessControl, xctestConfigPath string, bundleID string,
	sessionIdentifier string, testBundlePath string, wdaargs []string, wdaenv []string,
) (uint64, error) {
	args := []interface{}{
		"-NSTreatUnknownArgumentsAsOpen", "NO", "-ApplePersistenceIgnoreState", "YES",
	}
	for _, arg := range wdaargs {
		args = append(args, arg)
	}
	env := map[string]interface{}{
		"CA_ASSERT_MAIN_THREAD_TRANSACTIONS": "0",
		"CA_DEBUG_TRANSACTIONS":              "0",
		"DYLD_INSERT_LIBRARIES":              "/Developer/usr/lib/libMainThreadChecker.dylib",

		"MTC_CRASH_ON_REPORT":             "1",
		"NSUnbufferedIO":                  "YES",
		"OS_ACTIVITY_DT_MODE":             "YES",
		"SQLITE_ENABLE_THREAD_ASSERTIONS": "1",
		"XCTestBundlePath":                testBundlePath,
		"XCTestConfigurationFilePath":     xctestConfigPath,
		"XCTestSessionIdentifier":         sessionIdentifier,
	}

	for _, entrystring := range wdaenv {
		entry := strings.Split(entrystring, "=")
		key := entry[0]
		value := entry[1]
		env[key] = value
		log.Infof("adding extra env %s=%s", key, value)
	}

	opts := map[string]interface{}{
		"StartSuspendedKey": uint64(0),
		"ActivateSuspended": uint64(1),
	}

	return pControl.StartProcess(bundleID, env, args, opts)
}
