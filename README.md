# GOnitor

A simple Go server that monitors CPU and RAM usage of the host machine.

## Features

- Real-time CPU usage monitoring
- Memory usage statistics (total, used, free, and usage percentage)
- Human-readable memory values (B, KB, MB, GB, etc.)
- REST API endpoint for metrics
- JSON response format

## Prerequisites

- Go 1.21 or later
- Git

## Setup

1. Clone the repository:
```bash
git clone <repository-url>
cd gonitor
```

2. Install dependencies:
```bash
go mod download
```

3. Run the server:
```bash
go run main.go
```

The server will start on port 8080.

## API Usage

### Get System Metrics

```bash
curl http://localhost:8080/metrics
```

Example response:
```json
{
  "cpu_usage": 25.5,
  "memory_usage": {
    "total": "16.0 GB",
    "used": "8.4 GB",
    "free": "7.6 GB",
    "used_percent": 52.4
  },
  "timestamp": "2024-03-14T12:00:00Z"
}
```

## Metrics Description

- `cpu_usage`: CPU usage percentage (rounded to 2 decimal places)
- `memory_usage.total`: Total system memory in human-readable format
- `memory_usage.used`: Used memory in human-readable format
- `memory_usage.free`: Free memory in human-readable format
- `memory_usage.used_percent`: Memory usage percentage (rounded to 2 decimal places)
- `timestamp`: Time when the metrics were collected 