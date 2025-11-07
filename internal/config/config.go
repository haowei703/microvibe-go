package config

import (
	"errors"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Upload    UploadConfig
	Streaming StreamingConfig
	SFU       SFUConfig
	WebRTC    WebRTCConfig
	OAuth     OAuthConfig
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string
	Port string
	Mode string
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret string
	Expire int // 过期时间（小时）
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxSize      int64    // 最大文件大小（字节）
	AllowedTypes []string // 允许的文件类型
	Path         string   // 上传路径
}

// StreamingConfig 流媒体服务器配置
type StreamingConfig struct {
	// RTMP 推流服务器配置
	RTMPServer string `mapstructure:"rtmp_server"` // RTMP 推流地址前缀
	RTMPApp    string `mapstructure:"rtmp_app"`    // RTMP 应用名

	// 播放地址配置
	HLSServer      string `mapstructure:"hls_server"`       // HLS 播放地址前缀
	FLVServer      string `mapstructure:"flv_server"`       // FLV 播放地址前缀
	RTMPPlayServer string `mapstructure:"rtmp_play_server"` // RTMP 播放地址前缀

	// 默认流配置
	DefaultStreamType   string `mapstructure:"default_stream_type"`   // 默认流类型
	DefaultVideoCodec   string `mapstructure:"default_video_codec"`   // 默认视频编解码器
	DefaultAudioCodec   string `mapstructure:"default_audio_codec"`   // 默认音频编解码器
	DefaultVideoBitrate int    `mapstructure:"default_video_bitrate"` // 默认视频码率 (kbps)
	DefaultAudioBitrate int    `mapstructure:"default_audio_bitrate"` // 默认音频码率 (kbps)
	DefaultFrameRate    int    `mapstructure:"default_frame_rate"`    // 默认帧率
	DefaultResolution   string `mapstructure:"default_resolution"`    // 默认分辨率
}

// SFUConfig SFU 服务器配置
type SFUConfig struct {
	Enabled           bool     `mapstructure:"enabled"`             // 是否启用 SFU
	ServerURL         string   `mapstructure:"server_url"`          // SFU 服务器地址（gRPC 格式：host:port）
	Mode              string   `mapstructure:"mode"`                // standalone (独立) 或 cluster (集群)
	ClusterNodes      []string `mapstructure:"cluster_nodes"`       // 集群节点列表（gRPC 格式：host:port）
	LoadBalanceMethod string   `mapstructure:"load_balance_method"` // 负载均衡方法
}

// WebRTCConfig WebRTC 和 SFU 配置
type WebRTCConfig struct {
	// ICE 服务器配置
	ICEServers []ICEServerConfig `mapstructure:"ice_servers"`

	// 端口范围
	PortMin uint16 `mapstructure:"port_min"` // UDP 端口最小值
	PortMax uint16 `mapstructure:"port_max"` // UDP 端口最大值

	// 编解码器配置
	VideoCodecs []string `mapstructure:"video_codecs"` // 视频编解码器：VP8, VP9, H264, AV1
	AudioCodecs []string `mapstructure:"audio_codecs"` // 音频编解码器：Opus, G722

	// 质量配置
	MaxBandwidth       int  `mapstructure:"max_bandwidth"`        // 最大带宽 (kbps)
	VideoBitrate       int  `mapstructure:"video_bitrate"`        // 视频比特率 (kbps)
	AudioBitrate       int  `mapstructure:"audio_bitrate"`        // 音频比特率 (kbps)
	EnableSimulcast    bool `mapstructure:"enable_simulcast"`     // 启用联播 (多码率)
	EnableDynacast     bool `mapstructure:"enable_dynacast"`      // 启用动态广播
	EnableAdaptiveRate bool `mapstructure:"enable_adaptive_rate"` // 启用自适应码率

	// NACK 和重传配置
	EnableNACK  bool   `mapstructure:"enable_nack"`  // 启用 NACK (丢包重传)
	EnablePLI   bool   `mapstructure:"enable_pli"`   // 启用 PLI (关键帧请求)
	PLIInterval string `mapstructure:"pli_interval"` // PLI 请求间隔

	// SFU 服务器配置
	SFUAddress        string   `mapstructure:"sfu_address"`         // SFU 服务器地址（JSON-RPC）
	SFUMode           string   `mapstructure:"sfu_mode"`            // SFU 模式：standalone (独立) 或 cluster (集群)
	ClusterNodes      []string `mapstructure:"cluster_nodes"`       // 集群节点列表（格式：host:port）
	LoadBalanceMethod string   `mapstructure:"load_balance_method"` // 负载均衡方法：random, roundrobin, leastconn
}

