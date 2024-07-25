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

type GatewayConfig struct {
	mqttServer     *string
	mqttTopic      *string
	mqttQos        *int
	mqttClientId   *string
	mqttUsername   *string
	mqttPassword   *string
	configFile     *string
	monitoringItem []MonitoringItem
	debugLevel     string
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

var config GatewayConfig

func setLogLevel() {
	var ok bool

	if config.debugLevel == "unknown" {

		config.debugLevel, ok = os.LookupEnv("LOG_LEVEL")
		if !ok {
			config.debugLevel = "info"
		}
	}

	ll, err := log.ParseLevel(config.debugLevel)
	if err != nil {
		ll = log.DebugLevel
	}

	log.SetLevel(ll)
}

func loadConfigFromArgs() {

	config.mqttServer = flag.String("server", "192.168.178.3:1883", "IP:Port")
	config.mqttTopic = flag.String("topic", "/smartmeter1/power", "Topic to subscribe to")
	config.mqttQos = flag.Int("qos", 0, "The QoS to subscribe to messages at")
	config.mqttClientId = flag.String("clientid", "vz-homeassistant-gateway", "A clientid for the connection")
	config.mqttUsername = flag.String("username", "", "A username to authenticate to the MQTT server")
	config.mqttPassword = flag.String("password", "", "Password to match username")
	config.configFile = flag.String("config", "lookupTable.json", "path of config file ")
	debugLevel := flag.String("debug_level", "none", "error, debug, info")

	flag.Parse()

	config.debugLevel = *debugLevel

	log.Infof("Gateway: Load config from %s", *config.configFile)
	log.Infof("MQTT: clientId=%s", *config.mqttClientId)
	log.Infof("MQTT: server=%s", *config.mqttServer)

	batches, err := loadBatchesFromConfig(*config.configFile)
	if err != nil {
		log.Info("Gateway: Error parsing configFile")
		os.Exit(1)
	}

	config.monitoringItem = extractMonitoringItems(batches)
	//lookupTable:    extractLookupTable(batches),

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
