package pamodels

import (
	"bytes"
	json "encoding/json"
	easyjson "github.com/mailru/easyjson"
	jlexer "github.com/mailru/easyjson/jlexer"
	jwriter "github.com/mailru/easyjson/jwriter"
	"sync"
)

// suppress unused package warning
var (
	_ *json.RawMessage
	_ *jlexer.Lexer
	_ *jwriter.Writer
	_ easyjson.Marshaler
)

var writerPool sync.Pool

func acquireWriter() *jwriter.Writer {
	v := writerPool.Get()
	if v == nil {
		return &jwriter.Writer{}
	}
	wr := v.(*jwriter.Writer)
	return wr
}

func releaseWriter(wr *jwriter.Writer) {
	writerPool.Put(wr)
}

func easyjson7459bf99DecodeModels(in *jlexer.Lexer, out *PaTasksBody) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "task_id":
			out.TaskId = string(in.String())
		case "task_params":
			if in.IsNull() {
				in.Skip()
				out.TaskParams = nil
			} else {
				in.Delim('[')
				if out.TaskParams == nil {
					if !in.IsDelim(']') {
						out.TaskParams = make([]*PaTaskParam, 0, 8)
					} else {
						out.TaskParams = []*PaTaskParam{}
					}
				} else {
					out.TaskParams = (out.TaskParams)[:0]
				}
				for !in.IsDelim(']') {
					var v1 *PaTaskParam
					if in.IsNull() {
						in.Skip()
						v1 = nil
					} else {
						if v1 == nil {
							v1 = new(PaTaskParam)
						}
						(*v1).UnmarshalEasyJSON(in)
					}
					out.TaskParams = append(out.TaskParams, v1)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "task_add_time":
			out.TaskAddTime = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson7459bf99EncodeModels(out *jwriter.Writer, in PaTasksBody) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"task_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.TaskId))
	}
	{
		const prefix string = ",\"task_params\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.TaskParams == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v2, v3 := range in.TaskParams {
				if v2 > 0 {
					out.RawByte(',')
				}
				if v3 == nil {
					out.RawString("null")
				} else {
					(*v3).MarshalEasyJSON(out)
				}
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"task_add_time\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.TaskAddTime))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v PaTasksBody) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson7459bf99EncodeModels(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v PaTasksBody) MarshalJSONToByteBuffer(buf *bytes.Buffer) ([]byte, error) {
	w := acquireWriter()
	easyjson7459bf99EncodeModels(w, v)
	_, err := w.DumpTo(buf)
	releaseWriter(w)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v PaTasksBody) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson7459bf99EncodeModels(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *PaTasksBody) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson7459bf99DecodeModels(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *PaTasksBody) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson7459bf99DecodeModels(l, v)
}
func easyjson7459bf99DecodeModels1(in *jlexer.Lexer, out *PaTaskRes) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "task_res_subtaskid":
			out.Task_Res_SubTaskId = string(in.String())
		case "task_res_type":
			out.Task_Res_Type = string(in.String())
		case "task_res_paramobj":
			(out.Task_Res_ParamObj).UnmarshalEasyJSON(in)
		case "task_res_errcode":
			out.Task_Res_ErrCode = int(in.Int())
		case "task_res_errmsg":
			out.Task_Res_ErrMsg = string(in.String())
		case "task_res_addtime":
			out.Task_Res_Addtime = string(in.String())
		case "task_res_scene":
			out.Task_Res_Scene = string(in.String())
		case "task_res_results":
			if in.IsNull() {
				in.Skip()
				out.Task_Res_Results = nil
			} else {
				in.Delim('[')
				if out.Task_Res_Results == nil {
					if !in.IsDelim(']') {
						out.Task_Res_Results = make([]*PaProcRes, 0, 8)
					} else {
						out.Task_Res_Results = []*PaProcRes{}
					}
				} else {
					out.Task_Res_Results = (out.Task_Res_Results)[:0]
				}
				for !in.IsDelim(']') {
					var v4 *PaProcRes
					if in.IsNull() {
						in.Skip()
						v4 = nil
					} else {
						if v4 == nil {
							v4 = new(PaProcRes)
						}
						(*v4).UnmarshalEasyJSON(in)
					}
					out.Task_Res_Results = append(out.Task_Res_Results, v4)
					in.WantComma()
				}
				in.Delim(']')
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson7459bf99EncodeModels1(out *jwriter.Writer, in PaTaskRes) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"task_res_subtaskid\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Res_SubTaskId))
	}
	{
		const prefix string = ",\"task_res_type\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Res_Type))
	}
	{
		const prefix string = ",\"task_res_paramobj\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		(in.Task_Res_ParamObj).MarshalEasyJSON(out)
	}
	{
		const prefix string = ",\"task_res_errcode\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Res_ErrCode))
	}
	{
		const prefix string = ",\"task_res_errmsg\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Res_ErrMsg))
	}
	{
		const prefix string = ",\"task_res_addtime\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Res_Addtime))
	}
	{
		const prefix string = ",\"task_res_scene\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Res_Scene))
	}
	{
		const prefix string = ",\"task_res_results\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Task_Res_Results == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v5, v6 := range in.Task_Res_Results {
				if v5 > 0 {
					out.RawByte(',')
				}
				if v6 == nil {
					out.RawString("null")
				} else {
					(*v6).MarshalEasyJSON(out)
				}
			}
			out.RawByte(']')
		}
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v PaTaskRes) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson7459bf99EncodeModels1(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v PaTaskRes) MarshalJSONToByteBuffer(buf *bytes.Buffer) ([]byte, error) {
	w := acquireWriter()
	easyjson7459bf99EncodeModels1(w, v)
	_, err := w.DumpTo(buf)
	releaseWriter(w)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), w.Error
}


