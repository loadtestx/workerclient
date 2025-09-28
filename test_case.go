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

type CaseParmas struct {
	GlobalParams    map[string]string
	CoroutineParams map[string]string
	RuntimeParams   map[string]string
	CaseRunnerInfo  CaseRunnerInfo
}

type TestCase struct {
	Name      string
	Teststeps []*TestStep
	TearDown  func(coroutineParams map[string]string)
}

type TestStep struct {
	StepIndex            string
	StepName             string
	ReqPluginFunc        func(reqPamrams map[string]string) (res IResultV1)
	SetRuntimeParamsFunc func(caseParmas *CaseParmas)
	GenReqParamsFunc     func(caseParmas *CaseParmas) (p map[string]string)
	ContinueWhenFailed   bool
	ExecWhenFunc         func(caseParmas *CaseParmas, reqPamrams map[string]string) (b bool)
	PreFunc              func(caseParmas *CaseParmas, reqPamrams map[string]string)
	PostFunc             func(caseParmas *CaseParmas, reqPamrams map[string]string, res IResultV1)
	RpsLimitFunc         func(caseRunnerInfo CaseRunnerInfo, globalParams map[string]string) (rps uint64)
}

func (tc *TestCase) AddStep(ts *TestStep) {
	if ts.SetRuntimeParamsFunc == nil {
		ts.SetRuntimeParamsFunc = func(caseParmas *CaseParmas) {}
	}
	if ts.ExecWhenFunc == nil {
		ts.ExecWhenFunc = func(caseParmas *CaseParmas, reqPamrams map[string]string) (b bool) { return true }
	}

	if ts.PreFunc == nil {
		ts.PreFunc = func(caseParmas *CaseParmas, reqPamrams map[string]string) {}
	}

	if ts.PostFunc == nil {
		ts.PostFunc = func(caseParmas *CaseParmas, reqPamrams map[string]string, res IResultV1) {}
	}

	if ts.RpsLimitFunc == nil {
		ts.RpsLimitFunc = func(caseRunnerInfo CaseRunnerInfo, globalParams map[string]string) (rps uint64) {
			return 0
		}
	}

	ts.StepIndex = fmt.Sprintf("%v", len(tc.Teststeps))
	tc.Teststeps = append(tc.Teststeps, ts)
}

func (tc *TestCase) Run(globalParams, coroutineParams map[string]string, rpsQLimiter *RpsQLimiter, output *Output, caseRunner *CaseRunner) {
	caseParmas := &CaseParmas{
		GlobalParams:    globalParams,
		CoroutineParams: coroutineParams,
		RuntimeParams:   map[string]string{},
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
			ts.SetRuntimeParamsFunc(caseParmas)
			reqParams := ts.GenReqParamsFunc(caseParmas)
			reqParams[InnerVarName] = ts.StepName
			reqParams[InnerVarGoroutineId] = caseParmas.CoroutineParams[InnerVarGoroutineId]
			reqParams[InnerVarExecutorIndex] = caseParmas.CoroutineParams[InnerVarExecutorIndex]
			if !ts.ExecWhenFunc(caseParmas, reqParams) {
				continue
			}

			if rpsQLimiter.Limter.HasKey(ts.StepIndex) {
				ch := make(chan bool)
				rpsQLimiter.Lock.Lock()
				rpsQLimiter.QMap[ts.StepIndex].Add(ch)
				rpsQLimiter.Lock.Unlock()
				<-ch
			}

			if !caseRunner.IsRunning {
				break
			}

			ts.PreFunc(caseParmas, reqParams)
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
				ts.PostFunc(caseParmas, reqParams, result)
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
