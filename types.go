package workerclient

type ResponseBody struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

type RspWorkerPushStatusBody struct {
	Code int                  `json:"code"`
	Data *RspWorkerPushStatus `json:"data"`
	Msg  string               `json:"msg"`
}

type RspWorkerPushStatus struct {
	Worker         *Worker       `json:"worker"`
	ShouldRunCase  bool          `json:"shouldRunCase"`
	ShouldStopCase bool          `json:"shouldStopCase"`
	TestCaseInfo   *TestCaseInfo `json:"testCase"`
}

type CaseBaseInfo struct {
	Name                 string            `json:"name" binding:"required"`
	GlobalParams         map[string]string `json:"globalParams" binding:"required"`
	TotalMaxConcurrency  uint64            `json:"totalMaxConcurrency" binding:"required"`
	RampingSeconds       uint64            `json:"rampingSeconds" binding:"required"`
	DurationMinutes      uint64            `json:"durationMinutes"  binding:"required"`
	WorkName             string            `json:"workName" binding:"required"`
	WorkerConcurrency     uint64            `json:"workerConcurrency" binding:"required"`
	TaskId               string            `json:"taskId"`
}

type TestCaseInfo struct {
	BaseInfo           *CaseBaseInfo `json:"baseInfo"`
	WorkerTotal        uint64        `json:"workerTotal" binding:"optional"`
	RunningWorkerCount uint64        `json:"runningWorkerCount" binding:"optional"`
	RuningWorkerIds    []string      `json:"runningWorkerIds"`
	Status             string        `json:"status" binding:"optional"`
	BeginTime          uint64        `json:"beginTime"`
	LastTime           uint64        `json:"lastTime"`
	Summary            *CaseSummary  `json:"summary" binding:"optional"`
}

type CaseSummary struct {
	CallMonitors         map[string]*CallMonitor `json:"callMonitor" binding:"optional"`
	LastConcurrencyCount uint64                  `json:"lastConcurrencyCount"`
}

type CallMonitor struct {
	TotalCount uint64 `json:"totalCount"`
	TotalRt    uint64 `json:"totalRt"` // unit: millisecond
	MaxRt      uint64 `json:"maxRt"`
	MinRt      uint64 `json:"minRt"`
	SuccCount  uint64 `json:"succCount"`
	FailCount  uint64 `json:"failCount"`
	BeginTime  uint64 `json:"beginTime"`
	LastTime   uint64 `json:"lastTime"`
}

type TestCaseSummary struct {
	Name                   string `json:"name" binding:"required"`
	Status                 string `json:"status" binding:"required"`
	ActiveConcurrencyCount int64  `json:"activeConcurrencyCount"`
	TaskId                 string `json:"taskId"`
}

type Worker struct {
	BaseInfo    *WorkerBaseInfo `json:"baseInfo" binding:"required"`
	LastAciveAt int64           `json:"lastAciveAt"`
}

type WorkerBaseInfo struct {
	Name      string             `json:"name" binding:"required"`
	ID        string             `json:"id" binding:"required"`
	Index     int64              `json:"index"`
	Status    string             `json:"status" binding:"required"`
	TestCases []*TestCaseSummary `json:"testCases"`
}

type WorkerPushStatusParams struct {
	BaseInfo *WorkerBaseInfo `json:"baseInfo" binding:"required"`
}

type CallTimeMapKey struct {
	TaskId      string `json:"taskId"`
	MetricName  string `json:"metricName"`
	IsWholeCase bool   `json:"isWholeCase"`
	WorkerName  string `json:"workerName"`
	CaseName    string `json:"caseName"`
	StepName    string `json:"stepName"`
	Success     bool   `json:"success"`
	StatusCode  int    `json:"statusCode"`
	Ts          int    `json:"ts"`
}

type CallTimeMetric struct {
	Key   CallTimeMapKey `json:"key"`
	Value []TDNode       `json:"value"`
}
