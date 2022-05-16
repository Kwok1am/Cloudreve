package conf

import (
	"github.com/cloudreve/Cloudreve/v3/pkg/util"
	"github.com/go-ini/ini"
	"gopkg.in/go-playground/validator.v9"
)

// database 数据库
type database struct {
	Type        string
	User        string
	Password    string
	Host        string
	Name        string
	TablePrefix string
	DBFile      string
	Port        int
	Charset     string
	TLS         string
}

// system 系统通用配置
type system struct {
	Mode          string `validate:"eq=master|eq=slave"`
	Listen        string `validate:"required"`
	Debug         bool
	SessionSecret string
	HashIDSalt    string
}

type ssl struct {
	CertPath string `validate:"omitempty,required"`
	KeyPath  string `validate:"omitempty,required"`
	Listen   string `validate:"required"`
}

type unix struct {
	Listen string
}

// slave 作为slave存储端配置
type slave struct {
	Secret          string `validate:"omitempty,gte=64"`
	CallbackTimeout int    `validate:"omitempty,gte=1"`
	SignatureTTL    int    `validate:"omitempty,gte=1"`
}

// redis 配置
type redis struct {
	Network  string
	Server   string
	Password string
	DB       string
}

// 跨域配置
type cors struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
}

var cfg *ini.File

const defaultConf = `[System]
Debug = false
Mode = master
Listen = :5212
SessionSecret = {SessionSecret}
HashIDSalt = {HashIDSalt}
`

// Init 初始化配置文件
func Init(path string) {
	var err error

	if path == "" || !util.Exists(path) {
		// 创建初始配置文件
		confContent := util.Replace(map[string]string{
			"{SessionSecret}": util.RandStringRunes(64),
			"{HashIDSalt}":    util.RandStringRunes(64),
		}, defaultConf)
		f, err := util.CreatNestedFile(path)
		if err != nil {
			util.Log().Panic("无法创建配置文件, %s", err)
		}

		// 写入配置文件
		_, err = f.WriteString(confContent)
		if err != nil {
			util.Log().Panic("无法写入配置文件, %s", err)
		}

		f.Close()
	}

	cfg, err = ini.Load(path)
	if err != nil {
		util.Log().Panic("无法解析配置文件 '%s': %s", path, err)
	}

	sections := map[string]interface{}{
		"Database":   DatabaseConfig,
		"System":     SystemConfig,
		"SSL":        SSLConfig,
		"UnixSocket": UnixConfig,
		"Redis":      RedisConfig,
		"CORS":       CORSConfig,
		"Slave":      SlaveConfig,
	}
	for sectionName, sectionStruct := range sections {
		err = mapSection(sectionName, sectionStruct)
		if err != nil {
			util.Log().Panic("配置文件 %s 分区解析失败: %s", sectionName, err)
		}
	}

	// 映射数据库配置覆盖
	for _, key := range cfg.Section("OptionOverwrite").Keys() {
		OptionOverwrite[key.Name()] = key.Value()
	}

	// 重设log等级
	if !SystemConfig.Debug {
		util.Level = util.LevelInformational
		util.GloablLogger = nil
		util.Log()
	}

}

// mapSection 将配置文件的 Section 映射到结构体上
func mapSection(section string, confStruct interface{}) error {
	err := cfg.Section(section).MapTo(confStruct)
	if err != nil {
		return err
	}

	// 验证合法性
	validate := validator.New()
	err = validate.Struct(confStruct)
	if err != nil {
		return err
	}

	return nil
}
