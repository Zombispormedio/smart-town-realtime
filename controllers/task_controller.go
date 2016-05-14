package controllers

import (
	"errors"
	"strings"

	"os"

	"github.com/Zombispormedio/smart-push/config"
	"github.com/Zombispormedio/smart-push/lib/redis"
	"github.com/Zombispormedio/smart-push/lib/request"
	"github.com/Zombispormedio/smart-push/lib/response"
	"github.com/Zombispormedio/smart-push/lib/store"
)

func RefreshCredentials() error {
	var Error error
	hostname := os.Getenv("SENSOR_STORE_HOSTNAME")
	url := hostname + "push/credentials"

	msg := response.DataT{}

	RequestError := request.GetWithAuthorization(url, &msg)

	if RequestError != nil {
		return RequestError
	}

	if msg.Data == nil {
		return errors.New("No Authorized")
	}

	data := msg.Data.(map[string]interface{})

	StoringError := store.Put("identifier", data["key"].(string), "Config")

	if StoringError != nil {
		Error = StoringError
	}

	return Error
}

type PushSensorData struct {
	NodeID string `json:"node_id"`
	Value  string `json:"value"`
	Date   string `json:"date"`
}

type PushSensorGrid struct {
	ClientID string           `json:"client_id"`
	Data     []PushSensorData `json:"data"`
}

func PushOver() error {
	var Error error
	freq := config.PacketFrequency()

	Send := func(packet []PushSensorGrid) error {
		return SendSensorGridPacket(packet)
	}
	grids := []PushSensorGrid{}

	client := redis.Client()

	defer client.Close()

	gridKeys, Error := client.KeysGroup(os.Getenv("GRID_KEY"))

	if Error != nil {
		return Error
	}

	for _, gridkey := range gridKeys {

		if len(grids) == freq {
			SendError := Send(grids)
			if SendError != nil {
				Error = SendError
				break
			} else {
				grids = nil
			}
		}

		sensorKeys, SensorKeysError := client.SMembers(gridkey)

		if SensorKeysError != nil {
			Error = SensorKeysError
			break
		}

		elems := strings.Split(gridkey, ":")
		clientID := elems[len(elems)-1]

		grid := PushSensorGrid{}
		grid.ClientID = clientID

		for _, nodeID := range sensorKeys {
			sensorData := PushSensorData{}

			sensorData.NodeID = nodeID

			sensorKey := os.Getenv("SENSOR_KEY") + ":" + nodeID

			dataMap, SensorDataError := client.HGetAllMap(sensorKey)

			sensorData.Value = dataMap["value"]
			sensorData.Date = dataMap["date"]

			if SensorDataError != nil {
				Error = SensorDataError
				break
			}

			grid.Data = append(grid.Data, sensorData)

		}

		if Error != nil {
			break
		}

		grids = append(grids, grid)

	}

	if Error == nil && len(grids) > 0 {
		Error = Send(grids)
	}

	return Error
}

func SendSensorGridPacket(packet []PushSensorGrid) error {
	var Error error

	db, OpenDBError := store.OpenDB()

	if OpenDBError != nil {
		return OpenDBError
	}

	identifier, GetKeyError := store.GetWithDB(db, "Config", "identifier")

	if GetKeyError != nil {
		return GetKeyError
	}

	hostname := os.Getenv("SENSOR_STORE_HOSTNAME")
	url := hostname + "push/sensor_grid"
	headers := map[string]string{"Authorization": identifier}

	resBody := &response.MixedMessageT{}

	RequestError := request.PostWithHeaders(url, packet, headers, resBody)

	if RequestError != nil {
		return RequestError
	}

	if resBody.Status != 0 {
		Error = errors.New(resBody.Error)
	}

	db.Close()

	return Error
}

func Clean() error {
	var Error error

	client := redis.Client()

	defer client.Close()

	sensorKeys, SensorKeysError := client.KeysGroup(os.Getenv("Sensor_KEY"))

	if SensorKeysError != nil {
		return SensorKeysError
	}

	gridKeys, GridKeysError := client.KeysGroup(os.Getenv("GRID_KEY"))

	if GridKeysError != nil {
		return GridKeysError
	}

	for _, k := range sensorKeys {
		SensorCleanError := client.Del(k)

		if SensorCleanError != nil {
			Error = SensorCleanError
			break
		}
	}

	for _, k := range gridKeys {
		gridCleanError := client.Del(k)

		if gridCleanError != nil {
			Error = gridCleanError
			break
		}
	}

	return Error
}