# SSTS - System Stress Testing Suite

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](https://github.com/pranavgopavaram/ssts)

A comprehensive system stress testing suite designed to safely push your hardware to its limits while monitoring system health and performance metrics.

## 🚀 Quick Start

```bash
# Clone the repository
git clone https://github.com/pranavgopavaram/ssts.git
cd ssts

# Build the application
go build

# Run the server
./ssts server
```

Then open your browser to `http://localhost:8080` to access the web interface.

## 📋 Table of Contents

- [Features](#-features)
- [Architecture](#-architecture)
- [Installation](#-installation)
- [Usage](#-usage)
- [Configuration](#-configuration)
- [API Documentation](#-api-documentation)
- [Safety Features](#-safety-features)
- [Contributing](#-contributing)
- [License](#-license)

## ✨ Features

### 🔥 Stress Testing Capabilities
- **CPU Stress Testing**: Multi-core processor intensive workloads
- **Memory Stress Testing**: RAM allocation and usage patterns
- **Disk I/O Testing**: Storage read/write performance testing
- **Network Testing**: Bandwidth and latency stress testing

### 📊 Real-time Monitoring
- Live system metrics visualization
- Temperature monitoring with safety thresholds
- Resource usage tracking (CPU, Memory, Disk, Network)
- Performance degradation detection

### 🛡️ Safety First
- Automatic emergency stops on dangerous conditions
- Configurable safety limits and thresholds
- Gradual ramp-up to prevent system shock
- Cooldown periods between intensive tests

### 🌐 Web Interface
- Modern, responsive web dashboard
- Real-time charts and graphs
- Test configuration and management
- WebSocket-based live updates

### 🔌 Plugin Architecture
- Extensible plugin system for custom stress tests
- Easy integration of new testing scenarios
- Configurable test parameters
- Plugin health monitoring

## 🏗️ Architecture

SSTS follows a modular architecture with clear separation of concerns:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Interface │────│  API Server     │────│ Core Engine     │
│   (Frontend)    │    │  (REST/WS)      │    │ (Orchestrator)  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                       │
                       ┌─────────────────────────────────┼─────────────────────────────────┐
                       │                                 │                                 │
               ┌───────▼────────┐              ┌─────────▼─────────┐              ┌──────▼──────┐
               │ Safety Monitor │              │ Metrics Collector │              │   Plugins   │
               │   - Temp       │              │   - CPU Usage     │              │ - CPU Test  │
               │   - Resources  │              │   - Memory Usage  │              │ - Memory    │
               │   - Violations │              │   - Disk I/O      │              │ - Disk I/O  │
               └────────────────┘              └───────────────────┘              │ - Network   │
                                                                                  └─────────────┘
```

## 🛠️ Installation

### Prerequisites

- **Go 1.21+**: [Download Go](https://golang.org/dl/)
- **Git**: For cloning the repository
- **Administrative privileges**: Required for some system monitoring features

### From Source

```bash
# Clone the repository
git clone https://github.com/pranavgopavaram/ssts.git
cd ssts

# Install dependencies
go mod download

# Build the application
go build -o ssts

# Run tests to verify installation
make test
```

### Docker (Coming Soon)

```bash
docker run -p 8080:8080 pranavgopavaram/ssts:latest
```

## 🎯 Usage

### Command Line Interface

```bash
# Start the web server (default port 8080)
./ssts server

# Run a specific test configuration
./ssts run-test configs/cpu-stress.yaml

# List available plugins
./ssts plugins list

# Check system health
./ssts health
```

### Web Interface

1. Start the server: `./ssts server`
2. Open browser to `http://localhost:8080`
3. Select test type and configure parameters
4. Monitor real-time results
5. Export results when complete

### Configuration Files

Create YAML configuration files for repeatable tests:

```yaml
# example-cpu-test.yaml
name: "CPU Intensive Test"
description: "Tests CPU under heavy load for 5 minutes"
plugin: "cpu_stress"
duration: "5m"
safety:
  max_cpu_percent: 90
  max_temperature_celsius: 80
config:
  intensity: 75
  cores: 4
  pattern: "prime_calculation"
```

Run with: `./ssts run-test example-cpu-test.yaml`

## ⚙️ Configuration

### Environment Variables

```bash
# Server configuration
SSTS_PORT=8080
SSTS_HOST=localhost
SSTS_LOG_LEVEL=info

# Database configuration
SSTS_DB_TYPE=sqlite
SSTS_DB_PATH=./ssts.db

# InfluxDB for metrics (optional)
SSTS_INFLUX_URL=http://localhost:8086
SSTS_INFLUX_TOKEN=your-token
SSTS_INFLUX_ORG=your-org
SSTS_INFLUX_BUCKET=ssts-metrics
```

### Safety Limits

Configure default safety limits in `config.yaml`:

```yaml
safety:
  max_cpu_percent: 85
  max_memory_percent: 80
  max_disk_percent: 90
  max_temperature_celsius: 80
  emergency_stop_enabled: true
  cooldown_period: "60s"
```

## 📚 API Documentation

### REST Endpoints

- `GET /api/v1/health` - System health status
- `POST /api/v1/tests` - Start a new test
- `GET /api/v1/tests/{id}` - Get test status
- `DELETE /api/v1/tests/{id}` - Stop a running test
- `GET /api/v1/metrics/{id}` - Get test metrics
- `GET /api/v1/plugins` - List available plugins

### WebSocket Events

- `test.started` - Test execution began
- `test.metrics` - Real-time metrics update
- `test.completed` - Test finished
- `test.failed` - Test encountered an error
- `safety.violation` - Safety threshold exceeded

Full API documentation available at `/docs` when server is running.

## 🛡️ Safety Features

### Automatic Protection
- **Temperature monitoring**: Stops tests if CPU/GPU gets too hot
- **Resource limits**: Prevents system from becoming unresponsive
- **Emergency stops**: Immediate test termination when needed
- **Gradual ramp-up**: Slowly increases intensity to prevent shock

### Manual Controls
- **Emergency stop button**: Always accessible in web interface
- **Configurable limits**: Set your own safety thresholds
- **Test scheduling**: Avoid running during important work
- **Backup monitoring**: Multiple monitoring systems for redundancy

## 🧪 Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/core -v

# Run integration tests
make test-integration
```

## 📈 Performance Benchmarks

SSTS has been tested on various systems:

| System Type | CPU | RAM | Duration | Max Temp | Result |
|-------------|-----|-----|----------|----------|---------|
| Gaming PC | i7-9700K | 32GB | 1 hour | 76°C | ✅ Pass |
| Laptop | i5-8250U | 16GB | 30 min | 82°C | ✅ Pass |
| Server | Xeon E5-2680 | 128GB | 4 hours | 68°C | ✅ Pass |

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone your fork
git clone https://github.com/yourusername/ssts.git
cd ssts

# Install development dependencies
go mod download

# Run in development mode
go run main.go server --dev

# Run tests before submitting PR
make test-all
```

### Plugin Development

Create custom stress test plugins:

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my_plugin" }
func (p *MyPlugin) Execute(ctx context.Context, config interface{}) error {
    // Your stress test logic here
    return nil
}
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- 📖 **Documentation**: Check our [Getting Started Guide](GETTING_STARTED.md)
- 🐛 **Bug Reports**: [Create an issue](https://github.com/pranavgopavaram/ssts/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/pranavgopavaram/ssts/discussions)
- 📧 **Email**: [your-email@example.com](mailto:your-email@example.com)

## 🙏 Acknowledgments

- [Go Team](https://golang.org/) for the excellent programming language
- [Gin Framework](https://gin-gonic.com/) for the web framework
- [InfluxDB](https://www.influxdata.com/) for time-series metrics storage
- All our contributors and users

---

**⚠️ Important**: Always monitor your system during stress tests. While SSTS includes comprehensive safety features, you are responsible for your hardware's wellbeing. Start with low-intensity tests and gradually increase as you become familiar with your system's limits.

**Made with ❤️ by [Pranav Agopavaram](https://github.com/pranavgopavaram)**
