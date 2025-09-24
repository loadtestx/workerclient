package workerclient

import (
	"fmt"
	"time"
)

const (
	InnerVarName          = "__name"
	InnerVarGoroutineId   = "__goroutine_id"
	InnerVarExecutorIndex = "__executor_index"
	InnerVarWorkerTotal   = "__worker_total"
	InnerVarWorkerIndex   = "__worker_index"
	InnerVarWorkerSize    = "__worker_size"
)

type IResultV1 interface {
	GetName() string
	GetUrl() string
	GetMethod() string
	GetRequestHeader() map[string]string
	GetRequestBody() string
	GetSentBytes() int
	GetResponseCode() int
	GetResponseHeader() map[string]string
	GetResponseBody() string
	GetReceivedBytes() int
	GetFailureMessage() string
	IsSuccess() bool
	GetBeginTime() int64
	GetEndTime() int64
	GetSubResults() []interface{}
}

func AcquireResult(name string) *Result {
	result := &Result{}
	result.Name = name
	result.RequestHeader = map[string]string{}
	result.ResponseHeader = map[string]string{}
	result.ResponseCode = 0
	result.Success = true
	result.BeginTime = time.Now().UnixMilli()
	result.EndTime = time.Now().UnixMilli()
	return result
}

type Result struct {
	Name           string
	Url            string
	Method         string
	RequestHeader  map[string]string
	RequestBody    string
	SentBytes      int
	ResponseCode   int
	ResponseHeader map[string]string
	ResponseBody   string
	ReceivedBytes  int
	FailureMessage string
	Success        bool
	BeginTime      int64
	EndTime        int64
	SubResults     []interface{}
	SubIndex       int
}

func (r *Result) GetName() string {
	return r.Name
}

func (r *Result) GetUrl() string {
	return r.Url
}

func (r *Result) GetMethod() string {
	return r.Method
}

func (r *Result) GetRequestHeader() map[string]string {
	return r.RequestHeader
}

func (r *Result) GetRequestBody() string {
	return r.RequestBody
}

func (r *Result) GetSentBytes() int {
	return r.SentBytes
}

func (r *Result) GetResponseCode() int {
	return r.ResponseCode
}

func (r *Result) GetResponseHeader() map[string]string {
	return r.ResponseHeader
}

func (r *Result) GetResponseBody() string {
	return r.ResponseBody
}

func (r *Result) GetReceivedBytes() int {
	return r.ReceivedBytes
}

func (r *Result) GetFailureMessage() string {
	return r.FailureMessage
}

func (r *Result) IsSuccess() bool {
	return r.Success
}

func (r *Result) GetBeginTime() int64 {
	return r.BeginTime
}

func (r *Result) GetEndTime() int64 {
	return r.EndTime
}

func (r *Result) GetSubResults() []interface{} {
	return r.SubResults
}

// begin records begin time, do not forget call this function to update
func (r *Result) Begin() {
	r.BeginTime = time.Now().UnixMilli()
}

func (r *Result) End() {
	r.Success = r.ResponseCode == 200
	r.EndTime = time.Now().UnixMilli()
}

func (r *Result) AddSub(name string, useNamePrefix bool) *Result {
	if name == "" {
		name = fmt.Sprintf("%s-%d", r.Name, r.SubIndex)
		r.SubIndex++
	} else {
		if useNamePrefix {
			name = fmt.Sprintf("%s-%s", r.Name, name)
		}
	}
	sub := AcquireResult(name)
	r.SubResults = append(r.SubResults, sub)
	return sub
}