// MarshalEasyJSON supports easyjson.Marshaler interface
func (v PaTaskRes) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson7459bf99EncodeModels1(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *PaTaskRes) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson7459bf99DecodeModels1(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *PaTaskRes) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson7459bf99DecodeModels1(l, v)
}
func easyjson7459bf99DecodeModels2(in *jlexer.Lexer, out *PaTaskParam) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "task_param_sub_task_id":
			out.Task_Param_SubTaskId = string(in.String())
		case "task_param_type":
			out.Task_Param_Type = string(in.String())
		case "task_param_scene":
			out.Task_Param_Scene = string(in.String())
		case "task_param_wav_addr":
			out.Task_Param_WavAddr = string(in.String())
		case "task_param_channel":
			out.Task_Param_Channel = int(in.Int())
		case "task_param_gender":
			out.Task_Param_Gender = string(in.String())
		case "task_param_spkid":
			out.Task_Param_SpkId = string(in.String())
		case "task_param_record_id":
			out.Task_Param_RecordId = string(in.String())
		case "task_param_source":
			out.Task_Param_Source = string(in.String())
		case "task_param_top_n":
			out.Task_Param_TopN = int(in.Int())
		case "task_param_nodes":
			if in.IsNull() {
				in.Skip()
				out.Task_Param_Nodes = nil
			} else {
				in.Delim('[')
				if out.Task_Param_Nodes == nil {
					if !in.IsDelim(']') {
						out.Task_Param_Nodes = make([]string, 0, 4)
					} else {
						out.Task_Param_Nodes = []string{}
					}
				} else {
					out.Task_Param_Nodes = (out.Task_Param_Nodes)[:0]
				}
				for !in.IsDelim(']') {
					var v7 string
					v7 = string(in.String())
					out.Task_Param_Nodes = append(out.Task_Param_Nodes, v7)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "task_param_enroll_node":
			out.Task_Param_EnrollNode = string(in.String())
		case "task_param_delete_node":
			out.Task_Param_DeleteNode = string(in.String())
		case "task_param_origin_node":
			out.Task_Param_OriginNode = string(in.String())
		case "task_param_target_node":
			if in.IsNull() {
				in.Skip()
				out.Task_Param_TargetNode = nil
			} else {
				in.Delim('[')
				if out.Task_Param_TargetNode == nil {
					if !in.IsDelim(']') {
						out.Task_Param_TargetNode = make([]string, 0, 4)
					} else {
						out.Task_Param_TargetNode = []string{}
					}
				} else {
					out.Task_Param_TargetNode = (out.Task_Param_TargetNode)[:0]
				}
				for !in.IsDelim(']') {
					var v8 string
					v8 = string(in.String())
					out.Task_Param_TargetNode = append(out.Task_Param_TargetNode, v8)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "task_param_enroll_flag":
			out.Task_Param_EnrollFlag = bool(in.Bool())
		case "task_param_wav_addr_1":
			out.Task_Param_WavAddr1 = string(in.String())
		case "task_param_wav_addr_2":
			out.Task_Param_WavAddr2 = string(in.String())
		case "task_param_channel_1":
			out.Task_Param_Channel1 = int(in.Int())
		case "task_param_channel_2":
			out.Task_Param_Channel2 = int(in.Int())
		case "task_param_cut_len":
			out.Task_Param_CutLen = int64(in.Int64())
		case "task_param_has_tone":
			out.Task_Param_HasTone = bool(in.Bool())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson7459bf99EncodeModels2(out *jwriter.Writer, in PaTaskParam) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"task_param_sub_task_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_SubTaskId))
	}
	{
		const prefix string = ",\"task_param_type\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_Type))
	}
	{
		const prefix string = ",\"task_param_scene\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_Scene))
	}
	{
		const prefix string = ",\"task_param_wav_addr\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_WavAddr))
	}
	{
		const prefix string = ",\"task_param_channel\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Param_Channel))
	}
	{
		const prefix string = ",\"task_param_gender\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_Gender))
	}
	{
		const prefix string = ",\"task_param_spkid\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_SpkId))
	}
	{
		const prefix string = ",\"task_param_record_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_RecordId))
	}
	{
		const prefix string = ",\"task_param_source\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_Source))
	}
	{
		const prefix string = ",\"task_param_top_n\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Param_TopN))
	}
	{
		const prefix string = ",\"task_param_nodes\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Task_Param_Nodes == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v9, v10 := range in.Task_Param_Nodes {
				if v9 > 0 {
					out.RawByte(',')
				}
				out.String(string(v10))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"task_param_enroll_node\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_EnrollNode))
	}
	{
		const prefix string = ",\"task_param_delete_node\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_DeleteNode))
	}
	{
		const prefix string = ",\"task_param_origin_node\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_OriginNode))
	}
	{
		const prefix string = ",\"task_param_target_node\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Task_Param_TargetNode == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v11, v12 := range in.Task_Param_TargetNode {
				if v11 > 0 {
					out.RawByte(',')
				}
				out.String(string(v12))
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"task_param_enroll_flag\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Task_Param_EnrollFlag))
	}
	{
		const prefix string = ",\"task_param_wav_addr_1\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_WavAddr1))
	}
	{
		const prefix string = ",\"task_param_wav_addr_2\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Param_WavAddr2))
	}
	{
		const prefix string = ",\"task_param_channel_1\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Param_Channel1))
	}
	{
		const prefix string = ",\"task_param_channel_2\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Param_Channel2))
	}
	{
		const prefix string = ",\"task_param_cut_len\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int64(int64(in.Task_Param_CutLen))
	}
	{
		const prefix string = ",\"task_param_has_tone\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Bool(bool(in.Task_Param_HasTone))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v PaTaskParam) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson7459bf99EncodeModels2(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v PaTaskParam) MarshalJSONToByteBuffer(buf *bytes.Buffer) ([]byte, error) {
	w := acquireWriter()
	easyjson7459bf99EncodeModels2(w, v)
	_, err := w.DumpTo(buf)
	releaseWriter(w)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v PaTaskParam) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson7459bf99EncodeModels2(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *PaTaskParam) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson7459bf99DecodeModels2(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *PaTaskParam) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson7459bf99DecodeModels2(l, v)
}
func easyjson7459bf99DecodeModels3(in *jlexer.Lexer, out *PaTaskBody) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "task_id":
			out.TaskId = string(in.String())
		case "task_param":
			if in.IsNull() {
				in.Skip()
				out.TaskParam = nil
			} else {
				if out.TaskParam == nil {
					out.TaskParam = new(PaTaskParam)
				}
				(*out.TaskParam).UnmarshalEasyJSON(in)
			}
		case "task_add_time":
			out.TaskAddTime = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson7459bf99EncodeModels3(out *jwriter.Writer, in PaTaskBody) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"task_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.TaskId))
	}
	{
		const prefix string = ",\"task_param\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.TaskParam == nil {
			out.RawString("null")
		} else {
			(*in.TaskParam).MarshalEasyJSON(out)
		}
	}
	{
		const prefix string = ",\"task_add_time\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.TaskAddTime))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v PaTaskBody) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson7459bf99EncodeModels3(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v PaTaskBody) MarshalJSONToByteBuffer(buf *bytes.Buffer) ([]byte, error) {
	w := acquireWriter()
	easyjson7459bf99EncodeModels3(w, v)
	_, err := w.DumpTo(buf)
	releaseWriter(w)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v PaTaskBody) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson7459bf99EncodeModels3(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *PaTaskBody) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson7459bf99DecodeModels3(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *PaTaskBody) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson7459bf99DecodeModels3(l, v)
}
func easyjson7459bf99DecodeModels4(in *jlexer.Lexer, out *PaResBody) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "task_id":
			out.Task_Id = string(in.String())
		case "task_res":
			(out.Task_Res).UnmarshalEasyJSON(in)
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson7459bf99EncodeModels4(out *jwriter.Writer, in PaResBody) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"task_id\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Id))
	}
	{
		const prefix string = ",\"task_res\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		(in.Task_Res).MarshalEasyJSON(out)
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v PaResBody) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson7459bf99EncodeModels4(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v PaResBody) MarshalJSONToByteBuffer(buf *bytes.Buffer) ([]byte, error) {
	w := acquireWriter()
	easyjson7459bf99EncodeModels4(w, v)
	_, err := w.DumpTo(buf)
	releaseWriter(w)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v PaResBody) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson7459bf99EncodeModels4(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *PaResBody) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson7459bf99DecodeModels4(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *PaResBody) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson7459bf99DecodeModels4(l, v)
}
func easyjson7459bf99DecodeModels5(in *jlexer.Lexer, out *PaProcRes) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "task_proc_chan":
			out.Task_Proc_Chan = int(in.Int())
		case "task_proc_utt":
			out.Task_Proc_Utt = string(in.String())
		case "task_proc_duration":
			out.Task_Proc_Duration = float64(in.Float64())
		case "task_proc_validduration":
			out.Task_Proc_ValidDuration = float64(in.Float64())
		case "task_proc_checksum":
			out.Task_Proc_CheckSum = string(in.String())
		case "task_proc_errcode":
			out.Task_Proc_ErrCode = int(in.Int())
		case "task_proc_errmsg":
			out.Task_Proc_ErrMsg = string(in.String())
		case "task_proc_top":
			out.Task_Proc_Top = int(in.Int())
		case "task_proc_candidates":
			if in.IsNull() {
				in.Skip()
				out.Task_Proc_Candidates = nil
			} else {
				in.Delim('[')
				if out.Task_Proc_Candidates == nil {
					if !in.IsDelim(']') {
						out.Task_Proc_Candidates = make([]*PaIdentifyCandidate, 0, 8)
					} else {
						out.Task_Proc_Candidates = []*PaIdentifyCandidate{}
					}
				} else {
					out.Task_Proc_Candidates = (out.Task_Proc_Candidates)[:0]
				}
				for !in.IsDelim(']') {
					var v13 *PaIdentifyCandidate
					if in.IsNull() {
						in.Skip()
						v13 = nil
					} else {
						if v13 == nil {
							v13 = new(PaIdentifyCandidate)
						}
						(*v13).UnmarshalEasyJSON(in)
					}
					out.Task_Proc_Candidates = append(out.Task_Proc_Candidates, v13)
					in.WantComma()
				}
				in.Delim(']')
			}
		case "task_proc_spkid":
			out.Task_Proc_SpkId = string(in.String())
		case "task_proc_confidence":
			out.Task_Proc_Confidence = float32(in.Float32())
		case "task_proc_featurefile":
			if in.IsNull() {
				in.Skip()
				out.Task_Proc_FeatureFile = nil
			} else {
				out.Task_Proc_FeatureFile = in.Bytes()
			}
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson7459bf99EncodeModels5(out *jwriter.Writer, in PaProcRes) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"task_proc_chan\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Proc_Chan))
	}
	{
		const prefix string = ",\"task_proc_utt\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Proc_Utt))
	}
	{
		const prefix string = ",\"task_proc_duration\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Float64(float64(in.Task_Proc_Duration))
	}
	{
		const prefix string = ",\"task_proc_validduration\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Float64(float64(in.Task_Proc_ValidDuration))
	}
	{
		const prefix string = ",\"task_proc_checksum\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Proc_CheckSum))
	}
	{
		const prefix string = ",\"task_proc_errcode\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Proc_ErrCode))
	}
	{
		const prefix string = ",\"task_proc_errmsg\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Proc_ErrMsg))
	}
	{
		const prefix string = ",\"task_proc_top\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Int(int(in.Task_Proc_Top))
	}
	{
		const prefix string = ",\"task_proc_candidates\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		if in.Task_Proc_Candidates == nil && (out.Flags&jwriter.NilSliceAsEmpty) == 0 {
			out.RawString("null")
		} else {
			out.RawByte('[')
			for v15, v16 := range in.Task_Proc_Candidates {
				if v15 > 0 {
					out.RawByte(',')
				}
				if v16 == nil {
					out.RawString("null")
				} else {
					(*v16).MarshalEasyJSON(out)
				}
			}
			out.RawByte(']')
		}
	}
	{
		const prefix string = ",\"task_proc_spkid\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Task_Proc_SpkId))
	}
	{
		const prefix string = ",\"task_proc_confidence\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Float32(float32(in.Task_Proc_Confidence))
	}
	{
		const prefix string = ",\"task_proc_featurefile\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Base64Bytes(in.Task_Proc_FeatureFile)
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v PaProcRes) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson7459bf99EncodeModels5(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v PaProcRes) MarshalJSONToByteBuffer(buf *bytes.Buffer) ([]byte, error) {
	w := acquireWriter()
	easyjson7459bf99EncodeModels5(w, v)
	_, err := w.DumpTo(buf)
	releaseWriter(w)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v PaProcRes) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson7459bf99EncodeModels5(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *PaProcRes) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson7459bf99DecodeModels5(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *PaProcRes) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson7459bf99DecodeModels5(l, v)
}
func easyjson7459bf99DecodeModels6(in *jlexer.Lexer, out *PaIdentifyCandidate) {
	isTopLevel := in.IsStart()
	if in.IsNull() {
		if isTopLevel {
			in.Consumed()
		}
		in.Skip()
		return
	}
	in.Delim('{')
	for !in.IsDelim('}') {
		key := in.UnsafeString()
		in.WantColon()
		if in.IsNull() {
			in.Skip()
			in.WantComma()
			continue
		}
		switch key {
		case "identify_spkid":
			out.Identify_SpkId = string(in.String())
		case "identify_confidence":
			out.Identify_Confidence = float32(in.Float32())
		case "identify_node":
			out.Identify_Node = string(in.String())
		default:
			in.SkipRecursive()
		}
		in.WantComma()
	}
	in.Delim('}')
	if isTopLevel {
		in.Consumed()
	}
}
func easyjson7459bf99EncodeModels6(out *jwriter.Writer, in PaIdentifyCandidate) {
	out.RawByte('{')
	first := true
	_ = first
	{
		const prefix string = ",\"identify_spkid\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Identify_SpkId))
	}
	{
		const prefix string = ",\"identify_confidence\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.Float32(float32(in.Identify_Confidence))
	}
	{
		const prefix string = ",\"identify_node\":"
		if first {
			first = false
			out.RawString(prefix[1:])
		} else {
			out.RawString(prefix)
		}
		out.String(string(in.Identify_Node))
	}
	out.RawByte('}')
}

