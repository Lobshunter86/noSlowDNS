package DNS

type Config struct {
	Net               string
	Addr              string
	PrimaryUpstream   string
	SecondaryUpstream string
}

var Default = Config{
	Net:               "udp",
	Addr:              "127.0.0.1:5533",
	PrimaryUpstream:   "223.5.5.5:53",
	SecondaryUpstream: "8.8.8.8:53",
}

const (
	// DNS cache expiration time
	ExpireTime  = 300
	CleanupTime = 600
)
