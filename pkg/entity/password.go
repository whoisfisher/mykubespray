package entity

type PasswordConf struct {
	Host
	NewPassword string
}

// PasswordInfo 结构体用于存储用户密码状态信息
type PasswordInfo struct {
	LastChange       string
	PasswordExpires  string
	PasswordInactive string
	AccountExpires   string
	MinDays          int
	MaxDays          int
	WarningDays      int
}
