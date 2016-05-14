package controllers

import (
	"reflect"
	"time"
	"github.com/Zombispormedio/smart-push/lib/redis"
	"github.com/Zombispormedio/smartdb/lib/struts"
)

type SensorData struct {
	NodeID string `json:"node_id"`
	Value  string `json:"value"`
}

func (sensorData *SensorData) FillByMap(Map map[string]interface{}, LiteralTag string) {
	struts.FillByMap(*sensorData, reflect.ValueOf(sensorData).Elem(), Map, LiteralTag)
}

type SensorGridData struct {
	Data []SensorData `json:"data"`
}

func (sensorGridData *SensorGridData) FillByMap(Map map[string]interface{}, LiteralTag string) {
	struts.FillByMap(*sensorGridData, reflect.ValueOf(sensorGridData).Elem(), Map, LiteralTag)
}

func ManageSensorData(sensorGridID string, data interface{}) error {
	var Error error
	grid := SensorGridData{}

	grid.FillByMap(data.(map[string]interface{}), "json")

	numNodes := len(grid.Data)

	client := redis.Client()

	gridKey := "grid:" + sensorGridID
	
	DelError:=client.Del(gridKey)
	
	if DelError != nil{
		return DelError
	}

	for i := 0; i < numNodes; i++ {
		sensor := grid.Data[i]

		SADDError:=client.SAdd(gridKey, sensor.NodeID)
		
		if SADDError != nil{
			return SADDError
		}
		
		nodeKey:="sensor:"+sensor.NodeID
		
		nodeMap:=map[string]string{
			"value":sensor.Value,
			"date":time.Now().String(),
		}
		HMSetMapError:=client.HMSetMap(nodeKey, nodeMap)
		
		if HMSetMapError != nil{
			return HMSetMapError
		}
	}

	Error = client.Close()

	return Error
}