// ICEServerConfig ICE 服务器配置
type ICEServerConfig struct {
	URLs       []string `mapstructure:"urls"`
	Username   string   `mapstructure:"username"`
	Credential string   `mapstructure:"credential"`
}

// OAuthConfig OAuth2/OIDC 配置
type OAuthConfig struct {
	Authentik AuthentikConfig `mapstructure:"authentik"`
}

// AuthentikConfig Authentik OAuth2/OIDC 配置
type AuthentikConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	IssuerURL    string   `mapstructure:"issuer_url"` // 内部地址（Docker 网络）
	PublicURL    string   `mapstructure:"public_url"` // 外部地址（浏览器访问）
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	Scopes       []string `mapstructure:"scopes"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "microvibe")
	viper.SetDefault("database.sslmode", "disable")

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("jwt.secret", "microvibe-secret-key-change-in-production")
	viper.SetDefault("jwt.expire", 24)

	viper.SetDefault("upload.maxsize", 104857600) // 100MB
	viper.SetDefault("upload.allowedtypes", []string{"video/mp4", "video/avi", "image/jpeg", "image/png"})
	viper.SetDefault("upload.path", "./uploads")

	// 流媒体服务器默认配置
	viper.SetDefault("streaming.rtmp_server", "rtmp://localhost:1935/live")
	viper.SetDefault("streaming.rtmp_app", "live")
	viper.SetDefault("streaming.hls_server", "http://localhost:8080/hls")
	viper.SetDefault("streaming.flv_server", "http://localhost:8080/flv")
	viper.SetDefault("streaming.rtmp_play_server", "rtmp://localhost:1935/live")
	viper.SetDefault("streaming.default_stream_type", "video_audio")
	viper.SetDefault("streaming.default_video_codec", "h264")
	viper.SetDefault("streaming.default_audio_codec", "aac")
	viper.SetDefault("streaming.default_video_bitrate", 2500)
	viper.SetDefault("streaming.default_audio_bitrate", 128)
	viper.SetDefault("streaming.default_frame_rate", 30)
	viper.SetDefault("streaming.default_resolution", "720p")

	// SFU 服务器默认配置
	viper.SetDefault("sfu.enabled", true)
	viper.SetDefault("sfu.server_url", "localhost:7100")
	viper.SetDefault("sfu.mode", "standalone")
	viper.SetDefault("sfu.cluster_nodes", []string{})
	viper.SetDefault("sfu.load_balance_method", "roundrobin")

	// WebRTC 默认配置
	viper.SetDefault("webrtc.ice_servers", []map[string]interface{}{
		{"urls": []string{"stun:stun.l.google.com:19302"}},
	})
	viper.SetDefault("webrtc.port_min", 5000)
	viper.SetDefault("webrtc.port_max", 5100)
	viper.SetDefault("webrtc.video_codecs", []string{"VP8", "VP9", "H264"})
	viper.SetDefault("webrtc.audio_codecs", []string{"Opus"})
	viper.SetDefault("webrtc.max_bandwidth", 2000)        // 2 Mbps
	viper.SetDefault("webrtc.video_bitrate", 1500)        // 1.5 Mbps
	viper.SetDefault("webrtc.audio_bitrate", 128)         // 128 kbps
	viper.SetDefault("webrtc.enable_simulcast", true)     // 启用联播
	viper.SetDefault("webrtc.enable_dynacast", true)      // 启用动态广播
	viper.SetDefault("webrtc.enable_adaptive_rate", true) // 启用自适应码率
	viper.SetDefault("webrtc.enable_nack", true)          // 启用丢包重传
	viper.SetDefault("webrtc.enable_pli", true)           // 启用关键帧请求
	viper.SetDefault("webrtc.pli_interval", "3s")
	viper.SetDefault("webrtc.sfu_address", "http://ion-sfu:7001") // 默认 SFU 地址
	viper.SetDefault("webrtc.sfu_mode", "standalone")             // 默认独立模式
	viper.SetDefault("webrtc.load_balance_method", "roundrobin")

	// OAuth/Authentik 默认配置
	viper.SetDefault("oauth.authentik.enabled", false)
	viper.SetDefault("oauth.authentik.issuer_url", "http://localhost:9000/application/o/microvibe/")
	viper.SetDefault("oauth.authentik.client_id", "microvibe-backend")
	viper.SetDefault("oauth.authentik.client_secret", "")
	viper.SetDefault("oauth.authentik.redirect_url", "http://localhost:8888/api/v1/oauth/callback")
	viper.SetDefault("oauth.authentik.scopes", []string{"openid", "email", "profile"})

	// 允许环境变量覆盖
	// 将环境变量中的下划线转换为点号
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file (optional, will use defaults if not found)
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
