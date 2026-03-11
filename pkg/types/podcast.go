package types

type Podcast struct {
	Dir       string    `json:"dir" yaml:"dir"`
	ByteDance ByteDance `json:"byteDance" yaml:"byteDance"`
}
type ByteDance struct {
	AppID       string `json:"appID" yaml:"appID"`
	AccessToken string `json:"accessToken" yaml:"accessToken"`
}
