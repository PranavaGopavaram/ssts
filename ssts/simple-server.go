package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type SystemMetrics struct {
	Timestamp time.Time `json:"timestamp"`
	CPUUsage  float64   `json:"cpu_usage"`
	Memory    float64   `json:"memory"`
	Disk      float64   `json:"disk"`
}

type TestExecution struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
}

var executions = []TestExecution{
	{ID: "1", Name: "CPU Stress Test", Status: "completed", StartTime: time.Now().Add(-time.Hour)},
	{ID: "2", Name: "Memory Test", Status: "running", StartTime: time.Now().Add(-time.Minute * 30)},
	{ID: "3", Name: "I/O Test", Status: "pending", StartTime: time.Now()},
}

const dashboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SSTS - System Stress Testing Suite</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            color: #333;
        }
        .container { 
            max-width: 1200px; 
            margin: 0 auto; 
            padding: 20px;
        }
        .header {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 30px;
            margin-bottom: 30px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }
        .header h1 {
            font-size: 2.5rem;
            font-weight: 700;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-bottom: 10px;
        }
        .header p {
            font-size: 1.1rem;
            color: #666;
        }
        .dashboard {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 30px;
        }
        .card {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 25px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
            transition: transform 0.3s ease;
        }
        .card:hover {
            transform: translateY(-5px);
        }
        .card h3 {
            font-size: 1.3rem;
            margin-bottom: 20px;
            color: #333;
        }
        .metric {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 10px 0;
            border-bottom: 1px solid #eee;
        }
        .metric:last-child {
            border-bottom: none;
        }
        .metric-value {
            font-weight: 600;
            color: #667eea;
        }
        .status {
            padding: 5px 12px;
            border-radius: 20px;
            font-size: 0.8rem;
            font-weight: 500;
        }
        .status.completed { background: #d4edda; color: #155724; }
        .status.running { background: #fff3cd; color: #856404; }
        .status.pending { background: #f8d7da; color: #721c24; }
        .test-item {
            padding: 15px 0;
            border-bottom: 1px solid #eee;
        }
        .test-item:last-child {
            border-bottom: none;
        }
        .test-name {
            font-weight: 600;
            margin-bottom: 5px;
        }
        .test-time {
            font-size: 0.9rem;
            color: #666;
        }
        .connection-status {
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 10px 20px;
            border-radius: 25px;
            font-weight: 500;
            font-size: 0.9rem;
            background: #28a745;
            color: white;
        }
        .connection-status.disconnected {
            background: #dc3545;
        }
    </style>
</head>
<body>
    <div class="connection-status" id="connectionStatus">● Connected</div>
    
    <div class="container">
        <div class="header">
            <h1>System Stress Testing Suite</h1>
            <p>Real-time monitoring and stress testing dashboard</p>
        </div>
        
        <div class="dashboard">
            <div class="card">
                <h3>📊 System Metrics</h3>
                <div class="metric">
                    <span>CPU Usage</span>
                    <span class="metric-value" id="cpuUsage">0%</span>
                </div>
                <div class="metric">
                    <span>Memory Usage</span>
                    <span class="metric-value" id="memoryUsage">0%</span>
                </div>
                <div class="metric">
                    <span>Disk Usage</span>
                    <span class="metric-value" id="diskUsage">0%</span>
                </div>
                <div class="metric">
                    <span>Last Updated</span>
                    <span class="metric-value" id="lastUpdated">Never</span>
                </div>
            </div>
            
            <div class="card">
                <h3>🧪 Test Executions</h3>
                <div id="testExecutions">
                    {{range .Executions}}
                    <div class="test-item">
                        <div class="test-name">{{.Name}}</div>
                        <div class="test-time">{{.StartTime.Format "Jan 2, 15:04:05"}}</div>
                        <span class="status {{.Status}}">{{.Status}}</span>
                    </div>
                    {{end}}
                </div>
            </div>
            
            <div class="card">
                <h3>⚡ Quick Actions</h3>
                <div style="display: flex; flex-direction: column; gap: 15px;">
                    <button onclick="startCPUTest()" style="padding: 12px; border: none; border-radius: 8px; background: #667eea; color: white; font-weight: 500; cursor: pointer;">Start CPU Test</button>
                    <button onclick="startMemoryTest()" style="padding: 12px; border: none; border-radius: 8px; background: #764ba2; color: white; font-weight: 500; cursor: pointer;">Start Memory Test</button>
                    <button onclick="stopAllTests()" style="padding: 12px; border: none; border-radius: 8px; background: #dc3545; color: white; font-weight: 500; cursor: pointer;">Stop All Tests</button>
                </div>
            </div>
            
            <div class="card">
                <h3>📈 Performance Stats</h3>
                <div class="metric">
                    <span>Tests Completed</span>
                    <span class="metric-value">15</span>
                </div>
                <div class="metric">
                    <span>Tests Running</span>
                    <span class="metric-value">1</span>
                </div>
                <div class="metric">
                    <span>System Uptime</span>
                    <span class="metric-value">2h 34m</span>
                </div>
                <div class="metric">
                    <span>Average CPU</span>
                    <span class="metric-value">23.5%</span>
                </div>
            </div>
        </div>
    </div>

    <script>
        let ws;
        let reconnectInterval;

        function connectWebSocket() {
            ws = new WebSocket('ws://localhost:8080/ws');
            
            ws.onopen = function() {
                document.getElementById('connectionStatus').textContent = '● Connected';
                document.getElementById('connectionStatus').className = 'connection-status';
                clearInterval(reconnectInterval);
            };
            
            ws.onmessage = function(event) {
                const data = JSON.parse(event.data);
                if (data.type === 'metrics') {
                    updateMetrics(data.data);
                }
            };
            
            ws.onclose = function() {
                document.getElementById('connectionStatus').textContent = '● Disconnected';
                document.getElementById('connectionStatus').className = 'connection-status disconnected';
                
                // Reconnect after 3 seconds
                reconnectInterval = setInterval(connectWebSocket, 3000);
            };
            
            ws.onerror = function(error) {
                console.log('WebSocket error:', error);
            };
        }

        function updateMetrics(metrics) {
            document.getElementById('cpuUsage').textContent = metrics.cpu_usage.toFixed(1) + '%';
            document.getElementById('memoryUsage').textContent = metrics.memory.toFixed(1) + '%';
            document.getElementById('diskUsage').textContent = metrics.disk.toFixed(1) + '%';
            document.getElementById('lastUpdated').textContent = new Date().toLocaleTimeString();
        }

        function startCPUTest() {
            fetch('/api/tests/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ type: 'cpu', duration: 60 })
            }).then(response => response.json())
              .then(data => alert('CPU test started: ' + data.message));
        }

        function startMemoryTest() {
            fetch('/api/tests/start', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ type: 'memory', duration: 60 })
            }).then(response => response.json())
              .then(data => alert('Memory test started: ' + data.message));
        }

        function stopAllTests() {
            fetch('/api/tests/stop-all', { method: 'POST' })
              .then(response => response.json())
              .then(data => alert('All tests stopped: ' + data.message));
        }

        // Connect WebSocket on page load
        connectWebSocket();
        
        // Generate fake metrics for demo
        setInterval(() => {
            if (ws && ws.readyState === WebSocket.OPEN) {
                const fakeMetrics = {
                    type: 'metrics',
                    data: {
                        cpu_usage: Math.random() * 100,
                        memory: Math.random() * 80 + 20,
                        disk: Math.random() * 60 + 30,
                        timestamp: new Date()
                    }
                };
                updateMetrics(fakeMetrics.data);
            }
        }, 2000);
    </script>
