import React from 'react';
import { Box, Typography } from '@mui/material';

const ExecutionDetails: React.FC = () => {
  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 2, fontWeight: 600 }}>
        Execution Details
      </Typography>
      <Typography variant="body1" color="text.secondary">
        Execution details page - implementation in progress
      </Typography>
    </Box>
  );
};

export default ExecutionDetails;