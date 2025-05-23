package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type SystemMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage struct {
		Total     string  `json:"total"`
		Used      string  `json:"used"`
		Free      string  `json:"free"`
		UsedPercent float64 `json:"used_percent"`
	} `json:"memory_usage"`
	Timestamp time.Time `json:"timestamp"`
}

// MetricsManager handles shared metrics collection and distribution
type MetricsManager struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan *SystemMetrics
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func NewMetricsManager() *MetricsManager {
	return &MetricsManager{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan *SystemMetrics),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (m *MetricsManager) Run() {
	ticker := time.NewTicker(2 * time.Second) // Reduced frequency to 2 seconds
	defer ticker.Stop()

	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client] = true
			m.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(m.clients))

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client]; ok {
				delete(m.clients, client)
				client.Close()
			}
			m.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(m.clients))

		case <-ticker.C:
			metrics, err := getMetrics()
			if err != nil {
				log.Printf("Error getting metrics: %v", err)
				continue
			}
			m.broadcast <- metrics

		case metrics := <-m.broadcast:
			m.mu.RLock()
			for client := range m.clients {
				err := client.WriteJSON(metrics)
				if err != nil {
					log.Printf("Error writing to client: %v", err)
					client.Close()
					m.unregister <- client
				}
			}
			m.mu.RUnlock()
		}
	}
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func roundFloat(val float64, precision int) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func getMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// Get CPU usage with a shorter interval
	cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil {
		return nil, err
	}
	if len(cpuPercent) > 0 {
		metrics.CPUUsage = roundFloat(cpuPercent[0], 2)
	}

	// Get memory usage
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	metrics.MemoryUsage.Total = formatBytes(memInfo.Total)
	metrics.MemoryUsage.Used = formatBytes(memInfo.Used)
	metrics.MemoryUsage.Free = formatBytes(memInfo.Free)
	metrics.MemoryUsage.UsedPercent = roundFloat(memInfo.UsedPercent, 2)

	return metrics, nil
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics, err := getMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (m *MetricsManager) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	m.register <- conn

	// Set read deadline to detect stale connections
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping-pong to keep connection alive
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		defer func() {
			m.unregister <- conn
		}()

		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					return
				}
			}
		}
	}()
}

func main() {
	metricsManager := NewMetricsManager()
	go metricsManager.Run()

	// Static file server for the HTML page
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	// // API endpoints
	http.HandleFunc("/metrics", metricsHandler)
	// http.HandleFunc("/ws", metricsManager.wsHandler)

	port := ":8080"
	log.Printf("Server starting on port %s", port)
	log.Printf("Access metrics at http://localhost%s", port)
	
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
} 