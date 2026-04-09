package models

import "time"

// DataPacket representa a telemetria enviada via UDP (Sensores)
type DataPacket struct {
	DeviceID  string    `json:"device_id"` // Ex: "RACK_01_TEMP"
	Type      string    `json:"type"`      // "SENSOR"
	Metric    string    `json:"metric"`    // "temperatura", "umidade", "energia"
	Value     float64   `json:"value"`     // O valor lido
	Unit      string    `json:"unit"`      // "°C", "%", "A"
	Timestamp time.Time `json:"timestamp"`
}

// CommandPacket representa ordens enviadas via TCP (Clientes -> Atuadores)
type CommandPacket struct {
	TargetID string `json:"target_id"` // ID do Atuador (Ex: "AC_RACK_01")
	Action   string `json:"action"`    // "LIGAR", "DESLIGAR", "SET_POWER"
	Value    int    `json:"value"`     // Parâmetro extra (ex: potência 0-100)
}
