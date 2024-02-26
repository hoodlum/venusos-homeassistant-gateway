package main

import (
	"github.com/godbus/dbus/v5"
	log "github.com/sirupsen/logrus"
)

var conn *dbus.Conn

func initDbusService() error {

	var err error
	conn, err = dbus.SystemBus()
	return err

}

func getBusName(conn *dbus.Conn, wellKnownName string) string {
	var s string
	err := conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, wellKnownName).Store(&s)
	if err != nil {
		log.Error("DBUS: Failed to get list of owned names:", err)
	}
	log.Infof("DBUS: well-known name %s -> %s", wellKnownName, s)

	return s
}
