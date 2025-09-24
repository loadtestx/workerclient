# Load Test Worker Client

A high-performance distributed load testing worker client for executing load test tasks and communicating with the coordinator.

## Features

- **Distributed Architecture**: Supports multiple workers collaborating to execute load tests
- **High Concurrency Support**: Goroutine-based concurrency model with configurable maximum concurrency
- **RPS Control**: Supports requests per second limiting and gradual load ramping
- **Real-time Monitoring**: Collects and reports performance metrics like response time and success rate
- **Flexible Configuration**: Supports multi-step test flows, parameterized testing, and conditional execution
- **Plugin Architecture**: Supports different types of request plugin implementations

## Project Structure

```
/workerclient/
├── case_runner.go          # Test case runner
├── worker_runner.go        # Worker runner  
├── test_case.go           # Test case definition
├── result.go              # Test result processing
├── types.go               # Common data structures and type definitions
├── utils.go               # Utility functions
├── go.mod                 # Go module definition
└── README.md              # Project documentation
```

## Core Modules

### 1. Worker Management (`worker_runner.go`)
- Worker lifecycle management
- Communication with coordinator
- Task scheduling and concurrency control

### 2. Test Case Execution (`case_runner.go`)
- Concurrency control and RPS limiting
- Gradual load ramping
- Real-time performance monitoring and metrics collection

### 3. Test Case Definition (`test_case.go`)
- Multi-step test flows
- Parameterized testing support
- Conditional execution and error handling

### 4. Result Processing (`result.go`)
- Unified request result interface
- Detailed request/response information collection
- Sub-request support

### 5. Data Type Definitions (`types.go`)
- API communication data structures
- Test configuration and monitoring metrics
- Worker state management

## Quick Start

### Requirements

- Go 1.19+
- Coordinator service

### Install Dependencies

```bash
go mod tidy
```

### Basic Usage

```go
package main

import (
    "gitlab.corp.algento.com/loadtest/workerclient"
)

func main() {
    // Create Worker Runner
    workerRunner := workerclient.NewWorkerRunner("worker-1", "http://coordinator:8080")
    
    // Create test case
    testCase := workerclient.NewTestCase("api_test")
    
    // Add test step
    testCase.AddStep(&workerclient.TestStep{
        StepName: "login",
        ReqPluginFunc: func(params map[string]string) interface{} {
            // Implement specific request logic
            result := workerclient.AcquireResult("login")
            result.Begin()
            
            // Execute HTTP request...
            result.ResponseCode = 200
            result.End()
            
            return result
        },
        GenReqParamsFunc: func(caseParams *workerclient.CaseParams) map[string]string {
            return map[string]string{
                "username": "test",
                "password": "123456",
            }
        },
    })
    
    // Add test case to worker
    workerRunner.AddTestCase(testCase)
    
    // Start worker
    workerRunner.Run()
}
```

## Configuration

### Test Case Configuration

```go
type CaseBaseInfo struct {
    Name                string            `json:"name"`                // Test case name
    GlobalParams        map[string]string `json:"globalParams"`        // Global parameters
    MaxConcurrencyCount uint64            `json:"maxConcurrencyCount"` // Maximum concurrency
    RampingSeconds      uint64            `json:"rampingSeconds"`      // Ramping time (seconds)
    DurationMinutes     uint64            `json:"durationMinutes"`     // Duration (minutes)
    WorkerSize          uint64            `json:"workerSize"`          // Worker size
}
```

### Internal Variables

The system automatically injects the following internal variables into request parameters:

- `__name`: Step name
- `__goroutine_id`: Goroutine ID
- `__executor_index`: Executor index
- `__worker_total`: Total number of workers
- `__worker_index`: Worker index
- `__worker_size`: Worker size

## API Interfaces

### Communication with Coordinator

#### Push Status
```
POST /worker/push_status
```

#### Send Metrics
```
POST /worker/send_step_metrics
```

