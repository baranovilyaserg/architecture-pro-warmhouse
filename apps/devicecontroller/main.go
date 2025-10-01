package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type Device struct {
	ID     int    `json:"id"`
	Type   string `json:"type"`
	Model  string `json:"model"`
	Status string `json:"status"`
	HomeID int    `json:"home_id"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
}

type Telemetry struct {
	DeviceID  int    `json:"device_id"`
	Timestamp string `json:"timestamp"`
	Parameter string `json:"parameter"`
	Value     string `json:"value"`
}

var devices []Device
var wsClients = make(map[*websocket.Conn]bool)
var wsUpgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func main() {
	// 1. Получить список девайсов с devicecomponent API
	fetchDevices()

	// 2. Запустить генерацию телеметрии в WebSocket
	go telemetryBroadcaster()

	r := mux.NewRouter()
	r.HandleFunc("/devices/{deviceId}/command", handleCommand).Methods("POST")
	r.HandleFunc("/devices/{deviceId}/telemetry", handleTelemetry).Methods("GET")
	r.HandleFunc("/ws", wsHandler)

	srv := &http.Server{
		Addr: ":8082",
		Handler: r,
	}
	log.Println("DeviceController service started on :8082")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func fetchDevices() {
	url := os.Getenv("DEVICE_API_URL")
	if url == "" {
		url = "http://devicecomponent:8083/devices"
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch devices: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("Device API returned status: %d", resp.StatusCode)
		return
	}
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		log.Printf("Failed to decode devices: %v", err)
	}
	log.Printf("Fetched %d devices", len(devices))
}

func telemetryBroadcaster() {
	for {
		for _, d := range devices {
			// Здесь можно эмулировать обращение к d.IP:d.Port
			telemetry := generateTelemetry(d)
			broadcastTelemetry(telemetry)
		}
		time.Sleep(5 * time.Second)
	}
}

func generateTelemetry(d Device) Telemetry {
	var value string
	switch d.Type {
	case "bulb":
		if rand.Intn(2) == 0 {
			value = "on"
		} else {
			value = "off"
		}
	case "gates":
		if rand.Intn(2) == 0 {
			value = "open"
		} else {
			value = "closed"
		}
	default:
		value = "unknown"
	}
	return Telemetry{
		DeviceID:  d.ID,
		Timestamp: time.Now().Format(time.RFC3339),
		Parameter: d.Type,
		Value:     value,
	}
}

func broadcastTelemetry(t Telemetry) {
	msg, _ := json.Marshal(t)
	for client := range wsClients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	wsClients[conn] = true
	log.Println("WebSocket client connected")
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceId := vars["deviceId"]
	var body map[string]interface{}
	json.NewDecoder(r.Body).Decode(&body)
	log.Printf("Received command for device %s: %+v", deviceId, body)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"status\":\"ok\"}"))
}

func handleTelemetry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceId := vars["deviceId"]
	for _, d := range devices {
		if fmt.Sprintf("%d", d.ID) == deviceId {
			telemetry := generateTelemetry(d)
			json.NewEncoder(w).Encode(telemetry)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("{\"error\":\"device not found\"}"))
}
