package main

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var watchdog *Watchdog
var dbusName string

func init() {
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		lvl = "info"
	}

	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}

	log.SetLevel(ll)
}

func main() {

	config := getConfig()

	startMqttClient(config.mqttServer, mqttClient.ClientID)
	createAutoDiscovery(config.lookupTable)

	initDbusService()

	terminate := make(chan bool, 1)
	startHandlerOS(terminate)

	startWatchdog(terminate)

	initDbusMonitor(conn, config.monitoringItem)

	go startDbusMonitoring(conn, config.lookupTable, watchdog)

	log.Info("Gateway: wait for termination")

	<-terminate

	defer closeDbusService()

}

func startWatchdog(terminate chan bool) {
	watchdog = CreateWatchdog(time.Second*10, func() {
		log.Info("Watchdog: triggered, kill process to allow restart by venus-os")
		terminate <- true
	})
}

func startHandlerOS(terminate chan bool) {
	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ic
		terminate <- true
	}()
}
