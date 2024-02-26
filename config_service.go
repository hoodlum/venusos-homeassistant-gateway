package main

import (
	"encoding/json"
	"flag"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"slices"
)

// type LookupTable map[string][]BatchEntry

type Config struct {
	configFile   string
	mqttServer   string
	mqttClientId string
	//lookupTable    LookupTable
	monitoringItem []MonitoringItem
}

type BatchEntry struct {
	//DbusName string
	DataType string //encoding
	Name     string
	//	Topic    string
	DbusPath  string
	Unit      string
	Direction string
}

type Batch struct {
	DbusName       string
	UpdateStrategy string
	Entries        []BatchEntry
}

type MonitoringItem struct {
	DbusName   string
	Member     string
	ObjectPath string
	Entries    []BatchEntry
}

func getConfig() Config {

	configFile := flag.String("config", "lookupTable.json", "path of config file ")
	mqttServer := flag.String("server", "192.168.178.3:1883", "IP:Port")
	//mqttQos := flag.Int("qos", 0, "The QoS to subscribe to messages at")
	mqttClientId := flag.String("clientid", "vz-homeassistant-gateway", "A clientid for the connection")
	//username := flag.String("username", "", "A username to authenticate to the MQTT server")
	//password := flag.String("password", "", "Password to match username")
	flag.Parse()
	log.Infof("Gateway: Load config from %s", *configFile)
	log.Infof("MQTT: clientId=%s", *mqttClientId)
	log.Infof("MQTT: server=%s", *mqttServer)

	batches, err := loadBatchesFromConfig(*configFile)
	if err != nil {
		log.Info("Gateway: Error parsing configFile")
		os.Exit(1)
	}

	return Config{
		mqttClientId: *mqttClientId,
		mqttServer:   *mqttServer,
		configFile:   *configFile,
		//lookupTable:    extractLookupTable(batches),
		monitoringItem: extractMonitoringItems(batches),
	}
}

func loadBatchesFromConfig(fileName string) ([]Batch, error) {

	var input []Batch

	// Open our jsonFile
	if jsonFile, err := os.Open(fileName); err == nil {

		// read our opened jsonFile as a byte array.
		b, _ := io.ReadAll(jsonFile)

		if err := json.Unmarshal(b, &input); err != nil {
			log.Info("Gateway: Error parse batch config: ", err)
			return nil, err
		} else {
			return input, nil
		}

	} else {
		return nil, err
	}

}

/*
func extractLookupTable(batches []Batch) LookupTable {

	var key string
	lookupTable := make(LookupTable)

	for _, batch := range batches {

		for _, f := range batch.Entries {

			//key = batch.DbusName + "%" + f.DbusPath
			key = batch.DbusName

			a := lookupTable[key]
			if lookupTable[key] == nil {
				a = make([]BatchEntry, 0)
			}
			//f.DbusName = batch.DbusName
			a = append(a, f)
			lookupTable[key] = a
		}

	}

	return lookupTable
}
*/

func extractMonitoringItems(batches []Batch) []MonitoringItem {

	monitoringItems := make([]MonitoringItem, 0, 10)

	for _, batch := range batches {

		if batch.UpdateStrategy == "batch" {

			monitoringItem := MonitoringItem{
				DbusName:   batch.DbusName,
				Member:     "ItemsChanged",
				ObjectPath: "/", //Root path of sender
				Entries:    batch.Entries,
			}

			monitoringItems = append(monitoringItems, monitoringItem)
		} else {

			for _, f := range batch.Entries {

				idx := slices.IndexFunc(monitoringItems, func(i MonitoringItem) bool { return i.ObjectPath == f.DbusPath })
				if idx == -1 {
					monitoringItems = append(monitoringItems, MonitoringItem{
						DbusName:   batch.DbusName,
						Member:     "PropertiesChanged",
						ObjectPath: f.DbusPath,
						Entries:    []BatchEntry{f},
					})

				} else {
					monitoringItems[idx].Entries = append(monitoringItems[idx].Entries, f)
				}

			}
		}

	}

	return monitoringItems

}
