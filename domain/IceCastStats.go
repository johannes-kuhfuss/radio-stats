package domain

type IceStats struct {
	Admin              string `json:"admin"`
	Host               string `json:"host"`
	Location           string `json:"location"`
	ServerId           string `json:"server_id"`
	ServerStart        string `json:"server_start"`
	ServerStartIso8601 string `json:"server_start_iso8601"`
	Source             []struct {
		AudioInfo          string      `json:"audio_info,omitempty"`
		Channels           int         `json:"channels,omitempty"`
		Genre              string      `json:"genre,omitempty"`
		ListenerPeak       int         `json:"listener_peak,omitempty"`
		Listeners          int         `json:"listeners"`
		ListenUrl          string      `json:"listenurl"`
		SampleRate         int         `json:"samplerate,omitempty"`
		ServerDescription  string      `json:"server_description,omitempty"`
		ServerName         string      `json:"server_name,omitempty"`
		ServerType         string      `json:"server_type,omitempty"`
		ServerUrl          string      `json:"server_url,omitempty"`
		StreamStart        string      `json:"stream_start,omitempty"`
		StreamStartIso8601 string      `json:"stream_start_iso8601,omitempty"`
		Title              string      `json:"title,omitempty"`
		Dummy              interface{} `json:"dummy"`
		Artist             interface{} `json:"artist,omitempty"`
		AudioBitrate       int         `json:"audio_bitrate,omitempty"`
		AudioChannels      int         `json:"audio_channels,omitempty"`
		AudioSampleRate    int         `json:"audio_samplerate,omitempty"`
		BitRate            string      `json:"bitrate,omitempty"`
		IceBitRate         int         `json:"ice-bitrate,omitempty"`
		IceChannels        int         `json:"ice-channels,omitempty"`
		IceSampleRate      int         `json:"ice-samplerate,omitempty"`
		SubType            string      `json:"subtype,omitempty"`
		Quality            float64     `json:"quality,omitempty"`
	} `json:"source"`
}
