package pamodels

import (
	"fmt"
	"sync"
)

func init() {
	paTaskBodyPool = &sync.Pool{
		New: func() interface{} {
			return &PaTaskBody{}
		},
	}

	paResBodyPool = &sync.Pool{
		New: func() interface{} {
			return &PaResBody{}
		},
	}

	paTaskResPool = &sync.Pool{
		New: func() interface{} {
			return &PaTaskRes{}
		},
	}
}

var paTaskBodyPool *sync.Pool

func AcquirePaTaskBody() *PaTaskBody {
	v := paTaskBodyPool.Get()
	if v == nil {
		return &PaTaskBody{}
	}

	ptaskBody := v.(*PaTaskBody)
	return ptaskBody
}

func ReleasePaTaskBody(ptaskBody *PaTaskBody) {

	ptaskBody.TaskId = ""

	if ptaskBody.TaskParam != nil {
		ptaskBody.TaskParam.Task_Param_SubTaskId = ""
		ptaskBody.TaskParam.Task_Param_Type = ""
		ptaskBody.TaskParam.Task_Param_WavAddr = ""
		ptaskBody.TaskParam.Task_Param_TopN = -1
		ptaskBody.TaskParam.Task_Param_SpkId = ""
		ptaskBody.TaskParam.Task_Param_Scene = ""
		ptaskBody.TaskParam.Task_Param_Channel = -1
		ptaskBody.TaskParam.Task_Param_Gender = ""
		ptaskBody.TaskParam.Task_Param_CutLen = -1
		ptaskBody.TaskParam.Task_Param_HasTone = false
		ptaskBody.TaskParam.Task_Param_WavAddr1 = ""
		ptaskBody.TaskParam.Task_Param_WavAddr2 = ""
		ptaskBody.TaskParam.Task_Param_Channel1 = -1
		ptaskBody.TaskParam.Task_Param_Channel2 = -1
		ptaskBody.TaskParam.Task_Param_RecordId = ""

		if ptaskBody.TaskParam.Task_Param_Nodes != nil {
			ptaskBody.TaskParam.Task_Param_Nodes = ptaskBody.TaskParam.Task_Param_Nodes[:0]
		}

		ptaskBody.TaskParam.Task_Param_EnrollNode = ""
		ptaskBody.TaskParam.Task_Param_EnrollFlag = false
		ptaskBody.TaskParam.Task_Param_OriginNode = ""
		if ptaskBody.TaskParam.Task_Param_TargetNode != nil {
			ptaskBody.TaskParam.Task_Param_TargetNode = ptaskBody.TaskParam.Task_Param_TargetNode[:0]
		}
	}
	ptaskBody.TaskAddTime = ""
	paTaskBodyPool.Put(ptaskBody)
}

var paTaskResPool *sync.Pool

func AcquirePaTaskRes() *PaTaskRes {
	v := paTaskResPool.Get()
	if v == nil {
		return &PaTaskRes{}
	}

	paTaskRes := v.(*PaTaskRes)
	return paTaskRes
}

