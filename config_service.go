package main

import (
	"encoding/json"
	"flag"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

type Config struct {
	configFile   string
	mqttServer   string
	mqttClientId string
	batches      map[string][]BatchEntry
}

type BatchEntry struct {
	DbusName string
	DataType string //encoding
	Name     string
	//	Topic    string
	DbusPath  string
	Unit      string
	Direction string
}

type Batch struct {
	DbusName string
	Entries  []BatchEntry
}

func getConfig() (Config, error) {

	configFile := flag.String("config", "batches.json", "path of config file ")
	mqttServer := flag.String("server", "192.168.178.3:1883", "IP:Port")
	//mqttQos := flag.Int("qos", 0, "The QoS to subscribe to messages at")
	mqttClientId := flag.String("clientid", "vz-homeassistant-gateway", "A clientid for the connection")
	//username := flag.String("username", "", "A username to authenticate to the MQTT server")
	//password := flag.String("password", "", "Password to match username")
	flag.Parse()
	log.Infof("Gateway: Load config from %s", *configFile)
	log.Infof("MQTT: clientId=%s", *mqttClientId)
	log.Infof("MQTT: server=%s", *mqttServer)

	batches, err := loadBatchFromConfig(*configFile)

	if err != nil {
		return Config{}, err
	}

	return Config{
		mqttClientId: *mqttClientId,
		mqttServer:   *mqttServer,
		configFile:   *configFile,
		batches:      batches,
	}, nil

}

func loadBatchFromConfig(fileName string) (map[string][]BatchEntry, error) {

	var input []Batch

	// Open our jsonFile
	if jsonFile, err := os.Open(fileName); err == nil {

		//var batch []Batch

		// read our opened jsonFile as a byte array.
		b, _ := io.ReadAll(jsonFile)

		if err := json.Unmarshal(b, &input); err != nil {
			log.Info("Gateway: Error parse batch config: ", err)
			return nil, err
		}
	}

	var key string
	batches := make(map[string][]BatchEntry)

	for _, batch := range input {

		for _, f := range batch.Entries {
			key = batch.DbusName + "%" + f.DbusPath
			a := batches[key]
			if batches[key] == nil {
				a = make([]BatchEntry, 0)
			}
			f.DbusName = batch.DbusName
			a = append(a, f)
			batches[key] = a
		}

	}

	return batches, nil
}
