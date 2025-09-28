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
    "loadtestx/workerclient"
)

func main() {
    // Create Worker Runner
    workerRunner := workerclient.NewWorkerRunner("worker-1", "http://coordinator:8080")
    
    // Create test case
    testCase := workerclient.NewTestCase("api_test")
    
    // Add test step
    testCase.AddStep(&workerclient.TestStep{
        StepName: "login",
        ReqPluginFunc: func(params map[string]string) IResultV1 {
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

### GenReqParamsFunc 参数生成函数

`GenReqParamsFunc` 是一个关键的回调函数，用于在每个测试步骤执行前动态生成请求参数。它接收一个包含完整上下文信息的 `CaseParams` 对象，并返回一个参数映射，这些参数将被传递给 `ReqPluginFunc`。

### 函数签名
```go
func(caseParams *CaseParams) map[string]string
```

### CaseParams 结构体

`CaseParams` 结构体包含执行测试步骤时所需的所有上下文信息：

```go
type CaseParams struct {
    GlobalParams    map[string]string  // 全局参数（来自测试用例配置）
    CoroutineParams map[string]string  // 协程级别的参数（每个并发执行器独立）
    CaseRunnerInfo  CaseRunnerInfo     // 运行器信息
}
```

### CaseRunnerInfo 结构体

```go
type CaseRunnerInfo struct {
    WorkerName                string  // 工作器名称
    MaxConcurrencyInThisWoker uint64  // 当前工作器的最大并发数
    RampingSeconds            uint64  // 梯度增加时间（秒）
    DurationMinutes           uint64  // 持续时间（分钟）
    WorkerTotal               uint64  // 总工作器数量
    WorkerIndex               uint64  // 当前工作器索引
    WorkerSize                uint64  // 工作器规模
}
```

### 可用内部变量

在 `GenReqParamsFunc` 中可以访问以下内部变量（通过 `CoroutineParams`）：

- `__goroutine_id`: Goroutine ID（格式："testcaseName-index"）
- `__executor_index`: 执行器索引
- `__worker_total`: 总工作器数量
- `__worker_index`: 当前工作器索引
- `__worker_size`: 工作器规模

注意：`__name`（步骤名称）会在 `GenReqParamsFunc` 执行后自动注入到请求参数中。

### 使用场景示例

#### 1. 静态参数生成
```go
GenReqParamsFunc: func(caseParams *CaseParams) map[string]string {
    return map[string]string{
        "username": "test_user",
        "password": "test_password",
        "app_id": "1001",
    }
}
```

#### 2. 动态参数生成（基于执行上下文）
```go
GenReqParamsFunc: func(caseParams *CaseParams) map[string]string {
    executorIndex := caseParams.CoroutineParams["__executor_index"]
    workerIndex := caseParams.CoroutineParams["__worker_index"]
    
    return map[string]string{
        "user_id": fmt.Sprintf("user_%s_%s", workerIndex, executorIndex),
        "request_id": uuid.New().String(),
        "timestamp": fmt.Sprintf("%d", time.Now().Unix()),
    }
}
```

#### 3. 基于全局参数的动态配置
```go
GenReqParamsFunc: func(caseParams *CaseParams) map[string]string {
    baseUrl := caseParams.GlobalParams["base_url"]
    apiVersion := caseParams.GlobalParams["api_version"]
    
    return map[string]string{
        "base_url": baseUrl,
        "api_version": apiVersion,
        "endpoint": fmt.Sprintf("%s/v%s/api/login", baseUrl, apiVersion),
    }
}
```

#### 4. 复杂的业务逻辑参数生成
```go
GenReqParamsFunc: func(caseParams *CaseParams) map[string]string {
    params := make(map[string]string)
    
    // 基于执行器索引生成不同的测试数据
    index, _ := strconv.Atoi(caseParams.CoroutineParams["__executor_index"])
    params["test_data_id"] = fmt.Sprintf("data_%04d", index)
    
    return params
}
```

### 最佳实践

1. **参数复用性**: 尽量使用全局参数进行配置，便于统一管理
2. **唯一性保证**: 对于需要唯一性的参数（如用户ID、请求ID），利用执行器信息确保唯一
3. **错误处理**: 确保参数生成逻辑的健壮性，避免因为参数缺失导致测试失败
4. **性能考虑**: 避免在参数生成函数中执行耗时的操作

### 注意事项

- `GenReqParamsFunc` 在每个请求执行前都会被调用
- 返回的参数映射将被传递给 `ReqPluginFunc` 使用
- 可以修改和扩展参数，但不能修改 `CaseParams` 结构体本身
- 系统会自动注入 `__name` 参数（步骤名称）到最终的请求参数中

## Internal Variables

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
    ReqPluginFunc: func(params map[string]string) IResultV1 {
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

MIT License.

## Contributing

Issues and Pull Requests are welcome to improve the project.

## Contact

For questions, please contact the development team.