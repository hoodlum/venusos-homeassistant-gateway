package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	log "github.com/sirupsen/logrus"
	"math"
	"net"
	"os"
	"os/signal"
	"syscall"
	"unicode"
)

type HomeAssistantDevice struct {
	Name         string   `json:"name"`
	Model        string   `json:"model"`
	SwVersion    string   `json:"sw_version"`
	Manufacturer string   `json:"manufacturer"`
	Identifiers  []string `json:"identifiers"`
}

type HomeAssistantMeta struct {
	Name       string              `json:"name"`
	UniqId     string              `json:"uniq_id"`
	StatT      string              `json:"stat_t"`
	DevCla     string              `json:"dev_cla"`
	StatCla    string              `json:"stat_cla"`
	ValTpl     string              `json:"val_tpl"`
	UnitOfMeas string              `json:"unit_of_meas"`
	Device     HomeAssistantDevice `json:"device"`
	Expire     string              `json:"expire_after"`
}

const homeAssistantMetaTopic = "homeassistant/sensor/"

//const homeAssistantMetaTopic = "test/homeassistant/sensor/"

const deviceName = "jkbms"
const deviceModel = "diy-batterie"
const uniqueId = "batt1"
const swVersion = "v0.1"
const manufacturer = "DIY"

const statusTopic = deviceName + "/" + uniqueId

//const statusTopic = "test/" + deviceName + "/" + uniqueId

var mqttClient *paho.Client

func createAutoDiscovery(items []MonitoringItem) {
	for _, item := range items {
		createAutoDiscoveryMeta(item)
	}
}

func createAutoDiscoveryMeta(item MonitoringItem) {

	for _, entry := range item.Entries {

		devCla := ""
		statCla := "measurement"
		switch entry.Unit {
		case "Wh":
			devCla = "energy"
			statCla = "total_increasing"
		case "s":
			devCla = "duration"
		case "W":
			devCla = "power"
			statCla = "measurement"
		case "kWh":
			devCla = "energy"
			statCla = "total_increasing"
		case "MWh":
			devCla = "energy"
			statCla = "total_increasing"
		case "A":
			devCla = "current"
			statCla = "measurement"
		case "V":
			devCla = "voltage"
			statCla = "measurement"
		case "°C":
			devCla = "temperature"
			statCla = "measurement"
		case "%":
			devCla = "battery"
		}

		cleanedSensorName := removeSpace(entry.Name)

		ham := &HomeAssistantMeta{
			Name:       fmt.Sprintf("%s.%s", deviceName, cleanedSensorName),
			UniqId:     fmt.Sprintf("%s_%s", uniqueId, cleanedSensorName),
			StatT:      statusTopic,
			DevCla:     devCla,
			StatCla:    statCla,
			ValTpl:     fmt.Sprintf("{{ value_json.%s | is_defined }}", cleanedSensorName),
			UnitOfMeas: entry.Unit,
			Device: HomeAssistantDevice{
				Name:         deviceName,
				Model:        deviceModel,
				SwVersion:    swVersion,
				Manufacturer: manufacturer,
				Identifiers:  []string{uniqueId},
			},
		}

		if devCla == "measurement" {
			ham.Expire = "60"
		}

		if payload, err := json.Marshal(ham); err == nil {

			topic := homeAssistantMetaTopic + deviceName + "/" + ham.UniqId + "/config"

			if _, err := mqttClient.Publish(context.Background(), &paho.Publish{
				Topic:   topic,
				QoS:     0,
				Retain:  true,
				Payload: payload,
			}); err != nil {
				log.Debugln("MQTT: error sending message:", err)
			}
			//log.Debugln("MQTT: sent")
		}

	}

}

func removeSpace(s string) string {
	rr := make([]rune, 0, len(s))
	for _, r := range s {
		if r == ('[') || r == ']' {
			rr = append(rr, '_')
		} else if !unicode.IsSpace(r) {
			rr = append(rr, unicode.ToLower(r))
		} else {
			rr = append(rr, '_')
		}
	}
	return string(rr)
}

func startMqttClient(mqttServer, mqttClientId string) {

	conn, err := net.Dial("tcp", mqttServer)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %s", mqttServer, err)
	}

	mqttClient = paho.NewClient(paho.ClientConfig{
		Conn: conn,
	})

	mqttClient.SetErrorLogger(log.StandardLogger())

	cp := &paho.Connect{
		KeepAlive:    30,
		ClientID:     mqttClientId,
		CleanStart:   true,
		UsernameFlag: true,
		PasswordFlag: true,
	}

	ca, err := mqttClient.Connect(context.Background(), cp)
	if err != nil {
		log.Fatalln(err)
	}

	if ca.ReasonCode != 0 {
		log.Fatalf("Failed to connect to %s : %d - %s", mqttServer, ca.ReasonCode, ca.Properties.ReasonString)
	}

	log.Infof("MQTT: Connected to %s\n", mqttServer)

	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-ic
		log.Infof("signal received, exiting")
		if mqttClient != nil {
			d := &paho.Disconnect{ReasonCode: 0}
			mqttClient.Disconnect(d)
		}
		//os.Exit(0)
	}()

}

func publishData(batch []BatchEntry, value float64) error {

	jsonData := map[string]interface{}{}

	var v = value
	for _, field := range batch {

		switch field.Direction {
		case "in":
			v = math.Max(value, 0)
		case "out":
			v = math.Abs(math.Min(value, 0))
		default:
		}

		jsonData[removeSpace(field.Name)] = v
	}

	log.Debugf("MQTT: Publish: %v", jsonData)

	if payload, err := json.Marshal(jsonData); err == nil {

		if _, err := mqttClient.Publish(context.Background(), &paho.Publish{
			Topic:   statusTopic,
			QoS:     0,
			Retain:  false,
			Payload: payload,
		}); err != nil {
			log.Debugln("MQTT: error sending message:", err)
			return errors.New("MQTT: could not publish data or create payload")
		}
		log.Debugln("MQTT: successful send paylaod to: ", statusTopic)
		return nil
	}

	return errors.New("MQTT: could not publish data or create payload")
}

func publishDataRaw(jsonData map[string]interface{}) error {

	if len(jsonData) == 0 {
		return nil
	}

	log.Debugf("MQTT: Publish: %v", jsonData)

	if payload, err := json.Marshal(jsonData); err == nil {

		if _, err := mqttClient.Publish(context.Background(), &paho.Publish{
			Topic:   statusTopic,
			QoS:     0,
			Retain:  false,
			Payload: payload,
		}); err != nil {
			log.Debugln("MQTT: error sending message:", err)
			return errors.New("MQTT: could not publish data or create payload")
		}
		log.Debugln("MQTT: successful send paylaod to: ", statusTopic)
		return nil
	}

	return errors.New("MQTT: could not publish data or create payload")
}
