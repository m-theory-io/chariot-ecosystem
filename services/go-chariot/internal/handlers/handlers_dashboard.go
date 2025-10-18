package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// DashboardData represents the data shown on the dashboard
type DashboardData struct {
	ServerStatus   ServerStatus      `json:"server_status"`
	SessionStats   SessionStats      `json:"session_stats"`
	SystemMetrics  SystemMetrics     `json:"system_metrics"`
	Configuration  ConfigurationInfo `json:"configuration"`
	ActiveSessions []SessionInfo     `json:"active_sessions"`
	Listeners      []ListenerInfo    `json:"listeners"`
}

type ServerStatus struct {
	Status    string    `json:"status"`
	Uptime    string    `json:"uptime"`
	StartTime time.Time `json:"start_time"`
	Port      int       `json:"port"`
	SSL       bool      `json:"ssl"`
	Mode      string    `json:"mode"`
}

type SessionStats struct {
	ActiveCount int `json:"active_count"`
}

type SessionInfo struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	Created    time.Time `json:"created"`
	LastSeen   time.Time `json:"last_seen"`
	LastAccess time.Time `json:"last_access"`
	ExpiresAt  time.Time `json:"expires_at"`
	Status     string    `json:"status"`
}

type ListenerInfo struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	StartTime  time.Time `json:"start_time"`
	Script     string    `json:"script"`
	LastActive time.Time `json:"last_active"`
	IsHealthy  bool      `json:"is_healthy"`
}

type SystemMetrics struct {
	Memory     MemoryStats `json:"memory"`
	Goroutines int         `json:"goroutines"`
	CPUCount   int         `json:"cpu_count"`
	Version    string      `json:"version"`
}

type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

type ConfigurationInfo struct {
	DataPath    string `json:"data_path"`
	TreePath    string `json:"tree_path"`
	DiagramPath string `json:"diagram_path"`
	TreeFormat  string `json:"tree_format"`
	Timeout     int    `json:"timeout"`
	VaultName   string `json:"vault_name"`
	SQLDriver   string `json:"sql_driver"`
	CBBucket    string `json:"cb_bucket"`
}

