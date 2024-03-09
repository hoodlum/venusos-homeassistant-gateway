package main

import (
	"github.com/godbus/dbus/v5"
	"github.com/samber/lo"
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

func startDbusMonitoring(conn *dbus.Conn, items []MonitoringItem, watchdog *Watchdog) {

	busNames := map[string]string{}

	x := lo.Uniq(
		lo.Map(items, func(item MonitoringItem, idx int) string { return item.DbusName }),
	)

	for _, wellKnownBusName := range x {
		busName := getBusName(conn, wellKnownBusName)
		busNames[busName] = wellKnownBusName
		log.Infof("DbusLookup: %s => %s", wellKnownBusName, busName)
	}

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	for sig := range signals {

		if len(sig.Body) != 1 {
			continue //skip
		}

		//log.Infof("Message: %#v\n", sig)
		//log.Debugf("Sender: %#v\n", wellKnownBusName)

		item, ok := lo.Find(items, func(item MonitoringItem) bool {
			return item.DbusName == busNames[sig.Sender] && item.ObjectPath == string(sig.Path)
		})

		if !ok {
			log.Debugf("DBUS: Did not find a matching entry: %#v\n", item.DbusName)
			continue //skip
		}

		if item.Member == "PropertiesChanged" {

			m := sig.Body[0].(map[string]dbus.Variant) // {map[string]Variant{}, `@a{sv} {}`}
			vv := m["Value"]
			log.Debugf("DBUS: Variant: %#v\n", vv)

			if vv.Signature().String() == "d" {

				v := vv.Value().(float64)
				log.Debugf("DBUS: Value: %s = %#f\n", sig.Path, v)

				if err := publishData(item.Entries, v); err != nil {
					log.Errorf("Error %#v\\n\"", err)
				} else {
					watchdog.ResetWatchdog()
				}

			}

			if vv.Signature().String() == "i" {

				v := vv.Value().(int32)
				log.Debugf("DBUS: Value: %s = %#f\n", sig.Path, v)

				if err := publishData(item.Entries, float64(v)); err != nil {
					log.Errorf("Error %#v\\n\"", err)
				} else {
					watchdog.ResetWatchdog()
				}

			}

		}

		if item.Member == "ItemsChanged" {

			/*
				paths := lo.Map(item.Entries, func(item BatchEntry, index int) string {
					return item.DbusPath
				})
			*/

			jsonData := map[string]interface{}{}

			//log.Infof("Message: %#v\n", sig)
			ma := sig.Body[0].(map[string]map[string]dbus.Variant) // {map[string]map[string]Variant{}, `@a{sa{sv}} {}`}

			for k, v := range ma {

				if e, ok := lo.Find(item.Entries, func(entry BatchEntry) bool {
					return entry.DbusPath == k
				}); ok {
					vv := v["Value"]
					log.Debugf("Key: %s -> %s[%s]\n", k, vv, vv.Signature().String())
					//for x := range v { log.Infof("  Key: %s \n", x)}
					//if vv.Signature().String() == "d" {}
					switch vv.Signature().String() {
					case "n":
						jsonData[removeSpace(e.Name)] = vv.Value().(int16)
					case "i":
						jsonData[removeSpace(e.Name)] = vv.Value().(int32)
					}

				}

			}

			if err := publishDataRaw(jsonData); err != nil {
				log.Errorf("Error %#v\\n\"", err)
			} else {
				watchdog.ResetWatchdog()
			}
		}

	}
}

func initDbusMonitor(conn *dbus.Conn, monitoringItems []MonitoringItem) {

	for _, entry := range monitoringItems {
		log.Debugf("DBUS: add Match [%s]:%s:%s", entry.DbusName, entry.ObjectPath, entry.Member)

		if err := conn.AddMatchSignal(
			dbus.WithMatchSender(entry.DbusName),
			dbus.WithMatchMember(entry.Member),
			dbus.WithMatchObjectPath(dbus.ObjectPath(entry.ObjectPath)),
		); err != nil {
			panic(err)
		}
	}

}

func getKeyFromSignal(sig *dbus.Signal) string {

	if sig.Sender == dbusName {
		return "com.victronenergy.battery.ttyUSB0" + "%" + string(sig.Path)
	}

	return sig.Sender + "%" + string(sig.Path)
}
