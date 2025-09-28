package workerclient

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type CaseGenFunc func(caseRunnerInfo CaseRunnerInfo) *TestCase

type WorkerRunner struct {
	Worker            *Worker
	CoordinatorApi    string
	CaseMaps          map[string]*TestCase
	RunningCaseRunner *CaseRunner
	httpClient        *HTTPClient
}

func (rw *WorkerRunner) Run() {
	for {
		rw.RealRun()
		time.Sleep(time.Second * 6)
	}
}

func (rw *WorkerRunner) RealRun() {
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("RealRun Error: %v\n", p)
		}
	}()

	rspWPS := rw.PushStatus()
	if rspWPS == nil {
		return
	}
	rw.Worker.BaseInfo.Index = rspWPS.Worker.BaseInfo.Index
	if rspWPS.ShouldRunCase {
		tc := rw.CaseMaps[rspWPS.TestCaseInfo.BaseInfo.Name]
		if tc == nil {
			return
		}
		baseInfo := rspWPS.TestCaseInfo.BaseInfo
		rmc := int64(baseInfo.TotalMaxConcurrency)
		workerConc := int64(baseInfo.WorkerConcurrency)
		widx := rw.Worker.BaseInfo.Index

		// Calculate the concurrency that the current worker should use
		var currentWorkerConcurrency uint64
		remaining := rmc - workerConc*widx

		if remaining <= 0 {
			return
		}
		if remaining < workerConc {
			currentWorkerConcurrency = uint64(remaining)
		} else {
			currentWorkerConcurrency = uint64(workerConc)
		}
		rw.Worker.BaseInfo.Status = "running"
		caseRunnerInfo := CaseRunnerInfo{
			WorkerName:                rw.Worker.BaseInfo.Name,
			MaxConcurrencyInThisWoker: currentWorkerConcurrency,
			RampingSeconds:            baseInfo.RampingSeconds,
			DurationMinutes:           baseInfo.DurationMinutes,
			WorkerTotal:               rspWPS.TestCaseInfo.WorkerTotal,
			WorkerIndex:               uint64(widx),
			WorkerConcurrency:         baseInfo.WorkerConcurrency,
		}
		rw.RunningCaseRunner = &CaseRunner{
			Info:           caseRunnerInfo,
			TestCase:       tc,
			CoordinatorApi: rw.CoordinatorApi,
			httpClient:     rw.httpClient,
		}
		rw.RunningCaseRunner.SetGlobalParams(rspWPS.TestCaseInfo.BaseInfo.GlobalParams)
		go func() {
			rw.RunningCaseRunner.Run()
		}()
		return
	}

	if rspWPS.ShouldStopCase {
		if rw.RunningCaseRunner != nil {
			rw.RunningCaseRunner.StopRunChannel()
		}
	}
}

func (rw *WorkerRunner) PushStatus() (rwps *RspWorkerPushStatus) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("PushStatus Error: %v\n", p)
			rwps = nil
		}
	}()

	if rw.RunningCaseRunner != nil {
		if !rw.RunningCaseRunner.IsRunning {
			rw.RunningCaseRunner = nil
			rw.Worker.BaseInfo.Status = "idle"
		}
	}

	runningCaseName := ""
	activeConcurrencyCount := int64(0)
	if rw.RunningCaseRunner != nil {
		runningCaseName = rw.RunningCaseRunner.TestCase.Name
		activeConcurrencyCount = rw.RunningCaseRunner.ActiveConcurrencyCount
	}

	for _, tc := range rw.Worker.BaseInfo.TestCases {
		if tc.Name == runningCaseName {
			tc.Status = "running"
			tc.ActiveConcurrencyCount = activeConcurrencyCount
		} else {
			tc.Status = "idle"
			tc.ActiveConcurrencyCount = 0
		}
	}

	// Prepare request parameters
	params := &WorkerPushStatusParams{
		BaseInfo: rw.Worker.BaseInfo,
	}

	// Send HTTP request
	targetUrl := fmt.Sprintf("%v/worker/push_status", rw.CoordinatorApi)
	rsp := &RspWorkerPushStatusBody{}

	if err := rw.httpClient.PostJSON(targetUrl, params, rsp); err != nil {
		fmt.Printf("PushStatus HTTP request failed: %v\n", err)
		return nil
	}

	return rsp.Data
}

func (rw *WorkerRunner) AddTestCase(tc *TestCase) {
	if rw.CaseMaps[tc.Name] != nil {
		panic(fmt.Sprintf("test case %s already exists", tc.Name))
	}
	rw.CaseMaps[tc.Name] = tc
	rw.Worker.BaseInfo.TestCases = append(rw.Worker.BaseInfo.TestCases, &TestCaseSummary{
		Name:   tc.Name,
		Status: "idle",
	})
}

func NewWorkerRunner(workerName, coordinatorApi string) *WorkerRunner {
	wk := &Worker{
		BaseInfo: &WorkerBaseInfo{
			Name:   workerName,
			ID:     uuid.New().String(),
			Index:  -1,
			Status: "idle",
		},
	}
	return &WorkerRunner{
		Worker:         wk,
		CoordinatorApi: coordinatorApi,
		CaseMaps:       map[string]*TestCase{},
		httpClient:     NewHTTPClient(5 * time.Second),
	}
}
