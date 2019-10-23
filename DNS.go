package DNS

import (
	"log"
	"os"
	"time"

	"github.com/patrickmn/go-cache"

	"./slowIP"
	"github.com/miekg/dns"
)

type Server struct {
	Config
	Cache      *cache.Cache
	errorLog   *log.Logger
	resolveLog *log.Logger
	server     *dns.Server
}

// NewServer create a new dns server with Config.
func NewServer(cfg Config) *Server {
	server := new(Server)
	server.Config = cfg
	server.Cache = cache.New(cfg.ExpireTime*time.Second, cfg.CleanupTime*time.Second)

	server.resolveLog = log.New(os.Stdout, "Info: ", log.LstdFlags)
	server.errorLog = log.New(os.Stderr, "Error: ", log.LstdFlags)

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
			server.errorLog.Println(err)
		}
		err = server.server.ListenAndServe()
	}
}

// ServeDNS implements the ResponseWriter.LocalAddr method.
func (server *Server) ServeDNS(w dns.ResponseWriter, query *dns.Msg) {
	var reply *dns.Msg

	Debug("recv a query")
	server.resolveLog.Println("Recv from:", w.RemoteAddr().String())

	if len(query.Question) > 1 {
		reply = server.LookupNet(query)
	} else if len(query.Question) < 1 {
		reply = server.SERVERFAIL(query)
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
		Debug("cache hit", query.Question[0].String())
		server.resolveLog.Println("cache hit", query.Question[0].String())
		return reply.(*dns.Msg)
	}

	msg := server.LookupNet(query)
	if msg == nil {
		return server.SERVERFAIL(query)
	}

	// Debug("lookupnet:", msg.Answer)

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

	if ttl > server.Config.ExpireTime*time.Second {
		ttl = server.Config.ExpireTime * time.Second
	}

	// filter slow A record before cache
	if containA(msg) {
		server.filterSlow(msg)
	}

	server.Cache.Set(key, msg, ttl)

	return msg
}

// LookupNet ask upstream for DNS responses
func (server *Server) LookupNet(query *dns.Msg) *dns.Msg {
	server.resolveLog.Println("LookupNet:", query.Question)
	client := dns.Client{Net: server.Net}

	var reply *dns.Msg
	var rtt time.Duration
	var err error

	reply, rtt, err = client.Exchange(query, server.PrimaryUpstream)
	if err != nil {
		server.errorLog.Println("LookupNet:", err)
		Debug("error:", err)

		reply, rtt, err = client.Exchange(query, server.SecondaryUpstream)
		if err != nil {
			server.errorLog.Println("LookupNet:", err)
			Debug("error:", err)
		}
	}

	Debug("query upstream:", rtt)
	if reply != nil {
		reply.Compress = true
	}
	return reply
}

func containA(res *dns.Msg) bool {
	rrs := res.Answer

	for i := 0; i < len(rrs); i++ {
		_, ok := rrs[i].(*dns.A)
		if ok {
			return true
		}
	}
	return false
}

// filterSlow filter slow ip in msg.Answer if there are non-slow ip in msg
func (server *Server) filterSlow(msg *dns.Msg) {
	answer := make([]dns.RR, 0)

	ACount := 0 // number of a record, should return at least one
	for _, record := range msg.Answer {
		rr, ok := record.(*dns.A)
		if ok {
			ip := rr.A.String()
			if slowIP.IsSlowIP(ip) {
				Debug("filter slooooooooooooow")
				continue
			} else {
				ACount++
			}
		}
		answer = append(answer, record)
	}

	if ACount > 0 {
		Debug("cut off :", len(msg.Answer)-len(answer), "answers")
		msg.Answer = answer
	} else {
		server.resolveLog.Println("FilterFail:", "Question:", msg.Question, "Answer:", msg.Answer)
	}
}

func (server *Server) SERVERFAIL(query *dns.Msg) *dns.Msg {
	server.errorLog.Println("SERVERFAIL", query)
	reply := &dns.Msg{}
	reply.SetRcode(query, dns.RcodeServerFailure)
	return reply
}

func (server *Server) NXDOMAIN(query *dns.Msg) *dns.Msg {
	reply := &dns.Msg{}
	reply.SetRcode(query, dns.RcodeNameError)
	return reply
}
