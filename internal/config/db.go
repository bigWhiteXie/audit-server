package config

type MySQLConf struct {
	Host            string `json:"" yaml:"Host"`
	Port            int64  `json:"" yaml:"Port"`
	User            string `json:"" yaml:"User"`
	Password        string `json:"" yaml:"Password"`
	Database        string `json:"" yaml:"Database"`
	AutoMigrate     bool   `json:"" yaml:"AutoMigrate"`
	MaxOpenConns    int    `json:",default=1000" yaml:"MaxOpenConns"`
	MaxIdleConns    int    `json:",default=100" yaml:"MaxIdleConns"`
	ConnMaxLifetime int    `json:",default=100" yaml:"ConnMaxLifetime"`
}
