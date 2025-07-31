import React, { useState, useEffect } from 'react';
import {
  Box,
  Grid,
  Card,
  CardContent,
  Typography,
  LinearProgress,
  Chip,
  Button,
  Alert,
} from '@mui/material';
import {
  PlayArrow as PlayArrowIcon,
  Stop as StopIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

// Types
interface SystemMetrics {
  timestamp: string;
  cpu: {
    usage_percent: number;
    temperature_celsius: number;
  };
  memory: {
    usage_percent: number;
    total_bytes: number;
    used_bytes: number;
  };
  disk: {
    usage_percent: number;
  };
  network: {
    rx_bytes_per_sec: number;
    tx_bytes_per_sec: number;
  };
}

interface ActiveTest {
  id: string;
  name: string;
  plugin: string;
  status: string;
  start_time: string;
  duration: number;
}

// Mock data for demonstration
const mockSystemMetrics: SystemMetrics = {
  timestamp: new Date().toISOString(),
  cpu: {
    usage_percent: 45.2,
    temperature_celsius: 62.5,
  },
  memory: {
    usage_percent: 68.7,
    total_bytes: 16 * 1024 * 1024 * 1024, // 16GB
    used_bytes: 11 * 1024 * 1024 * 1024,  // 11GB
  },
  disk: {
    usage_percent: 34.8,
  },
  network: {
    rx_bytes_per_sec: 1024 * 1024 * 2.5, // 2.5 MB/s
    tx_bytes_per_sec: 1024 * 1024 * 1.2, // 1.2 MB/s
  },
};

const mockActiveTests: ActiveTest[] = [
  {
    id: '1',
    name: 'CPU Stress Test',
    plugin: 'cpu-stress',
    status: 'running',
    start_time: new Date(Date.now() - 300000).toISOString(), // 5 minutes ago
    duration: 600, // 10 minutes
  },
  {
    id: '2',
    name: 'Memory Load Test',
    plugin: 'memory-stress',
    status: 'pending',
    start_time: '',
    duration: 300, // 5 minutes
  },
];

// Generate mock historical data
const generateMockData = () => {
  const data = [];
  for (let i = 29; i >= 0; i--) {
    const timestamp = new Date(Date.now() - i * 60000);
    data.push({
      time: timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      cpu: Math.random() * 30 + 40, // 40-70%
      memory: Math.random() * 20 + 60, // 60-80%
      disk: Math.random() * 10 + 30, // 30-40%
    });
  }
  return data;
};

const Dashboard: React.FC = () => {
  const [systemMetrics, setSystemMetrics] = useState<SystemMetrics>(mockSystemMetrics);
  const [activeTests, setActiveTests] = useState<ActiveTest[]>(mockActiveTests);
  const [historicalData, setHistoricalData] = useState(generateMockData());
  const [websocket, setWebsocket] = useState<WebSocket | null>(null);

  useEffect(() => {
    // Initialize WebSocket connection
    const ws = new WebSocket(`ws://${window.location.host}/ws`);
    
    ws.onopen = () => {
      console.log('WebSocket connected');
      setWebsocket(ws);
    };

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        if (message.type === 'system_metrics') {
          setSystemMetrics(message.data);
        } else if (message.type === 'test_update') {
          // Update active tests based on test updates
          setActiveTests(prev => 
            prev.map(test => 
              test.id === message.data.test_id 
                ? { ...test, status: message.data.status }
                : test
            )
          );
        }
      } catch (error) {
        console.error('Error parsing WebSocket message:', error);
      }
    };

    ws.onclose = () => {
      console.log('WebSocket disconnected');
      setWebsocket(null);
    };

    // Cleanup on unmount
    return () => {
      ws.close();
    };
  }, []);

  // Update historical data periodically
  useEffect(() => {
    const interval = setInterval(() => {
      setHistoricalData(prev => {
        const newData = [...prev.slice(1)];
        const timestamp = new Date();
        newData.push({
          time: timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
          cpu: systemMetrics.cpu.usage_percent,
          memory: systemMetrics.memory.usage_percent,
          disk: systemMetrics.disk.usage_percent,
        });
        return newData;
      });
    }, 60000); // Update every minute

    return () => clearInterval(interval);
  }, [systemMetrics]);

  const formatBytes = (bytes: number) => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let unitIndex = 0;
    let size = bytes;
    
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    
    return `${size.toFixed(1)} ${units[unitIndex]}`;
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running': return 'success';
      case 'pending': return 'warning';
      case 'failed': return 'error';
      case 'completed': return 'info';
      default: return 'default';
    }
  };

  const handleStartTest = (testId: string) => {
    // TODO: Implement test start functionality
    console.log('Starting test:', testId);
  };

  const handleStopTest = (testId: string) => {
    // TODO: Implement test stop functionality
    console.log('Stopping test:', testId);
  };

  return (
    <Box>
      {/* Header */}
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" sx={{ mb: 1, fontWeight: 600 }}>
          Dashboard
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Real-time system monitoring and test management
        </Typography>
      </Box>

      {/* Connection Status */}
      {!websocket && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          WebSocket connection lost. Real-time updates may not work properly.
          <Button
            startIcon={<RefreshIcon />}
            onClick={() => window.location.reload()}
            sx={{ ml: 2 }}
          >
            Refresh
          </Button>
        </Alert>
      )}

      {/* System Metrics Cards */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="text.secondary" gutterBottom>
                CPU Usage
              </Typography>
              <Typography variant="h4" sx={{ mb: 2, fontWeight: 600 }}>
                {systemMetrics.cpu.usage_percent.toFixed(1)}%
              </Typography>
              <LinearProgress
                variant="determinate"
                value={systemMetrics.cpu.usage_percent}
                sx={{ mb: 1 }}
                color={systemMetrics.cpu.usage_percent > 80 ? 'error' : 'primary'}
              />
              <Typography variant="body2" color="text.secondary">
                Temperature: {systemMetrics.cpu.temperature_celsius.toFixed(1)}°C
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="text.secondary" gutterBottom>
                Memory Usage
              </Typography>
              <Typography variant="h4" sx={{ mb: 2, fontWeight: 600 }}>
                {systemMetrics.memory.usage_percent.toFixed(1)}%
              </Typography>
              <LinearProgress
                variant="determinate"
                value={systemMetrics.memory.usage_percent}
                sx={{ mb: 1 }}
                color={systemMetrics.memory.usage_percent > 80 ? 'error' : 'primary'}
              />
              <Typography variant="body2" color="text.secondary">
                {formatBytes(systemMetrics.memory.used_bytes)} / {formatBytes(systemMetrics.memory.total_bytes)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="text.secondary" gutterBottom>
                Disk Usage
              </Typography>
              <Typography variant="h4" sx={{ mb: 2, fontWeight: 600 }}>
                {systemMetrics.disk.usage_percent.toFixed(1)}%
              </Typography>
              <LinearProgress
                variant="determinate"
                value={systemMetrics.disk.usage_percent}
                sx={{ mb: 1 }}
                color={systemMetrics.disk.usage_percent > 80 ? 'error' : 'primary'}
              />
              <Typography variant="body2" color="text.secondary">
                Primary disk
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="text.secondary" gutterBottom>
                Network I/O
              </Typography>
              <Typography variant="h4" sx={{ mb: 2, fontWeight: 600 }}>
                {formatBytes(systemMetrics.network.rx_bytes_per_sec + systemMetrics.network.tx_bytes_per_sec)}/s
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 0.5 }}>
                ↓ {formatBytes(systemMetrics.network.rx_bytes_per_sec)}/s
              </Typography>
              <Typography variant="body2" color="text.secondary">
                ↑ {formatBytes(systemMetrics.network.tx_bytes_per_sec)}/s
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Charts and Active Tests */}
      <Grid container spacing={3}>
        {/* Historical Performance Chart */}
        <Grid item xs={12} lg={8}>
          <Card>
            <CardContent>
              <Typography variant="h6" sx={{ mb: 3, fontWeight: 600 }}>
                System Performance (Last 30 minutes)
              </Typography>
              <Box sx={{ width: '100%', height: 300 }}>
                <ResponsiveContainer>
                  <LineChart data={historicalData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="time" />
                    <YAxis domain={[0, 100]} />
                    <Tooltip formatter={(value: any) => [`${value.toFixed(1)}%`, '']} />
                    <Line
                      type="monotone"
                      dataKey="cpu"
                      stroke="#2563EB"
                      strokeWidth={2}
                      dot={false}
                      name="CPU"
                    />
                    <Line
                      type="monotone"
                      dataKey="memory"
                      stroke="#7C3AED"
                      strokeWidth={2}
                      dot={false}
                      name="Memory"
                    />
                    <Line
                      type="monotone"
                      dataKey="disk"
                      stroke="#059669"
                      strokeWidth={2}
                      dot={false}
                      name="Disk"
                    />
                  </LineChart>
                </ResponsiveContainer>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Active Tests */}
        <Grid item xs={12} lg={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" sx={{ mb: 3, fontWeight: 600 }}>
                Active Tests
              </Typography>
              {activeTests.length === 0 ? (
                <Typography variant="body2" color="text.secondary">
                  No active tests
                </Typography>
              ) : (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  {activeTests.map((test) => (
                    <Box
                      key={test.id}
                      sx={{
                        p: 2,
                        border: 1,
                        borderColor: 'divider',
                        borderRadius: 2,
                        backgroundColor: 'background.paper',
                      }}
                    >
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1 }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                          {test.name}
                        </Typography>
                        <Chip
                          label={test.status}
                          size="small"
                          color={getStatusColor(test.status) as any}
                          variant="filled"
                        />
                      </Box>
                      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                        Plugin: {test.plugin}
                      </Typography>
                      <Box sx={{ display: 'flex', gap: 1 }}>
                        {test.status === 'running' ? (
                          <Button
                            size="small"
                            variant="outlined"
                            startIcon={<StopIcon />}
                            onClick={() => handleStopTest(test.id)}
                            color="error"
                          >
                            Stop
                          </Button>
                        ) : (
                          <Button
                            size="small"
                            variant="contained"
                            startIcon={<PlayArrowIcon />}
                            onClick={() => handleStartTest(test.id)}
                          >
                            Start
                          </Button>
                        )}
                      </Box>
                    </Box>
                  ))}
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default Dashboard;