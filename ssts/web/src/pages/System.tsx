import React from 'react';
import { Box, Typography, Grid, Card, CardContent, Chip } from '@mui/material';

const System: React.FC = () => {
  const systemInfo = {
    hostname: 'localhost',
    os: 'Linux',
    platform: 'Ubuntu 22.04',
    cpu_model: 'Intel Core i7-9700K',
    cpu_cores: 8,
    total_memory: '16 GB',
    go_version: 'go1.21.0',
  };

  const healthStatus = {
    overall: 'healthy',
    database: 'healthy',
    influxdb: 'healthy',
    plugins: 'healthy',
  };

  return (
    <Box>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" sx={{ mb: 1, fontWeight: 600 }}>
          System Information
        </Typography>
        <Typography variant="body2" color="text.secondary">
          System health and configuration details
        </Typography>
      </Box>

      <Grid container spacing={3}>
        {/* System Health */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" sx={{ mb: 3, fontWeight: 600 }}>
                System Health
              </Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography>Overall Status</Typography>
                  <Chip
                    label={healthStatus.overall}
                    color="success"
                    size="small"
                  />
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography>Database</Typography>
                  <Chip
                    label={healthStatus.database}
                    color="success"
                    size="small"
                  />
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography>InfluxDB</Typography>
                  <Chip
                    label={healthStatus.influxdb}
                    color="success"
                    size="small"
                  />
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography>Plugins</Typography>
                  <Chip
                    label={healthStatus.plugins}
                    color="success"
                    size="small"
                  />
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* System Information */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" sx={{ mb: 3, fontWeight: 600 }}>
                System Details
              </Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography color="text.secondary">Hostname</Typography>
                  <Typography>{systemInfo.hostname}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography color="text.secondary">OS</Typography>
                  <Typography>{systemInfo.os}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography color="text.secondary">Platform</Typography>
                  <Typography>{systemInfo.platform}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography color="text.secondary">CPU</Typography>
                  <Typography>{systemInfo.cpu_model}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography color="text.secondary">CPU Cores</Typography>
                  <Typography>{systemInfo.cpu_cores}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography color="text.secondary">Memory</Typography>
                  <Typography>{systemInfo.total_memory}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography color="text.secondary">Go Version</Typography>
                  <Typography>{systemInfo.go_version}</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default System;