package main

import (
	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
	"os"
)

var conn *dbus.Conn
var dbusNames map[string]string

func initDbusService() {

	var err error
	conn, err = dbus.SystemBus()
	if err == nil {
		log.Info("DBUS: Succesfully connected to Systembus")
	} else {
		log.Info("DBUS: Could not connect to Systembus")
		os.Exit(1)
	}
}

func closeDbusService() {
	conn.Close()
}

func getBusName(conn *dbus.Conn, wellKnownName string) string {
	var s string
	err := conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, wellKnownName).Store(&s)
	if err != nil {
		log.Error("DBUS: Failed to get list of owned names:", err)
	}

	return s
}

func startDbusMonitoring(conn *dbus.Conn, batches map[string][]BatchEntry, watchdog *Watchdog) {

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

func initDbusMonitor(conn *dbus.Conn, monitoringItems []MonitoringItem) {

	for wellKnownDbusName, entries := range batches {

		dbusName = getBusName(conn, wellKnownDbusName)
		log.Infof("DBUS: well-known dbus name %s -> %s", wellKnownDbusName, dbusName)
		dbusNames[dbusName] = wellKnownDbusName

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

func getKeyFromSignal(sig *dbus.Signal) string {

	if sig.Sender == dbusName {
		return "com.victronenergy.battery.ttyUSB0" + "%" + string(sig.Path)
	}

	return sig.Sender + "%" + string(sig.Path)
}