func ReleasePaTaskRes(paTaskRes *PaTaskRes) {

	paTaskRes.Task_Res_Type = ""
	paTaskRes.Task_Res_SubTaskId = ""
	paTaskRes.Task_Res_Scene = ""

	if paTaskRes.Task_Res_ParamObj != nil {
		paTaskRes.Task_Res_ParamObj.Task_Param_SubTaskId = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_Type = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_WavAddr = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_TopN = -1
		paTaskRes.Task_Res_ParamObj.Task_Param_SpkId = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_Scene = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_Channel = -1
		paTaskRes.Task_Res_ParamObj.Task_Param_Gender = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_CutLen = -1
		paTaskRes.Task_Res_ParamObj.Task_Param_HasTone = false
		paTaskRes.Task_Res_ParamObj.Task_Param_Channel1 = -1
		paTaskRes.Task_Res_ParamObj.Task_Param_Channel2 = -1
		paTaskRes.Task_Res_ParamObj.Task_Param_WavAddr1 = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_WavAddr2 = ""

		if paTaskRes.Task_Res_ParamObj.Task_Param_Nodes != nil {
			paTaskRes.Task_Res_ParamObj.Task_Param_Nodes = paTaskRes.Task_Res_ParamObj.Task_Param_Nodes[:0]
		}
		paTaskRes.Task_Res_ParamObj.Task_Param_EnrollNode = ""
		paTaskRes.Task_Res_ParamObj.Task_Param_EnrollFlag = false
		paTaskRes.Task_Res_ParamObj.Task_Param_OriginNode = ""
		if paTaskRes.Task_Res_ParamObj.Task_Param_TargetNode != nil {
			paTaskRes.Task_Res_ParamObj.Task_Param_TargetNode = paTaskRes.Task_Res_ParamObj.Task_Param_TargetNode[:0]
		}
	}

	if paTaskRes.Task_Res_Results != nil {
		paTaskRes.Task_Res_Results = paTaskRes.Task_Res_Results[:0]
	}
	paTaskRes.Task_Res_ErrCode = -1
	paTaskRes.Task_Res_ErrMsg = ""
	paTaskRes.Task_Res_Addtime = ""

	paTaskResPool.Put(paTaskRes)
}

var paResBodyPool *sync.Pool

func AcquirePaResBody() *PaResBody {
	v := paResBodyPool.Get()
	if v == nil {
		return &PaResBody{}
	}

	PaResBody := v.(*PaResBody)
	return PaResBody
}

func ReleasePaResBody(paResBody *PaResBody) {

	paResBody.Task_Id = ""

	if paResBody.Task_Res != nil {
		paResBody.Task_Res.Task_Res_Addtime = ""
		paResBody.Task_Res.Task_Res_ErrCode = -1
		paResBody.Task_Res.Task_Res_ErrMsg = ""
		if paResBody.Task_Res.Task_Res_Results != nil {
			paResBody.Task_Res.Task_Res_Results = paResBody.Task_Res.Task_Res_Results[:0]
		}
		if paResBody.Task_Res.Task_Res_ParamObj != nil {
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_SubTaskId = ""
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_Type = ""
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_WavAddr = ""
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_TopN = -1
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_SpkId = ""
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_Scene = ""
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_Channel = -1
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_Gender = ""
			paResBody.Task_Res.Task_Res_ParamObj.Task_Param_CutLen = -1
			paResBody.Task_Res.Task_Res_Scene = ""
			paResBody.Task_Res.Task_Res_SubTaskId = ""
			paResBody.Task_Res.Task_Res_Type = ""
		}
	}
	paResBodyPool.Put(paResBody)
}

type PaTasksBody struct {
	TaskId      string         `json:"task_id"`       // 字符串，32位长度，全局唯一的任务ID
	TaskParams  []*PaTaskParam `json:"task_params"`   // PaTaskParam数组，任务参数
	TaskAddTime string         `json:"task_add_time"` // 字符串，任务插入时间
}

type PaTaskBody struct {
	TaskId      string       `json:"task_id"`       // 字符串，32位长度，全局唯一的任务ID
	TaskParam   *PaTaskParam `json:"task_param"`    // PaTaskParam，任务参数
	TaskAddTime string       `json:"task_add_time"` // 字符串，任务插入时间
}

func (ptb *PaTaskBody) SetTaskParam(tp *PaTaskParam) {
	ptb.TaskParam = tp
}