// MarshalJSON supports json.Marshaler interface
func (v PaIdentifyCandidate) MarshalJSON() ([]byte, error) {
	w := jwriter.Writer{}
	easyjson7459bf99EncodeModels6(&w, v)
	return w.Buffer.BuildBytes(), w.Error
}

func (v PaIdentifyCandidate) MarshalJSONToByteBuffer(buf *bytes.Buffer) ([]byte, error) {
	w := acquireWriter()
	easyjson7459bf99EncodeModels6(w, v)
	_, err := w.DumpTo(buf)
	releaseWriter(w)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), w.Error
}

// MarshalEasyJSON supports easyjson.Marshaler interface
func (v PaIdentifyCandidate) MarshalEasyJSON(w *jwriter.Writer) {
	easyjson7459bf99EncodeModels6(w, v)
}

// UnmarshalJSON supports json.Unmarshaler interface
func (v *PaIdentifyCandidate) UnmarshalJSON(data []byte) error {
	r := jlexer.Lexer{Data: data}
	easyjson7459bf99DecodeModels6(&r, v)
	return r.Error()
}

// UnmarshalEasyJSON supports easyjson.Unmarshaler interface
func (v *PaIdentifyCandidate) UnmarshalEasyJSON(l *jlexer.Lexer) {
	easyjson7459bf99DecodeModels6(l, v)
}
