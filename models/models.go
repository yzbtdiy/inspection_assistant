package models

// 配置文件结构体
type Config struct {
	BrowserPath  string `yaml:"browser_path"`
	BrowserData  string `yaml:"browser_data"`
	KeeppassPath string `yaml:"keepass_path"`
	KpdbPath     string `yaml:"kpdb_path"`
	SitesInfo    []Site `yaml:"sites_info"`
}

// 站点标题和选择器结构体
type Site struct {
	EntryTitle      string `yaml:"entry_title"`
	UserLocator     string `yaml:"user_locator"`
	PassLocator     string `yaml:"pass_locator"`
	PassFillLocator string `yaml:"pass_fill_locator" default:""`
	LoginLocator    string `yaml:"login_locator"`
	AutoLogin       bool   `yaml:"auto_login" default:"false"`
}

// 站点信息结构体
type Entry struct {
	Url      string
	Username string
	Password string
}