import React from 'react';
import { Box, Typography } from '@mui/material';

const Executions: React.FC = () => {
  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 2, fontWeight: 600 }}>
        Test Executions
      </Typography>
      <Typography variant="body1" color="text.secondary">
        Test executions page - implementation in progress
      </Typography>
    </Box>
  );
};

export default Executions;