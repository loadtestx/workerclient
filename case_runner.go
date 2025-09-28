package workerclient

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Narasimha1997/ratelimiter"
	"github.com/caio/go-tdigest/v4"
	"github.com/eapache/queue"
)

type CaseRunnerInfo struct {
	WorkerName                string
	MaxConcurrencyInThisWoker uint64
	RampingSeconds            uint64
	DurationMinutes           uint64
	WorkerTotal               uint64
	WorkerIndex               uint64
	WorkerSize                uint64
}

type CaseRunner struct {
	Info                   CaseRunnerInfo
	TestCase               *TestCase
	GlobalParams           map[string]string
	IsRunning              bool
	Output                 *Output
	MetricsChan            chan ([]*CallTimeMetric)
	ActiveConcurrencyCount int64
	CoordinatorApi         string
	httpClient             *HTTPClient
}

type RpsQLimiter struct {
	Lock   sync.Mutex
	Limter *ratelimiter.AttributeBasedLimiter
	QMap   map[string]*queue.Queue
}

type Output struct {
	ResChans chan IResultV1
}

func (cr *CaseRunner) Run() {
	cr.IsRunning = true
	cr.ActiveConcurrencyCount = 0
	cr.Output = &Output{
		ResChans: make(chan IResultV1, 1000),
	}
	cr.MetricsChan = make(chan ([]*CallTimeMetric), 1000)
	go func() {
		cr.HandleOuput()
	}()

	go func() {
		cr.SendMetrics()
	}()

	rpsQLimiter := &RpsQLimiter{
		Lock:   sync.Mutex{},
		Limter: ratelimiter.NewAttributeBasedLimiter(true),
		QMap:   map[string]*queue.Queue{},
	}
	for _, ts := range cr.TestCase.Teststeps {
		rps := ts.RpsLimitFunc(cr.Info, cr.GlobalParams)
		if rps > 0 {
			rpsQLimiter.Limter.CreateNewKey(ts.GetStepIndex(), rps, time.Second)
			rpsQLimiter.QMap[ts.GetStepIndex()] = queue.New()
		}
	}

	go func(rql *RpsQLimiter) {
		for {
			isHit := false
			for k, v := range rql.QMap {
				if v.Length() > 0 {
					aw, _ := rql.Limter.ShouldAllow(k, 1)
					if aw || !cr.IsRunning {
						rql.Lock.Lock()
						ch := (v.Remove()).(chan bool)
						ch <- true
						rql.Lock.Unlock()
						isHit = true
					}
				}
			}
			if !isHit {
				time.Sleep(time.Millisecond * 10)
			}
		}
	}(rpsQLimiter)

	rampingLimit := uint64(10000)
	rampingLimitDuration := time.Millisecond * 10
	if cr.Info.RampingSeconds > 0 {
		rampingLimitDuration = time.Second
		rampingLimit = cr.Info.MaxConcurrencyInThisWoker / cr.Info.RampingSeconds
		for {
			if rampingLimit > 0 {
				break
			}
			rampingLimitDuration += time.Second
			rampingLimit = cr.Info.MaxConcurrencyInThisWoker * uint64(rampingLimitDuration/time.Second) / cr.Info.RampingSeconds
		}
	}
	rampingLimiter := ratelimiter.NewDefaultLimiter(rampingLimit, rampingLimitDuration)
	for i := 0; i < int(cr.Info.MaxConcurrencyInThisWoker); i++ {
		for {
			allowed, _ := rampingLimiter.ShouldAllow(1)
			if allowed || !cr.IsRunning {
				break
			} else {
				time.Sleep(time.Millisecond * 25)
			}
		}

		if !cr.IsRunning {
			return
		}
		coroutineParams := map[string]string{
			InnerVarGoroutineId:   fmt.Sprintf("%v-%v", cr.TestCase.Name, i),
			InnerVarExecutorIndex: fmt.Sprintf("%v", i),
			InnerVarWorkerTotal:   fmt.Sprintf("%v", cr.Info.WorkerTotal),
			InnerVarWorkerIndex:   fmt.Sprintf("%v", cr.Info.WorkerIndex),
			InnerVarWorkerSize:    fmt.Sprintf("%v", cr.Info.WorkerSize),
		}
		go func(gp, cp map[string]string, rql *RpsQLimiter, op *Output, _cr *CaseRunner) {
			cr.TestCase.Run(gp, cp, rql, op, _cr)
		}(cr.GlobalParams, coroutineParams, rpsQLimiter, cr.Output, cr)
		cr.ActiveConcurrencyCount += 1
	}
}

