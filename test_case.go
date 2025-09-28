package workerclient

import (
	"fmt"
	"time"
)

func NewTestCase(caseName string) *TestCase {
	return &TestCase{
		Name: caseName,
	}
}

type CaseParams struct {
	GlobalParams    map[string]string
	CoroutineParams map[string]string
	CaseRunnerInfo  CaseRunnerInfo
}

type TestCase struct {
	Name      string
	Teststeps []*TestStep
	TearDown  func(coroutineParams map[string]string)
}

type TestStep struct {
	stepIndex            string
	StepName             string
	ReqPluginFunc        func(reqPamrams map[string]string) (res IResultV1)
	GenReqParamsFunc     func(caseParams *CaseParams) (p map[string]string)
	ContinueWhenFailed   bool
	ExecWhenFunc         func(caseParams *CaseParams, reqPamrams map[string]string) (b bool)
	PreFunc              func(caseParams *CaseParams, reqPamrams map[string]string)
	PostFunc             func(caseParams *CaseParams, reqPamrams map[string]string, res IResultV1)
	RpsLimitFunc         func(caseRunnerInfo CaseRunnerInfo, globalParams map[string]string) (rps uint64)
}

func (ts *TestStep) GetStepIndex() string {
	return ts.stepIndex
}

func (tc *TestCase) AddStep(ts *TestStep) {
	if ts.ExecWhenFunc == nil {
		ts.ExecWhenFunc = func(caseParams *CaseParams, reqPamrams map[string]string) (b bool) { return true }
	}

	if ts.PreFunc == nil {
		ts.PreFunc = func(caseParams *CaseParams, reqPamrams map[string]string) {}
	}

	if ts.PostFunc == nil {
		ts.PostFunc = func(caseParams *CaseParams, reqPamrams map[string]string, res IResultV1) {}
	}

	if ts.RpsLimitFunc == nil {
		ts.RpsLimitFunc = func(caseRunnerInfo CaseRunnerInfo, globalParams map[string]string) (rps uint64) {
			return 0
		}
	}

	ts.stepIndex = fmt.Sprintf("%v", len(tc.Teststeps))
	tc.Teststeps = append(tc.Teststeps, ts)
}

func (tc *TestCase) Run(globalParams, coroutineParams map[string]string, rpsQLimiter *RpsQLimiter, output *Output, caseRunner *CaseRunner) {
	caseParams := &CaseParams{
		GlobalParams:    globalParams,
		CoroutineParams: coroutineParams,
		CaseRunnerInfo:  caseRunner.Info,
	}

	for {
		if !caseRunner.IsRunning {
			break
		}
		for _, ts := range tc.Teststeps {
			if !caseRunner.IsRunning {
				break
			}
			reqParams := ts.GenReqParamsFunc(caseParams)
			reqParams[InnerVarName] = ts.StepName
			reqParams[InnerVarGoroutineId] = caseParams.CoroutineParams[InnerVarGoroutineId]
			reqParams[InnerVarExecutorIndex] = caseParams.CoroutineParams[InnerVarExecutorIndex]
			if !ts.ExecWhenFunc(caseParams, reqParams) {
				continue
			}

			if rpsQLimiter.Limter.HasKey(ts.GetStepIndex()) {
				ch := make(chan bool)
				rpsQLimiter.Lock.Lock()
				rpsQLimiter.QMap[ts.GetStepIndex()].Add(ch)
				rpsQLimiter.Lock.Unlock()
				<-ch
			}

			if !caseRunner.IsRunning {
				break
			}

			ts.PreFunc(caseParams, reqParams)
			results := []IResultV1{}
			res := ts.ReqPluginFunc(reqParams)
			subResults := res.GetSubResults()
			if len(subResults) == 0 {
				results = append(results, res)
			} else {
				for _, sr := range subResults {
					results = append(results, interface{}(sr).(IResultV1))
				}
			}

			ok := true
			for _, result := range results {
				ts.PostFunc(caseParams, reqParams, result)
				ok = result.IsSuccess() && ok
				if output.ResChans != nil {
					output.ResChans <- result
				}
			}
			if !ok && !ts.ContinueWhenFailed {
				break
			}


		}
		time.Sleep(100 * time.Millisecond)
	}

	if tc.TearDown != nil {
		tc.TearDown(coroutineParams)
	}

}