## Performance Monitoring

The system automatically collects the following performance metrics:

- **Response Time**: Uses TDigest algorithm for data compression
- **Success Rate**: Based on HTTP status code judgment
- **Throughput**: Requests per second and bytes per second
- **Concurrency**: Real-time active concurrency count

### Metric Types

- `step_call`: Step call metrics
- `step_call_integral`: Step call cumulative metrics

## Dependencies

- `github.com/Narasimha1997/ratelimiter`: RPS limiting
- `github.com/caio/go-tdigest/v4`: Performance data compression
- `github.com/eapache/queue`: Queue management
- `github.com/google/uuid`: UUID generation

## Development Guide

### Implementing Custom Request Plugins

```go
type CustomRequestPlugin struct {
    // Custom fields
}

func (p *CustomRequestPlugin) Execute(params map[string]string) workerclient.IResultV1 {
    result := workerclient.AcquireResult("custom_request")
    result.Begin()
    
    // Implement custom request logic
    
    result.End()
    return result
}
```

### Adding Test Steps

```go
testStep := &workerclient.TestStep{
    StepName: "custom_step",
    ReqPluginFunc: func(params map[string]string) interface{} {
        // Request processing logic
    },
    GenReqParamsFunc: func(caseParams *workerclient.CaseParams) map[string]string {
        // Parameter generation logic
    },
    PreFunc: func(caseParams *workerclient.CaseParams, reqParams map[string]string) {
        // Pre-processing
    },
    PostFunc: func(caseParams *workerclient.CaseParams, reqParams map[string]string, res workerclient.IResultV1) {
        // Post-processing
    },
    ExecWhenFunc: func(caseParams *workerclient.CaseParams, reqParams map[string]string) bool {
        // Execution condition judgment
        return true
    },
    ContinueWhenFailed: false, // Whether to continue on failure
    RpsLimitFunc: func(caseRunnerInfo workerclient.CaseRunnerInfo, globalParams map[string]string) uint64 {
        // RPS limiting
        return 100
    },
}
```

## Architecture Overview

### System Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Coordinator   │◄──►│  Worker Client  │◄──►│  Target System  │
│                 │    │                 │    │                 │
│ - Task Schedule │    │ - Test Execution│    │ - API Endpoints │
│ - Metrics Collect│   │ - Result Process│    │ - Services      │
│ - Worker Manage │    │ - Status Report │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Data Flow

1. **Task Assignment**: Coordinator assigns test tasks to workers
2. **Test Execution**: Workers execute test cases with specified concurrency
3. **Metrics Collection**: Real-time collection of performance metrics
4. **Result Reporting**: Workers report results back to coordinator
5. **Analysis**: Coordinator analyzes and aggregates results

## Best Practices

### Test Case Design

- Use meaningful step names for better debugging
- Implement proper error handling in request plugins
- Set appropriate RPS limits to avoid overwhelming target systems
- Use parameterized testing for data-driven scenarios

### Performance Optimization

- Configure appropriate concurrency levels based on target system capacity
- Use gradual ramping to avoid sudden load spikes
- Monitor system resources during test execution
- Implement proper cleanup in tear-down functions

### Monitoring and Debugging

- Check coordinator logs for task assignment issues
- Monitor worker status and active concurrency
- Analyze response time distributions using TDigest data
- Use internal variables for request correlation

## Troubleshooting

### Common Issues

1. **Worker not receiving tasks**: Check coordinator connectivity and worker registration
2. **High failure rates**: Verify target system availability and request parameters
3. **Memory issues**: Reduce concurrency or optimize request plugins
4. **Network timeouts**: Adjust timeout settings and check network connectivity

### Debug Mode

Enable debug logging by setting appropriate log levels in your application.

## License

This project is licensed under internal license for Algento company use only.

## Contributing

Issues and Pull Requests are welcome to improve the project.

## Contact

For questions, please contact the development team.