type PaTaskParam struct {
	// 通用参数
	Task_Param_SubTaskId string `json:"task_param_sub_task_id"` // 字符串，16位长度，语音ID
	Task_Param_Type      string `json:"task_param_type"`        // 字符串，任务类型

	// 注册、验证、辨认参数
	Task_Param_Scene    string `json:"task_param_scene"`    // 字符串，场景
	Task_Param_WavAddr  string `json:"task_param_wav_addr"` // 字符串，语音文件路径
	Task_Param_Channel  int    `json:"task_param_channel"`  // 整型，指定左右声道
	Task_Param_Gender   string `json:"task_param_gender"`   // 字符串，性别
	Task_Param_SpkId    string `json:"task_param_spkid"`    // 字符串，16位长度，说话人ID
	Task_Param_RecordId string `json:"task_param_record_id"`
	Task_Param_Source   string `json:"task_param_source"`
	Task_Param_TopN     int    `json:"task_param_top_n"` // 整数，Top N

	//Task_Param_Version    string   `json:"task_param_version"` // verify, identify 比对语音库节点版本号
	Task_Param_Nodes      []string `json:"task_param_nodes"` // verify, identify 比对语音库节点
	Task_Param_EnrollNode string   `json:"task_param_enroll_node"`
	Task_Param_DeleteNode string   `json:"task_param_delete_node"`
	Task_Param_OriginNode string   `json:"task_param_origin_node"`
	Task_Param_TargetNode []string `json:"task_param_target_node"`
	Task_Param_EnrollFlag bool     `json:"task_param_enroll_flag"` // Identify时，是否同时还需要注册入库

	Task_Param_WavAddr1 string `json:"task_param_wav_addr_1"`
	Task_Param_WavAddr2 string `json:"task_param_wav_addr_2"`
	Task_Param_Channel1 int    `json:"task_param_channel_1"`
	Task_Param_Channel2 int    `json:"task_param_channel_2"`

	// 其他
	Task_Param_CutLen  int64 `json:"task_param_cut_len"`  // 长整型，需要CUT掉的语音采样点，从头开始算
	Task_Param_HasTone bool  `json:"task_param_has_tone"` // 布尔型，是否有tone音
}

func (p *PaTaskParam) Copy() *PaTaskParam {
	newTaskParam := new(PaTaskParam)

	newTaskParam.Task_Param_SubTaskId = p.Task_Param_SubTaskId
	newTaskParam.Task_Param_Type = p.Task_Param_Type
	// 注册、验证、辨认参数
	newTaskParam.Task_Param_Scene = p.Task_Param_Scene
	newTaskParam.Task_Param_WavAddr = p.Task_Param_WavAddr
	newTaskParam.Task_Param_Channel = p.Task_Param_Channel
	newTaskParam.Task_Param_Gender = p.Task_Param_Gender
	newTaskParam.Task_Param_SpkId = p.Task_Param_SpkId
	newTaskParam.Task_Param_RecordId = p.Task_Param_RecordId
	newTaskParam.Task_Param_Source = p.Task_Param_Source
	newTaskParam.Task_Param_TopN = p.Task_Param_TopN
	//Task_Param_Version
	newTaskParam.Task_Param_Nodes = make([]string, len(p.Task_Param_Nodes), len(p.Task_Param_Nodes))
	for i := 0; i < len(p.Task_Param_Nodes); i++ {
		newTaskParam.Task_Param_Nodes[i] = p.Task_Param_Nodes[i]
	}

	newTaskParam.Task_Param_EnrollNode = p.Task_Param_EnrollNode
	newTaskParam.Task_Param_DeleteNode = p.Task_Param_DeleteNode
	newTaskParam.Task_Param_OriginNode = p.Task_Param_OriginNode
	newTaskParam.Task_Param_TargetNode = make([]string, len(p.Task_Param_TargetNode), len(p.Task_Param_TargetNode))
	for i := 0; i < len(p.Task_Param_TargetNode); i++ {
		newTaskParam.Task_Param_TargetNode[i] = p.Task_Param_TargetNode[i]
	}
	newTaskParam.Task_Param_EnrollFlag = p.Task_Param_EnrollFlag
	newTaskParam.Task_Param_WavAddr1 = p.Task_Param_WavAddr1
	newTaskParam.Task_Param_WavAddr2 = p.Task_Param_WavAddr2
	newTaskParam.Task_Param_Channel1 = p.Task_Param_Channel1
	newTaskParam.Task_Param_Channel2 = p.Task_Param_Channel2
	// 其他
	newTaskParam.Task_Param_CutLen = p.Task_Param_CutLen
	newTaskParam.Task_Param_HasTone = p.Task_Param_HasTone

	return newTaskParam
}

type PaResBody struct {
	Task_Id  string     `json:"task_id"`  // 字符串，32位长度，全局唯一的任务ID
	Task_Res *PaTaskRes `json:"task_res"` // PaRespParam对象，任务处理结果
}

