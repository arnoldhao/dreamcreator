package types

import (
	"errors"
	"fmt"
	"time"

	"CanMe/backend/consts"
)

type TaskRequest struct {
	TaskID    string   `json:"taskId"`
	ContentID string   `json:"contentId"`
	Total     int      `json:"total"`
	Stream    string   `json:"stream"`
	Captions  []string `json:"captions"`
	Danmaku   bool     `json:"danmaku"`
}

func (r *TaskRequest) Validate() error {
	if r.ContentID == "" {
		return errors.New("content_id is required")
	}
	if r.Total <= 0 {
		return errors.New("total must be positive")
	}
	if r.Stream == "" {
		return errors.New("stream must be specified")
	}
	return nil
}

type ContentResponse struct {
	Videos []*ExtractorData `json:"videos"`
	Total  int              `json:"total"`
}

type TaskResponse struct {
	TaskID string `json:"taskId"`
	Status string `json:"status"`
}

type DownloadTask struct {
	ID          string                `json:"id" yaml:"id"`
	Status      consts.DownloadStatus `json:"status" yaml:"status"`
	Streams     map[string]*Part      `json:"streams" yaml:"streams"`
	Captions    map[string]*Part      `json:"captions" yaml:"captions"`
	Total       int64                 `json:"total" yaml:"total"`
	Finished    int64                 `json:"finished" yaml:"finished"`
	Size        int64                 `json:"size" yaml:"size"`
	Current     int64                 `json:"current" yaml:"current"`
	Speed       string                `json:"speed" yaml:"speed"`
	LastTime    time.Time             `json:"lastTime" yaml:"lastTime"`
	LastCurrent int64                 `json:"lastCurrent" yaml:"lastCurrent"`
	SpeedFloat  float64               `json:"speedFloat" yaml:"speedFloat"`
	Progress    float64               `json:"progress" yaml:"progress"`
	Error       error                 `json:"error" yaml:"error"`
}

func (d *DownloadTask) UpdateSpeed(current int64) {
	now := time.Now()
	duration := now.Sub(d.LastTime).Seconds()
	if duration > 0 {
		downloaded := current - d.LastCurrent
		d.SpeedFloat = float64(downloaded) / duration
		d.Speed = d.GetSpeedString()
	}
	d.LastTime = now
	d.Current = current
	d.LastCurrent = current

	// fix progress
	if d.Total > 0 { // prevent division by zero
		d.Progress = float64(d.Current) / float64(d.Size) * 100
		// limit progress range
		if d.Progress > 100 {
			d.Progress = 100
		} else if d.Progress < 0 {
			d.Progress = 0
		}
	} else {
		d.Progress = 0
	}
}

func (d *DownloadTask) GetSpeed() float64 {
	return d.SpeedFloat / 1024 / 1024 // convert to MB/s
}

// GetSpeedString returns formatted speed string
func (d *DownloadTask) GetSpeedString() string {
	speed := d.GetSpeed()
	if speed >= 1.0 {
		return fmt.Sprintf("%.2f MB/s", speed)
	}
	// if speed is less than 1MB/s, display KB/s
	return fmt.Sprintf("%.2f KB/s", speed*1024)
}

type PageInfo struct {
	source string
	site   string
	title  string
}

func NewPageInfo(source, site, title string) *PageInfo {
	return &PageInfo{
		source: source,
		site:   site,
		title:  title,
	}
}

func (p *PageInfo) GetSource() string {
	return p.source
}

func (p *PageInfo) GetSite() string {
	return p.site
}

func (p *PageInfo) GetTitle() string {
	return p.title
}

type StreamInfo struct {
	size    int64
	current int64
	quality string
	needMux bool
}

func NewStreamInfo(size int64, quality string, needMux bool) *StreamInfo {
	return &StreamInfo{
		size:    size,
		current: 0,
		quality: quality,
		needMux: needMux,
	}
}

func (s *StreamInfo) GetSize() int64 {
	return s.size
}

func (s *StreamInfo) SetCurrent(current int64) {
	s.current = current
}

func (s *StreamInfo) GetCurrent() int64 {
	return s.current
}

func (s *StreamInfo) GetQuality() string {
	return s.quality
}

func (s *StreamInfo) GetNeedMux() bool {
	return s.needMux
}

type Part struct {
	*PageInfo
	*StreamInfo
	id          string
	requestCode string
	sourceDir   string
	fileName    string
	url         string
	ext         string
	finished    bool
	err         error
}

func NewPart(id, requestCode, sourceDir, fileName, url, ext string, pageInfo *PageInfo, streamInfo *StreamInfo) *Part {
	return &Part{
		id:          id,
		requestCode: requestCode,
		sourceDir:   sourceDir,
		fileName:    fileName,
		url:         url,
		ext:         ext,
		PageInfo:    pageInfo,
		StreamInfo:  streamInfo,
	}
}

func (p *Part) SetFinished(err error) {
	p.finished = true
	p.err = err
}

func (p *Part) SetCurrent(current int64) {
	if p.StreamInfo != nil {
		p.StreamInfo.SetCurrent(current)
	}
}

func (p *Part) GetCurrent() int64 {
	if p.StreamInfo != nil {
		return p.StreamInfo.GetCurrent()
	}
	return 0
}

func (p *Part) GetID() string {
	return p.id
}

func (p *Part) GetSourceDir() string {
	return p.sourceDir
}

func (p *Part) GetFileName() string {
	return p.fileName
}

func (p *Part) GetURL() string {
	return p.url
}

func (p *Part) GetExt() string {
	return p.ext
}

func (p *Part) GetStatus() (bool, error) {
	return p.finished, p.err
}

func (p *Part) GetSource() string {
	if p.PageInfo != nil {
		return p.PageInfo.GetSource()
	}
	return ""
}

func (p *Part) GetSite() string {
	if p.PageInfo != nil {
		return p.PageInfo.GetSite()
	}
	return ""
}

func (p *Part) GetTitle() string {
	if p.PageInfo != nil {
		return p.PageInfo.GetTitle()
	}
	return ""
}

func (p *Part) GetQuality() string {
	if p.StreamInfo != nil {
		return p.StreamInfo.GetQuality()
	}
	return ""
}

func (p *Part) GetSize() int64 {
	if p.StreamInfo != nil {
		return p.StreamInfo.GetSize()
	}
	return 0
}

func (p *Part) GetNeedMux() bool {
	if p.StreamInfo != nil {
		return p.StreamInfo.GetNeedMux()
	}
	return false
}

func (p *Part) GetRequestCode() string {
	return p.requestCode
}

type ProgressReport struct {
	Part     *Part             `json:"part" yaml:"part"`
	DataType ExtractorDataType `json:"dataType" yaml:"dataType"`
}

func NewProgressReport(part *Part, dataType ExtractorDataType) *ProgressReport {
	return &ProgressReport{
		Part:     part,
		DataType: dataType,
	}
}

func (p *ProgressReport) GetPart() *Part {
	return p.Part
}

func (p *ProgressReport) GetDataType() ExtractorDataType {
	return p.DataType
}

func (p *ProgressReport) GetID() string {
	return p.Part.GetID()
}

func (p *ProgressReport) GetFileName() string {
	return p.Part.GetFileName()
}

func (p *ProgressReport) GetCurrent() int64 {
	return p.Part.GetCurrent()
}
