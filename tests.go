package main

import (
	"encoding/json"
	"fmt"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"os"
)

func monitorSignals(conn *dbus.Conn) {

	if err := conn.AddMatchSignal(
		dbus.WithMatchObjectPath("/Dc/0"),
		//dbus.WithMatchInterface("com.victronenergy.*"),
		//dbus.WithMatchInterface("com.victronenergy.BusItem"),
	); err != nil {
		panic(err)
	}

	//signals := make(chan *dbus.Signal, 10)
	//conn.Signal(signals)

	signals := make(chan *dbus.Message, 10)
	conn.Eavesdrop(signals)

	for {
		select {
		case message := <-signals:
			fmt.Println("Message:", message)
		}
	}
}

func listNames(conn *dbus.Conn) {

	var s []string
	var err = conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&s)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get list of owned names:", err)
		os.Exit(1)
	}

	fmt.Println("Currently owned names on the session bus:")
	for _, v := range s {
		fmt.Println(v)
	}

}

func doIntrospect(conn *dbus.Conn) {

	/*
		s is std::string.
		v is variant.
		a{} is std::map.
		a{sv} is std::map<std::string, Variant>
		a{sa{sv}} is std::map<std::string, std::map<std::string, Variant>>
	*/

	node, err := introspect.Call(conn.Object("com.victronenergy.BusItem", "/"))
	if err != nil {
		panic(err)
	}
	data, _ := json.MarshalIndent(node, "", "    ")
	os.Stdout.Write(data)

}

/*

	//for _, v := range []string{"method_call", "method_return", "error", "signal"} {
	for _, v := range []string{"signal"} {
		call := conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, "eavesdrop='true',type='"+v+"'")
		if call.Err != nil {
			fmt.Fprintln(os.Stderr, "Failed to add match:", call.Err)
			os.Exit(1)
		}
	}
	c := make(chan *dbus.Message, 10)
	conn.Eavesdrop(c)
	fmt.Println("Listening for everything")
	for v := range c {
		fmt.Println(v)
	}
*/
