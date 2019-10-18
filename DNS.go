package DNS

import (
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/miekg/dns"
)

type Server struct {
	Config
	server *dns.Server
	Cache  *cache.Cache
}

// NewServer create a new dns server with Config.
func NewServer(cfg Config) *Server {
	server := new(Server)
	server.Config = cfg
	server.Cache = cache.New(ExpireTime*time.Second, CleanupTime*time.Second)
	server.server = &dns.Server{
		Addr:    server.Addr,
		Net:     server.Net,
		Handler: server,
	}

	return server
}

// Run starts the server.
func (server *Server) Run() {

	println("Server start")
	err := server.server.ListenAndServe()
	for {
		if err != nil {
			//// log error
		}
		err = server.server.ListenAndServe()
	}
}

// ServeDNS implements the ResponseWriter.LocalAddr method.
func (server *Server) ServeDNS(w dns.ResponseWriter, query *dns.Msg) {
	Debug("recv a query")

	var reply *dns.Msg

	if len(query.Question) > 1 {
		reply = server.LookupNet(query)
	} else if len(query.Question) < 1 {
		reply = SERVERFAIL(query)
	} else {
		reply = server.LookupCache(query)
	}

	reply = reply.Copy() // no write to cached, so no read/write conflit
	reply.Id = query.Id
	w.WriteMsg(reply)
}

// LookupCache search reply in cache
// if search fails, call LookupNet and store reply in cache before return
func (server *Server) LookupCache(query *dns.Msg) *dns.Msg {
	key := query.Question[0].String()
	reply, _ := server.Cache.Get(key)
	if reply != nil {
		Debug("cache hit", query.Question[0])
		return reply.(*dns.Msg)
	}

	msg := server.LookupNet(query)
	if msg == nil {
		return SERVERFAIL(query)
	}

	Debug("lookupnet:", msg.Answer)

	// ttl = cache time, drop msg after ttl seconds
	var ttl time.Duration
	if len(msg.Answer) == 0 {
		if len(msg.Ns) == 0 { // no usable answer, dont cache
			return msg
		}
		ttl = time.Duration(msg.Ns[0].Header().Ttl) * time.Second
	} else {
		ttl = time.Duration(msg.Answer[0].Header().Ttl) * time.Second
	}

	if ttl > ExpireTime*time.Second {
		ttl = ExpireTime * time.Second
	}
	server.Cache.Set(key, msg, ttl)

	return msg
}

// LookupNet ask upstream for DNS responses
func (server *Server) LookupNet(query *dns.Msg) *dns.Msg {
	client := dns.Client{Net: server.Net}

	var reply *dns.Msg
	var rtt time.Duration
	var err error

	reply, rtt, err = client.Exchange(query, server.PrimaryUpstream)
	if err != nil {
		//// log
		Debug("error:", err)

		reply, rtt, err = client.Exchange(query, server.PrimaryUpstream)
		if err != nil {
			//// log
			Debug("error:", err)
		}
	}

	Debug("query upstream:", rtt)
	if reply != nil {
		reply.Compress = true
	}
	return reply
}

func SERVERFAIL(query *dns.Msg) *dns.Msg {
	reply := &dns.Msg{}
	reply.SetRcode(query, dns.RcodeServerFailure)
	return reply
}

func NXDOMAIN(query *dns.Msg) *dns.Msg {
	reply := &dns.Msg{}
	reply.SetRcode(query, dns.RcodeNameError)
	return reply
}
