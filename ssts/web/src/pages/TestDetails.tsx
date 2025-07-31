import React from 'react';
import { Box, Typography } from '@mui/material';

const TestDetails: React.FC = () => {
  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 2, fontWeight: 600 }}>
        Test Details
      </Typography>
      <Typography variant="body1" color="text.secondary">
        Test details page - implementation in progress
      </Typography>
    </Box>
  );
};

export default TestDetails;