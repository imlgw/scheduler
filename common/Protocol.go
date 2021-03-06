package common

import (
	"context"
	"encoding/json"
	"github.com/gorhill/cronexpr"
	"strings"
	"time"
)

//定时任务
type Job struct {
	Name     string `json:"name"`     //任务名
	Command  string `json:"command"`  //shell命令
	CronExpr string `json:"cronExpr"` //cron表达式
}

//任务调度计划
type JobSchedulerPlan struct {
	Job      *Job                 //要调度的任务信息
	Expr     *cronexpr.Expression //解析过的cronEXpr表达式
	NextTime time.Time            //下次调度的时间
}

//任务执行状态
type JobExecuteInfo struct {
	Job        *Job
	PlanTime   time.Time //理论上调度时间
	RealTime   time.Time //实际的调度时间
	CancelCtx  context.Context
	CancelFunc context.CancelFunc
}

//HTTP接口的应答消息
type Response struct {
	ErrorNo int         `json:"errno"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data"`
}

//变化事件
type JobEvent struct {
	EventType int
	Job       *Job
}

//任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo //执行状态
	Output      []byte          //shell的输出
	Err         error           //脚本错误原因
	StartTime   time.Time       //启动时间
	EndTime     time.Time       //结束时间
}

//优化点：日志以批次的形式发送去存储
type LogBatch struct {
	Logs []*JobLog
}

//任务日志
type JobLog struct {
	ID           int    `gorm:"AUTO_INCREMENT;primary_key;not null" json:"id"` //主键
	JobName      string `gorm:"type:varchar(100)" json:"jobName"`              //任务名                                            //任务名
	Command      string `gorm:"type:text" json:"command"`                      //脚本命令
	Err          string `gorm:"type:varchar(3000)" json:"err"`                 //错误原因
	Output       string `gorm:"type:varchar(3000)" json:"output"`              //脚本输出
	PlanTime     int64  `gorm:"type:bigint(20)" json:"planTime"`               //计划调度时间
	ScheduleTime int64  `gorm:"type:bigint(20)" json:"scheduleTime"`           //实际调度时间
	StartTime    int64  `gorm:"type:bigint(20)" json:"startTime"`              //任务开始指向时间
	EndTime      int64  `gorm:"type:bigint(20)" json:"endTime"`                //任务结束时间
}

func BuildResp(errno int, msg string, data interface{}) ([]byte, error) {
	var (
		err      error
		respObj  *Response
		response []byte
	)
	respObj = &Response{
		ErrorNo: errno,
		Msg:     msg,
		Data:    data,
	}
	if response, err = json.Marshal(respObj); err != nil {
		return nil, err
	}
	return response, err
}

func UnmarshalJob(data []byte) (*Job, error) {
	var job = &Job{}
	if err := json.Unmarshal(data, job); err != nil {
		return nil, err
	}
	return job, nil
}

//脱去目录前缀，获得最后的内容
//删除任务目录前缀，获取任务名
func StripDir(dirPrefix string, jobKey string) string {
	return strings.TrimPrefix(jobKey, dirPrefix)
}

//构建Event 1) 更新任务 2)删除任务
func BuildJobEvent(eventType int, job *Job) *JobEvent {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

func BuildJobSchedulePlan(job *Job) (*JobSchedulerPlan, error) {
	var (
		err  error
		expr *cronexpr.Expression
	)
	if expr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return nil, err
	}
	return &JobSchedulerPlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}, err
}

func BuildJobExecuteInfo(plan *JobSchedulerPlan) *JobExecuteInfo {
	info := &JobExecuteInfo{
		Job:      plan.Job,
		PlanTime: plan.NextTime, //计算调度时间
		RealTime: time.Now(),    //真实执行时间
	}
	//增加上下文，用于cancel任务
	info.CancelCtx, info.CancelFunc = context.WithCancel(context.TODO())
	return info
}