func (p *PaResBody) String() string {
	return fmt.Sprintf("Task id:%s, task resp: %s", p.Task_Id, p.Task_Res)
}

type PaTaskRes struct {
	// 通用参数
	Task_Res_SubTaskId string       `json:"task_res_subtaskid"` // 字符串，16位长度，语音ID
	Task_Res_Type      string       `json:"task_res_type"`      // 字符串，任务类型
	Task_Res_ParamObj  *PaTaskParam `json:"task_res_paramobj"`  // PaTaskParam对象，请求参数

	Task_Res_ErrCode int    `json:"task_res_errcode"` // 整型，错误码
	Task_Res_ErrMsg  string `json:"task_res_errmsg"`  // 字符串，错误消息
	Task_Res_Addtime string `json:"task_res_addtime"` // 字符串，任务插入时间

	// 其他
	Task_Res_Scene   string       `json:"task_res_scene"`   // 字符串，场景
	Task_Res_Results []*PaProcRes `json:"task_res_results"` // PaProcRes对象，处理结果
}

func (p PaTaskRes) String() string {
	results := ""
	for _, res := range p.Task_Res_Results {
		results += fmt.Sprintf("proc_errcode: %d, proc_errmsg: %s, proc_spkid: %s, proc_confidence: %.02f, proc_candidates： %v",
			res.Task_Proc_ErrCode, res.Task_Proc_ErrMsg, res.Task_Proc_SpkId,
			res.Task_Proc_Confidence, res.Task_Proc_Candidates)
	}

	return fmt.Sprintf("subid: %s, type: %s, taskParam: %v, code: %d, msg: %s, results: [%s]",
		p.Task_Res_SubTaskId, p.Task_Res_Type, p.Task_Res_ParamObj, p.Task_Res_ErrCode,
		p.Task_Res_ErrMsg, results,
	)
}

type PaProcRes struct {
	Task_Proc_Chan int `json:"task_proc_chan"` // 整型，信道标记，0：左声道；1：右声道

	Task_Proc_Utt           string  `json:"task_proc_utt"`           // 字符串，语音路径
	Task_Proc_Duration      float64 `json:"task_proc_duration"`      // 浮点型，语音时长，以秒为单位
	Task_Proc_ValidDuration float64 `json:"task_proc_validduration"` // 浮点型，语音有效时长，以秒为单位
	Task_Proc_CheckSum      string  `json:"task_proc_checksum"`      // 字符串，checksum

	Task_Proc_ErrCode int    `json:"task_proc_errcode"` // 整型，错误码
	Task_Proc_ErrMsg  string `json:"task_proc_errmsg"`  // 字符串，错误消息

	// 说话人辨认、验证
	Task_Proc_Top        int                    `json:"task_proc_top"`        // 整型，候选声纹集数目
	Task_Proc_Candidates []*PaIdentifyCandidate `json:"task_proc_candidates"` // Json数组，候选集

	// 说话人注册
	Task_Proc_SpkId      string  `json:"task_proc_spkid"`      // 字符串，16位长度，说话人ID
	Task_Proc_Confidence float32 `json:"task_proc_confidence"` // 得分

	// 注册
	Task_Proc_FeatureFile []byte `json:"task_proc_featurefile"` //声纹特征文件 返回二进制流
}

type PaIdentifyCandidate struct {
	Identify_SpkId      string  `json:"identify_spkid"`      // 字符串，16位长度，说话人ID
	Identify_Confidence float32 `json:"identify_confidence"` // 浮点数，置信度
	Identify_Node       string  `json:"identify_node"`       // 候选者所在node节点
}

//////////////////////////////////////////////////////////////////////////////////////////

// @Generate PaTaskBody instance
//////////////////////////////////////////////////////////////////
func NewPaTaskBody() *PaTaskBody {
	return &PaTaskBody{}
}

// @Generate PaResBody instance
//////////////////////////////////////////////////////////////////
func NewPaResBody() *PaResBody {
	return &PaResBody{}
}

func (this *PaResBody) SetTaskId(task_id string) {
	this.Task_Id = task_id
}

