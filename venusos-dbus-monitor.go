package main

import (
	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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

	var err error
	//var batches map[string][]BatchEntry
	//var conn *dbus.Conn

	config, err := getConfig()
	if err != nil {
		os.Exit(1)
	}

	startMqttClient(config.mqttServer, mqttClient.ClientID)
	createAutoDiscovery(config.batches)

	terminate := make(chan bool, 1)
	initDbusService()
	conn, err = dbus.SystemBus()

	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ic
		terminate <- true
	}()

	if err != nil {
		log.Info("DBUS: Could not connect to Systembus")
	}

	log.Info("DBUS: connected to Systembus")

	watchdog := CreateWatchdog(time.Second*10, func() {
		//fmt.Println("Watchdog triggered, handle situation")
		log.Info("Watchdog: triggered, kill process to allow restart by venus-os")
		terminate <- true
	})

	dbusName = getBusName(conn, "com.victronenergy.battery.ttyUSB0")
	initDbusMonitor(conn, batches)

	go startMonitoring(conn, batches, watchdog)

	log.Info("Gateway: wait for termination")

	<-terminate

	defer conn.Close()

}

func startMonitoring(conn *dbus.Conn, batches map[string][]BatchEntry, watchdog *Watchdog) {

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	for sig := range signals {
		log.Debugf("Message: %#v\n", sig)

		key := getKeyFromSignal(sig)
		log.Debugf("Key: %#v\n", key)

		if len(sig.Body) == 1 {

			m := sig.Body[0].(map[string]dbus.Variant) // {map[string]Variant{}, `@a{sv} {}`}
			vv := m["Value"]
			log.Debugf("Value: %#v\n", vv)

			if vv.Signature().String() == "d" {

				v := vv.Value().(float64)
				log.Debugf("Path: %s = %#f\n", sig.Path, v)

				if err := publishData(batches[key], v); err != nil {
					log.Errorf("Error %#v\\n\"", err)
				} else {
					watchdog.ResetWatchdog()
				}

			}
		}

	}
}

func getKeyFromSignal(sig *dbus.Signal) string {

	if sig.Sender == dbusName {
		return "com.victronenergy.battery.ttyUSB0" + "%" + string(sig.Path)
	}

	return sig.Sender + "%" + string(sig.Path)
}

func initDbusMonitor(conn *dbus.Conn, batches map[string][]BatchEntry) {

	for _, entries := range batches {

		for _, entry := range entries {
			log.Debugf("DBUS: [Name=%s] addMatch %s:%s", entry.Name, entry.DbusName, entry.DbusPath)
			if err := conn.AddMatchSignal(
				dbus.WithMatchSender(entry.DbusName),
				dbus.WithMatchMember("PropertiesChanged"),
				dbus.WithMatchObjectPath(dbus.ObjectPath(entry.DbusPath)),
			); err != nil {
				panic(err)
			}
		}
	}

}
