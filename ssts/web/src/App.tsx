import React from 'react';
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { Box } from '@mui/material';

// Components
import Navbar from './components/Layout/Navbar';
import Sidebar from './components/Layout/Sidebar';
import Dashboard from './pages/Dashboard';
import Tests from './pages/Tests';
import TestDetails from './pages/TestDetails';
import Executions from './pages/Executions';
import ExecutionDetails from './pages/ExecutionDetails';
import Plugins from './pages/Plugins';
import System from './pages/System';

// Theme configuration following UI/UX requirements
const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#2563EB', // Primary Blue
      dark: '#1D4ED8', // Primary Dark
      light: '#3B82F6', // Primary Light
    },
    secondary: {
      main: '#7C3AED', // Accent Purple
    },
    success: {
      main: '#10B981', // Success color
    },
    warning: {
      main: '#F59E0B', // Warning color
    },
    error: {
      main: '#EF4444', // Error color
    },
    info: {
      main: '#3B82F6', // Info color
    },
    background: {
      default: '#F9FAFB', // Gray 50
      paper: '#FFFFFF',
    },
    text: {
      primary: '#111827', // Gray 900
      secondary: '#6B7280', // Gray 500
    },
  },
  typography: {
    fontFamily: 'Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    h1: {
      fontSize: '2.25rem', // 36px
      fontWeight: 600,
      lineHeight: '2.75rem', // 44px
    },
    h2: {
      fontSize: '1.875rem', // 30px
      fontWeight: 600,
      lineHeight: '2.375rem', // 38px
    },
    h3: {
      fontSize: '1.5rem', // 24px
      fontWeight: 600,
      lineHeight: '2rem', // 32px
    },
    h4: {
      fontSize: '1.25rem', // 20px
      fontWeight: 600,
      lineHeight: '1.75rem', // 28px
    },
    h5: {
      fontSize: '1.125rem', // 18px
      fontWeight: 500,
      lineHeight: '1.625rem', // 26px
    },
    h6: {
      fontSize: '1rem', // 16px
      fontWeight: 500,
      lineHeight: '1.5rem', // 24px
    },
    body1: {
      fontSize: '1rem', // 16px
      fontWeight: 400,
      lineHeight: '1.5rem', // 24px
    },
    body2: {
      fontSize: '0.875rem', // 14px
      fontWeight: 400,
      lineHeight: '1.25rem', // 20px
    },
    caption: {
      fontSize: '0.6875rem', // 11px
      fontWeight: 400,
      lineHeight: '0.875rem', // 14px
    },
  },
  shape: {
    borderRadius: 8, // 8px border radius
  },
  spacing: 8, // 8px base spacing unit
  shadows: [
    'none',
    '0 1px 2px rgba(0,0,0,0.05)',
    '0 1px 3px rgba(0,0,0,0.1), 0 1px 2px rgba(0,0,0,0.06)',
    '0 4px 6px rgba(0,0,0,0.1)',
    '0 10px 15px rgba(0,0,0,0.1)',
    // ... other shadow levels
  ] as any,
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          textTransform: 'none',
          fontWeight: 500,
          minHeight: 40,
          paddingLeft: 24,
          paddingRight: 24,
        },
        contained: {
          boxShadow: '0 1px 2px rgba(0,0,0,0.05)',
          '&:hover': {
            boxShadow: '0 4px 6px rgba(0,0,0,0.1)',
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 12,
          boxShadow: '0 1px 3px rgba(0,0,0,0.1), 0 1px 2px rgba(0,0,0,0.06)',
          border: 'none',
        },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius: 8,
            height: 44,
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          borderRadius: 12,
        },
      },
    },
  },
});

const App: React.FC = () => {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Router>
        <Box sx={{ display: 'flex' }}>
          <Navbar />
          <Sidebar />
          <Box
            component="main"
            sx={{
              flexGrow: 1,
              p: 3,
              width: { sm: `calc(100% - 240px)` },
              ml: { sm: '240px' },
              mt: '64px',
              minHeight: 'calc(100vh - 64px)',
              backgroundColor: 'background.default',
            }}
          >
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/tests" element={<Tests />} />
              <Route path="/tests/:id" element={<TestDetails />} />
              <Route path="/executions" element={<Executions />} />
              <Route path="/executions/:id" element={<ExecutionDetails />} />
              <Route path="/plugins" element={<Plugins />} />
              <Route path="/system" element={<System />} />
            </Routes>
          </Box>
        </Box>
      </Router>
    </ThemeProvider>
  );
};

export default App;