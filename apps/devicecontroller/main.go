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
	   go func() {
		   time.Sleep(5 * time.Second)
		   fetchDevices()
	   }()

	   go telemetryBroadcaster()

	   // HTTP API на 8082
	   go func() {
		   r := mux.NewRouter()
		   r.HandleFunc("/devices/{deviceId}/command", handleCommand).Methods("POST")
		   r.HandleFunc("/devices/{deviceId}/telemetry", handleTelemetry).Methods("GET")
		   srv := &http.Server{
			   Addr: ":8082",
			   Handler: r,
		   }
		   log.Println("DeviceController HTTP API started on :8082")
		   if err := srv.ListenAndServe(); err != nil {
			   log.Fatalf("HTTP API server error: %v", err)
		   }
	   }()

	   // WebSocket сервер на 41200
	   wsRouter := mux.NewRouter()
	   wsRouter.HandleFunc("/ws", wsHandler)
	   wsSrv := &http.Server{
		   Addr: ":41200",
		   Handler: wsRouter,
	   }
	   log.Println("DeviceController WebSocket started on :41200/ws")
	   if err := wsSrv.ListenAndServe(); err != nil {
		   log.Fatalf("WebSocket server error: %v", err)
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
			// ГГенерация телеметрии в веб сокет
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
	   // Формат времени без таймзоны (YYYY-MM-DDTHH:MM:SS)
	   ts := time.Now().Format("2006-01-02T15:04:05")
	   return Telemetry{
		   DeviceID:  d.ID,
		   Timestamp: ts,
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