func (this *PaResBody) SetTaskRes(res *PaTaskRes) {
	this.Task_Res = res
}

//////////////////////////////////////////////////////////////////

// @Generate TaskRes instance
//////////////////////////////////////////////////////////////////
func NewPaTaskRes() *PaTaskRes {
	return &PaTaskRes{
		Task_Res_Results: make([]*PaProcRes, 0),
	}
}

func (this *PaTaskRes) SetSubTaskId(task_id string) {
	this.Task_Res_SubTaskId = task_id
}

func (this *PaTaskRes) SetTaskResType(task_type string) {
	this.Task_Res_Type = task_type
}

func (this *PaTaskRes) SetTaskResParamObj(param *PaTaskParam) {
	this.Task_Res_ParamObj = param
}

func (this *PaTaskRes) SetTaskResErrCode(code int) {
	this.Task_Res_ErrCode = code
}

func (this *PaTaskRes) SetTaskResErrMsg(msg string) {
	this.Task_Res_ErrMsg = msg
}

func (this *PaTaskRes) SetTaskResAddTime(addtime string) {
	this.Task_Res_Addtime = addtime
}

func (this *PaTaskRes) SetTaskResScene(scene string) {
	this.Task_Res_Scene = scene
}

func (this *PaTaskRes) AddTaskResResult(res *PaProcRes) {
	this.Task_Res_Results = append(this.Task_Res_Results, res)
}

//////////////////////////////////////////////////////////////////

// @Generate PaProcRes instance
//////////////////////////////////////////////////////////////////
func NewPaProcRes() *PaProcRes {
	return &PaProcRes{
		Task_Proc_Candidates: make([]*PaIdentifyCandidate, 0),
	}
}

func (this *PaProcRes) SetTaskProcChan(chn int) {
	this.Task_Proc_Chan = chn
}

func (this *PaProcRes) SetTaskProcUtt(utt string) {
	this.Task_Proc_Utt = utt
}

func (this *PaProcRes) SetTaskProcDuration(duration float64) {
	this.Task_Proc_Duration = duration
}

func (this *PaProcRes) SetTaskProcValidDuration(valid_duration float64) {
	this.Task_Proc_ValidDuration = valid_duration
}

func (this *PaProcRes) SetTaskProcCheckSum(checksum string) {
	this.Task_Proc_CheckSum = checksum
}

func (this *PaProcRes) SetTaskProcErrCode(err_code int) {
	this.Task_Proc_ErrCode = err_code
}

func (this *PaProcRes) SetTaskProcErrMsg(err_msg string) {
	this.Task_Proc_ErrMsg = err_msg
}

func (this *PaProcRes) SetTaskProcTop(top int) {
	this.Task_Proc_Top = top
}

func (this *PaProcRes) SetTaskProcSpkId(spkid string) {
	this.Task_Proc_SpkId = spkid
}

func (this *PaProcRes) SetTaskProcConfidence(confidence float32) {
	this.Task_Proc_Confidence = confidence
}

func (this *PaProcRes) SetTaskProcFeatureFile(featureFile []byte) {
	this.Task_Proc_FeatureFile = featureFile
}

func (this *PaProcRes) SetTaskProcCandidates(candidates []*PaIdentifyCandidate) {
	this.Task_Proc_Candidates = append(this.Task_Proc_Candidates, candidates...)
}

func (this *PaProcRes) AddCandidate(candidate *PaIdentifyCandidate) {
	this.Task_Proc_Candidates = append(this.Task_Proc_Candidates, candidate)
}

func NewPaIdentifyCandidate() *PaIdentifyCandidate {
	return &PaIdentifyCandidate{}
}

func (this *PaIdentifyCandidate) SetSpkId(spkid string) {
	this.Identify_SpkId = spkid
}

func (this *PaIdentifyCandidate) SetConfidence(confidence float32) {
	this.Identify_Confidence = confidence
}

func (this *PaIdentifyCandidate) SetSpkIdAndConfidence(spkid string, confidence float32, node string) {
	this.Identify_SpkId = spkid
	this.Identify_Confidence = confidence
	this.Identify_Node = node
}
