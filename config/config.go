package config

type Config struct {
	Title     string
	Log       Log
	Instances []Instance
}

type Log struct {
	Goid    bool
	File    string
	Level   string
	Console bool
	Type    string
	Maxnum  int32
	Size    int64
	Unit    string
}

type Instance struct {
	Name      string
	Enabled   bool
	Bind      string
	Accounts  map[string]Account
	Write     Host
	Reads     []Host
	Balance   string
	KeepAlive int64
	MaxIdle   uint32
	MaxCount  uint32
}

type Account struct {
	// Username string
	Password string
	Readonly bool
}

type Host struct {
	Addr     string
	Username string
	Password string
}
