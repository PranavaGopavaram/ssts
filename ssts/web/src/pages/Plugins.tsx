import React from 'react';
import { Box, Typography, Grid, Card, CardContent, Chip } from '@mui/material';

const Plugins: React.FC = () => {
  const plugins = [
    {
      name: 'cpu-stress',
      version: '1.0.0',
      description: 'CPU stress testing plugin with multiple algorithms',
      enabled: true,
    },
    {
      name: 'memory-stress',
      version: '1.0.0',
      description: 'Memory allocation and access pattern stress testing',
      enabled: true,
    },
    {
      name: 'io-stress',
      version: '1.0.0',
      description: 'Disk I/O performance stress testing',
      enabled: true,
    },
  ];

  return (
    <Box>
      <Box sx={{ mb: 4 }}>
        <Typography variant="h4" sx={{ mb: 1, fontWeight: 600 }}>
          Plugins
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Available stress testing plugins
        </Typography>
      </Box>

      <Grid container spacing={3}>
        {plugins.map((plugin) => (
          <Grid item xs={12} md={6} lg={4} key={plugin.name}>
            <Card>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                  <Typography variant="h6" sx={{ fontWeight: 600 }}>
                    {plugin.name}
                  </Typography>
                  <Chip
                    label={plugin.enabled ? 'Enabled' : 'Disabled'}
                    color={plugin.enabled ? 'success' : 'default'}
                    size="small"
                  />
                </Box>
                <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                  {plugin.description}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  Version: {plugin.version}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>
    </Box>
  );
};

export default Plugins;