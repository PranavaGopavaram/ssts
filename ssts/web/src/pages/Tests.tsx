import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Button,
  Card,
  CardContent,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Grid,
  Alert,
} from '@mui/material';
import {
  Add as AddIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  PlayArrow as PlayArrowIcon,
  Visibility as ViewIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';

interface TestConfiguration {
  id: string;
  name: string;
  description: string;
  plugin: string;
  duration: string;
  created: string;
  updated: string;
}

const Tests: React.FC = () => {
  const navigate = useNavigate();
  const [tests, setTests] = useState<TestConfiguration[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newTest, setNewTest] = useState({
    name: '',
    description: '',
    plugin: 'cpu-stress',
    duration: '300s',
  });

  useEffect(() => {
    fetchTests();
  }, []);

  const fetchTests = async () => {
    try {
      const response = await fetch('/api/v1/tests');
      if (response.ok) {
        const data = await response.json();
        setTests(data);
      } else {
        setError('Failed to fetch tests');
      }
    } catch (err) {
      setError('Network error');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateTest = async () => {
    try {
      const response = await fetch('/api/v1/tests', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ...newTest,
          config: {},
          safety: {
            max_cpu_percent: 80,
            max_memory_percent: 70,
            max_disk_percent: 90,
            max_network_mbps: 100,
          },
        }),
      });

      if (response.ok) {
        setCreateDialogOpen(false);
        setNewTest({
          name: '',
          description: '',
          plugin: 'cpu-stress',
          duration: '300s',
        });
        fetchTests();
      } else {
        setError('Failed to create test');
      }
    } catch (err) {
      setError('Network error');
    }
  };

  const handleRunTest = async (testId: string) => {
    try {
      const response = await fetch(`/api/v1/tests/${testId}/run`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          intensity: 70,
        }),
      });

      if (response.ok) {
        const result = await response.json();
        navigate(`/executions/${result.execution_id}`);
      } else {
        setError('Failed to start test');
      }
    } catch (err) {
      setError('Network error');
    }
  };

  const handleDeleteTest = async (testId: string) => {
    if (!window.confirm('Are you sure you want to delete this test?')) {
      return;
    }

    try {
      const response = await fetch(`/api/v1/tests/${testId}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        fetchTests();
      } else {
        setError('Failed to delete test');
      }
    } catch (err) {
      setError('Network error');
    }
  };

  const getPluginName = (plugin: string) => {
    const pluginNames: Record<string, string> = {
      'cpu-stress': 'CPU Stress',
      'memory-stress': 'Memory Stress',
      'io-stress': 'I/O Stress',
    };
    return pluginNames[plugin] || plugin;
  };

  const formatDuration = (duration: string) => {
    const seconds = parseInt(duration.replace('s', ''));
    if (seconds >= 60) {
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = seconds % 60;
      return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
    }
    return duration;
  };

  if (loading) {
    return (
      <Box>
        <Typography>Loading tests...</Typography>
      </Box>
    );
  }

  return (
    <Box>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Box>
          <Typography variant="h4" sx={{ mb: 1, fontWeight: 600 }}>
            Test Configurations
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Manage and execute stress test configurations
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setCreateDialogOpen(true)}
        >
          Create Test
        </Button>
      </Box>

      {/* Error Alert */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {/* Tests Table */}
      <Card>
        <CardContent sx={{ p: 0 }}>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Name</TableCell>
                  <TableCell>Plugin</TableCell>
                  <TableCell>Duration</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {tests.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} sx={{ textAlign: 'center', py: 4 }}>
                      <Typography variant="body2" color="text.secondary">
                        No tests configured. Create your first test to get started.
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  tests.map((test) => (
                    <TableRow key={test.id} hover>
                      <TableCell>
                        <Box>
                          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
                            {test.name}
                          </Typography>
                          {test.description && (
                            <Typography variant="body2" color="text.secondary">
                              {test.description}
                            </Typography>
                          )}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip
                          label={getPluginName(test.plugin)}
                          size="small"
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>{formatDuration(test.duration)}</TableCell>
                      <TableCell>
                        {new Date(test.created).toLocaleDateString()}
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', gap: 1 }}>
                          <IconButton
                            size="small"
                            onClick={() => handleRunTest(test.id)}
                            color="primary"
                            title="Run Test"
                          >
                            <PlayArrowIcon />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => navigate(`/tests/${test.id}`)}
                            title="View Details"
                          >
                            <ViewIcon />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => navigate(`/tests/${test.id}/edit`)}
                            title="Edit Test"
                          >
                            <EditIcon />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => handleDeleteTest(test.id)}
                            color="error"
                            title="Delete Test"
                          >
                            <DeleteIcon />
                          </IconButton>
                        </Box>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* Create Test Dialog */}
      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Test</DialogTitle>
        <DialogContent>
          <Grid container spacing={2} sx={{ mt: 1 }}>
            <Grid item xs={12}>
              <TextField
                label="Test Name"
                fullWidth
                value={newTest.name}
                onChange={(e) => setNewTest({ ...newTest, name: e.target.value })}
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                label="Description"
                fullWidth
                multiline
                rows={2}
                value={newTest.description}
                onChange={(e) => setNewTest({ ...newTest, description: e.target.value })}
              />
            </Grid>
            <Grid item xs={6}>
              <TextField
                label="Plugin"
                fullWidth
                select
                SelectProps={{ native: true }}
                value={newTest.plugin}
                onChange={(e) => setNewTest({ ...newTest, plugin: e.target.value })}
              >
                <option value="cpu-stress">CPU Stress</option>
                <option value="memory-stress">Memory Stress</option>
                <option value="io-stress">I/O Stress</option>
              </TextField>
            </Grid>
            <Grid item xs={6}>
              <TextField
                label="Duration"
                fullWidth
                value={newTest.duration}
                onChange={(e) => setNewTest({ ...newTest, duration: e.target.value })}
                helperText="e.g., 300s, 5m, 1h"
              />
            </Grid>
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleCreateTest} variant="contained">
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Tests;