</body>
</html>
`

func main() {
	r := mux.NewRouter()

	// Serve the dashboard
	r.HandleFunc("/", dashboardHandler)
	
	// API endpoints
	r.HandleFunc("/api/metrics", metricsHandler).Methods("GET")
	r.HandleFunc("/api/executions", executionsHandler).Methods("GET")
	r.HandleFunc("/api/tests/start", startTestHandler).Methods("POST")
	r.HandleFunc("/api/tests/stop-all", stopTestsHandler).Methods("POST")
	r.HandleFunc("/health", healthHandler).Methods("GET")
	
	// WebSocket endpoint
	r.HandleFunc("/ws", websocketHandler)

	fmt.Println("🚀 SSTS Server starting on http://localhost:8080")
	fmt.Println("📊 Dashboard: http://localhost:8080")
	fmt.Println("❤️  Health Check: http://localhost:8080/health")
	
	log.Fatal(http.ListenAndServe(":8080", r))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))
	data := struct {
		Executions []TestExecution
	}{
		Executions: executions,
	}
	tmpl.Execute(w, data)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	metrics := SystemMetrics{
		Timestamp: time.Now(),
		CPUUsage:  float64(time.Now().Unix()%100) / 2,
		Memory:    float64(time.Now().Unix()%80) + 20,
		Disk:      float64(time.Now().Unix()%60) + 30,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func executionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

func startTestHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type     string `json:"type"`
		Duration int    `json:"duration"`
	}
	
	json.NewDecoder(r.Body).Decode(&req)
	
	response := map[string]string{
		"message": fmt.Sprintf("%s test started for %d seconds", req.Type, req.Duration),
		"status":  "success",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func stopTestsHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "All tests stopped successfully",
		"status":  "success",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",
		"uptime":    "2h 34m 15s",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Send metrics every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := SystemMetrics{
				Timestamp: time.Now(),
				CPUUsage:  float64(time.Now().Unix()%100) / 2,
				Memory:    float64(time.Now().Unix()%80) + 20,
				Disk:      float64(time.Now().Unix()%60) + 30,
			}
			
			message := map[string]interface{}{
				"type": "metrics",
				"data": metrics,
			}
			
			if err := conn.WriteJSON(message); err != nil {
				log.Println("WebSocket write error:", err)
				return
			}
		}
	}
}