// HandleDashboard serves the dashboard HTML page
func (h *Handlers) HandleDashboard(c echo.Context) error {
	dashboardHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chariot Dashboard</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; text-align: center; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .card { background: white; border-radius: 8px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .status-good { color: #22c55e; font-weight: bold; }
        .status-warning { color: #f59e0b; font-weight: bold; }
        .status-error { color: #ef4444; font-weight: bold; }
        .metric { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #e5e7eb; }
        .metric:last-child { border-bottom: none; }
        .refresh-btn { background: #3b82f6; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; margin-left: 10px; }
        .refresh-btn:hover { background: #2563eb; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { text-align: left; padding: 8px; border-bottom: 1px solid #e5e7eb; }
        th { background: #f9fafb; font-weight: 600; }
        .timestamp { color: #6b7280; font-size: 0.875rem; margin-left: 20px; }
        h3 { margin-top: 0; color: #374151; }
        .loading { text-align: center; color: #6b7280; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚀 Chariot Dashboard</h1>
            <p>Real-time monitoring and status information</p>
            <button class="refresh-btn" onclick="refreshData()">🔄 Refresh</button>
            <span class="timestamp" id="lastUpdate"></span>
        </div>
        
        <div class="grid">
            <div class="card">
                <h3>📊 Server Status</h3>
                <div id="serverStatus" class="loading">Loading...</div>
            </div>
            
            <div class="card">
                <h3>👥 Active Sessions</h3>
                <div id="sessions" class="loading">Loading...</div>
            </div>
            
            <div class="card">
				<h3>Listeners</h3>
                <div id="listeners" class="loading">Loading...</div>
            </div>
            
            <div class="card">
                <h3>💾 System Metrics</h3>
                <div id="metrics" class="loading">Loading...</div>
            </div>
            
            <div class="card">
                <h3>⚙️ Configuration</h3>
                <div id="configuration" class="loading">Loading...</div>
            </div>
        </div>
    </div>

    <script>
        function refreshData() {
            document.getElementById('lastUpdate').textContent = 'Updating...';
            
            fetch('/api/dashboard/status')
                .then(response => {
                    if (!response.ok) {
                        throw new Error('Network response was not ok');
                    }
                    return response.json();
                })
                .then(data => {
                    updateServerStatus(data.server_status);
                    updateSessions(data.session_stats, data.active_sessions);
                    updateListeners(data.listeners);
                    updateMetrics(data.system_metrics);
                    updateConfiguration(data.configuration);
                    document.getElementById('lastUpdate').textContent = 'Last updated: ' + new Date().toLocaleTimeString();
                })
                .catch(error => {
                    console.error('Error fetching data:', error);
                    document.getElementById('lastUpdate').textContent = 'Update failed: ' + new Date().toLocaleTimeString();
                    // Show error in each section
                    ['serverStatus', 'sessions', 'listeners', 'metrics', 'configuration'].forEach(id => {
                        document.getElementById(id).innerHTML = '<span class="status-error">Failed to load data</span>';
                    });
                });
        }
        
        function updateServerStatus(status) {
            const statusClass = status.status === 'running' ? 'status-good' : 'status-error';
            document.getElementById('serverStatus').innerHTML = ` + "`" + `
                <div class="metric"><span>Status:</span><span class="${statusClass}">●&#160;${status.status.toUpperCase()}</span></div>
                <div class="metric"><span>Uptime:</span><span>${status.uptime}</span></div>
                <div class="metric"><span>Port:</span><span>${status.port}</span></div>
                <div class="metric"><span>SSL:</span><span>${status.ssl ? '🔒 Enabled' : '🔓 Disabled'}</span></div>
                <div class="metric"><span>Mode:</span><span>${status.mode}</span></div>
            ` + "`" + `;
        }
        
        function updateSessions(stats, sessions) {
            let html = ` + "`" + `<div class="metric"><span>Active Sessions:</span><span class="status-good">${stats.active_count}</span></div>` + "`" + `;
            
            if (sessions && sessions.length > 0) {
                html += '<table><tr><th>User ID</th><th>Created</th><th>Status</th></tr>';
                sessions.forEach(session => {
                    const statusClass = session.status === 'active' ? 'status-good' : 'status-warning';
                    html += ` + "`" + `<tr><td>${session.user_id}</td><td>${new Date(session.created).toLocaleString()}</td><td class="${statusClass}">${session.status}</td></tr>` + "`" + `;
                });
                html += '</table>';
            } else if (stats.active_count === 0) {
                html += '<p style="color: #6b7280; margin-top: 10px;">No active sessions</p>';
            }
            
            document.getElementById('sessions').innerHTML = html;
        }
        
        function updateListeners(listeners) {
            if (!listeners || listeners.length === 0) {
                document.getElementById('listeners').innerHTML = '<p style="color: #6b7280;">No listeners configured</p>';
                return;
            }
            
            let html = '<table><tr><th>Name</th><th>Status</th><th>Script</th><th>Health</th></tr>';
            listeners.forEach(listener => {
                const statusClass = listener.status === 'running' ? 'status-good' : 'status-error';
                const healthClass = listener.is_healthy ? 'status-good' : 'status-error';
                const healthIcon = listener.is_healthy ? '✅' : '❌';
                html += ` + "`" + `<tr><td>${listener.name}</td><td class="${statusClass}">${listener.status}</td><td>${listener.script || 'N/A'}</td><td class="${healthClass}">${healthIcon}&#160;${listener.is_healthy ? 'Healthy' : 'Unhealthy'}</td></tr>` + "`" + `;
            });
            html += '</table>';
            document.getElementById('listeners').innerHTML = html;
        }
        
        function updateMetrics(metrics) {
            document.getElementById('metrics').innerHTML = ` + "`" + `
                <div class="metric"><span>Memory (Alloc):</span><span>${(metrics.memory.alloc / 1024 / 1024).toFixed(2)} MB</span></div>
                <div class="metric"><span>Memory (Sys):</span><span>${(metrics.memory.sys / 1024 / 1024).toFixed(2)} MB</span></div>
                <div class="metric"><span>Goroutines:</span><span>${metrics.goroutines}</span></div>
                <div class="metric"><span>GC Runs:</span><span>${metrics.memory.num_gc}</span></div>
                <div class="metric"><span>CPU Cores:</span><span>${metrics.cpu_count}</span></div>
                <div class="metric"><span>Go Version:</span><span>${metrics.version}</span></div>
            ` + "`" + `;
        }
        
        function updateConfiguration(config) {
            document.getElementById('configuration').innerHTML = ` + "`" + `
                <div class="metric"><span>Data Path:</span><span>${config.data_path}</span></div>
                <div class="metric"><span>Tree Path:</span><span>${config.tree_path}</span></div>
				<div class="metric"><span>Diagram Path:</span><span>${config.diagram_path}</span></div>
                <div class="metric"><span>Tree Format:</span><span>${config.tree_format}</span></div>
                <div class="metric"><span>Session Timeout:</span><span>${config.timeout} min</span></div>
                <div class="metric"><span>Vault:</span><span>${config.vault_name}</span></div>
                <div class="metric"><span>SQL Driver:</span><span>${config.sql_driver}</span></div>
            ` + "`" + `;
        }
        
        // Initial load and auto-refresh every 30 seconds
        refreshData();
        setInterval(refreshData, 30000);
    </script>
</body>
</html>`

	return c.HTML(http.StatusOK, dashboardHTML)
}

// HandleDashboardAPI provides JSON data for the dashboard
func (h *Handlers) HandleDashboardAPI(c echo.Context) error {
	data := h.collectDashboardData()
	return c.JSON(http.StatusOK, data)
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow same-origin and reverse-proxy scenarios; rely on token for auth
		return true
	},
}

// HandleDashboardWS upgrades to a WebSocket and streams dashboard data periodically.
// Auth: requires an Authorization header with a valid session token for the initial upgrade.
// After upgrade, the connection stays alive regardless of session TTL to keep the dashboard visible.
func (h *Handlers) HandleDashboardWS(c echo.Context) error {
	cfg.ChariotLogger.Info("WS connection attempt", zap.String("remote_addr", c.Request().RemoteAddr))
	// Perform a non-extending auth check
	token := c.Request().Header.Get("Authorization")
	if token == "" {
		cfg.ChariotLogger.Warn("WS upgrade rejected: missing token")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authorization required"})
	}
	if _, ok := h.sessionManager.LookupSession(token); !ok {
		cfg.ChariotLogger.Warn("WS upgrade rejected: invalid/expired token", zap.String("token", token))
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired session"})
	}

	// Upgrade to WebSocket
	conn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		cfg.ChariotLogger.Error("WS upgrade failed", zap.Error(err))
		return err
	}
	cfg.ChariotLogger.Info("WS connected for dashboard", zap.String("token", token))
	defer conn.Close()

	// Writer loop: send dashboard data every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Optional: read loop to allow client pings or close
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Launch a goroutine to read and discard messages (to process pings/close frames)
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				cfg.ChariotLogger.Info("WS read loop terminated", zap.Error(err))
				return
			}
		}
	}()

	for range ticker.C {
		data := h.collectDashboardData()
		payload, _ := json.Marshal(ResultJSON{Result: "OK", Data: data})
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			cfg.ChariotLogger.Warn("WS write failed; closing stream", zap.Time("at", time.Now()), zap.Error(err))
			return nil
		}
	}
	return nil
}

func (h *Handlers) collectDashboardData() DashboardData {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get session information
	activeSessionCount := h.sessionManager.GetActiveSessions()
	sessionData := h.sessionManager.GetActiveSessionsInfo()

	// Convert to SessionInfo structs with safe type handling
	var activeSessions []SessionInfo
	for _, m := range sessionData {
		var si SessionInfo
		if v, ok := m["id"].(string); ok {
			si.ID = v
		}
		if v, ok := m["session_id"].(string); ok {
			si.SessionID = v
		} else {
			si.SessionID = si.ID
		}
		if v, ok := m["user_id"].(string); ok {
			si.UserID = v
		}
		if v, ok := m["username"].(string); ok {
			si.Username = v
		}
		if v, ok := m["created"].(time.Time); ok {
			si.Created = v
		}
		if v, ok := m["last_seen"].(time.Time); ok {
			si.LastSeen = v
		}
		if v, ok := m["last_access"].(time.Time); ok {
			si.LastAccess = v
		} else {
			si.LastAccess = si.LastSeen
		}
		if v, ok := m["expires_at"].(time.Time); ok {
			si.ExpiresAt = v
		}
		if v, ok := m["status"].(string); ok {
			si.Status = v
		} else {
			if !si.ExpiresAt.IsZero() && si.ExpiresAt.After(time.Now()) {
				si.Status = "active"
			} else if !si.ExpiresAt.IsZero() {
				si.Status = "expired"
			} else {
				si.Status = "active"
			}
		}
		activeSessions = append(activeSessions, si)
	}

	// Pull listeners from registry
	var lInfos []ListenerInfo
	if h.listenerManager != nil {
		for _, l := range h.listenerManager.List() {
			lInfos = append(lInfos, ListenerInfo{
				Name:       l.Name,
				Status:     l.Status,
				StartTime:  l.StartTime,
				Script:     l.Script,
				LastActive: l.LastActive,
				IsHealthy:  l.IsHealthy,
			})
		}
	}

	return DashboardData{
		ServerStatus: ServerStatus{
			Status:    "running",
			Uptime:    time.Since(h.startTime).String(),
			StartTime: h.startTime,
			Port:      cfg.ChariotConfig.Port,
			SSL:       cfg.ChariotConfig.SSL,
			Mode: func() string {
				if cfg.ChariotConfig.Headless {
					return "headless"
				}
				return "api"
			}(),
		},
		SessionStats: SessionStats{
			ActiveCount: activeSessionCount,
		},
		SystemMetrics: SystemMetrics{
			Memory: MemoryStats{
				Alloc:      memStats.Alloc,
				TotalAlloc: memStats.TotalAlloc,
				Sys:        memStats.Sys,
				NumGC:      memStats.NumGC,
			},
			Goroutines: runtime.NumGoroutine(),
			CPUCount:   runtime.NumCPU(),
			Version:    runtime.Version(),
		},
		Configuration: ConfigurationInfo{
			DataPath:    cfg.ChariotConfig.DataPath,
			TreePath:    cfg.ChariotConfig.TreePath,
			DiagramPath: cfg.ChariotConfig.DiagramPath,
			TreeFormat:  cfg.ChariotConfig.TreeFormat,
			Timeout:     cfg.ChariotConfig.Timeout,
			VaultName:   cfg.ChariotConfig.VaultName,
			SQLDriver:   cfg.ChariotConfig.SQLDriver,
			CBBucket:    cfg.ChariotConfig.CBBucket,
		},
		ActiveSessions: activeSessions,
		Listeners:      lInfos,
	}
}
