package logger

type Config struct {
	Name           string
	LogLevel       string `json:"logLevel"`
	LogTimeFormat  string `json:"logTimeFormat"`
	LogFile        string `json:"logFile"`
	LogStream      string `json:"logStream"`
	LogNoTime      bool   `json:"logNoTime"`
	ShowCallerInfo bool   `json:"showCallerInfo"`
	LogBufferSync  bool   `json:"logBufferSync"`
}

func newLoggerConfig() *Config {
	return &Config{
		LogLevel:       "info",
		LogTimeFormat:  "2006-01-02 15:04:05.000",
		LogFile:        "",
		LogStream:      "",
		LogNoTime:      false,
		ShowCallerInfo: false,
		LogBufferSync:  false,
	}
}

func (c *Config) SetName(name string) *Config {
	c.Name = name
	return c
}

func (c *Config) SetLogLevel(level string) *Config {
	c.LogLevel = level
	return c
}

func (c *Config) SetLogTimeFormat(format string) *Config {
	c.LogTimeFormat = format
	return c
}

func (c *Config) SetLogFile(file string) *Config {
	c.LogFile = file
	return c
}

func (c *Config) SetLogStream(stream string) *Config {
	c.LogStream = stream
	return c
}

func (c *Config) SetLogNoTime(noTime bool) *Config {
	c.LogNoTime = noTime
	return c
}

func (c *Config) SetShowCallerInfo(showCallerInfo bool) *Config {
	c.ShowCallerInfo = showCallerInfo
	return c
}

func (c *Config) SetLogBufferSync(logBufferSync bool) *Config {
	c.LogBufferSync = logBufferSync
	return c
}