func (cr *CaseRunner) SetGlobalParams(globalParams map[string]string) {
	cr.GlobalParams = globalParams
}

func (cr *CaseRunner) StopRunChannel() {
	cr.IsRunning = false
	rc := cr.Output.ResChans
	mc := cr.MetricsChan
	time.Sleep(time.Second * 6)
	cr.Output.ResChans = nil
	time.Sleep(time.Second * 5)
	close(rc)
	time.Sleep(time.Second * 2)
	cr.MetricsChan = nil
	time.Sleep(time.Second * 3)
	close(mc)
}

func (cr *CaseRunner) HandleOuput() {
	callTimeMap := map[CallTimeMapKey]*tdigest.TDigest{}
	lastTs := int(time.Now().UnixMilli() / 1000 / 60)
	for res := range cr.Output.ResChans {
		ts := int(time.Now().UnixMilli() / 1000 / 60)
		if lastTs != ts {
			metrics := []*CallTimeMetric{}
			for k, v := range callTimeMap {
				outKey := k
				outKey.Ts = lastTs
				metrics = append(metrics, &CallTimeMetric{
					Key:   outKey,
					Value: SerializeTDigest(v),
				})
				if !strings.HasSuffix(k.MetricName, "_integral") {
					delete(callTimeMap, k)
				}
			}
			if len(metrics) > 0 {
				cr.MetricsChan <- metrics
			}
			lastTs = ts
		}
		keys := []CallTimeMapKey{
			{
				MetricName:  "step_call",
				IsWholeCase: true,
				WorkerName:  cr.Info.WorkerName,
				CaseName:    cr.TestCase.Name,
				StepName:    "_NONE_",
				Success:     res.IsSuccess(),
				StatusCode:  res.GetResponseCode(),
				Ts:          0,
			},
			{
				MetricName:  "step_call_integral",
				IsWholeCase: true,
				WorkerName:  cr.Info.WorkerName,
				CaseName:    cr.TestCase.Name,
				StepName:    "_NONE_",
				Success:     res.IsSuccess(),
				StatusCode:  res.GetResponseCode(),
				Ts:          0,
			},
			{
				MetricName:  "step_call",
				IsWholeCase: false,
				WorkerName:  cr.Info.WorkerName,
				CaseName:    cr.TestCase.Name,
				StepName:    res.GetName(),
				Success:     res.IsSuccess(),
				StatusCode:  res.GetResponseCode(),
				Ts:          0,
			},
			{
				MetricName:  "step_call_integral",
				IsWholeCase: false,
				WorkerName:  cr.Info.WorkerName,
				CaseName:    cr.TestCase.Name,
				StepName:    res.GetName(),
				Success:     res.IsSuccess(),
				StatusCode:  res.GetResponseCode(),
				Ts:          0,
			},
		}
		for _, key := range keys {
			v := callTimeMap[key]
			if v == nil {
				v, _ = tdigest.New()
				callTimeMap[key] = v
			}
			v.Add(float64(res.GetEndTime() - res.GetBeginTime()))
		}
	}

	nowts := int(time.Now().UnixMilli() / 1000 / 60)
	metrics := []*CallTimeMetric{}
	for k, v := range callTimeMap {
		outKey := k
		outKey.Ts = nowts
		metrics = append(metrics, &CallTimeMetric{
			Key:   outKey,
			Value: SerializeTDigest(v),
		})
		delete(callTimeMap, k)
	}
	if len(metrics) > 0 {
		cr.MetricsChan <- metrics
	}
}

func (cr *CaseRunner) SendMetrics() {
	for metrics := range cr.MetricsChan {
		targetUrl := fmt.Sprintf("%v/worker/send_step_metrics", cr.CoordinatorApi)
		if err := cr.httpClient.PostJSON(targetUrl, metrics, nil); err != nil {
			fmt.Println("Error sending metrics: " + err.Error())
		}
	}
}
