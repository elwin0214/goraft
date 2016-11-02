package raft

type Cluster struct {
	servers []*Server
}

func NewCluster(peers []string, observeChan chan interface{}) *Cluster {
	config := NewConfig(peers)
	servers := make([]*Server, 0, len(config.peers))
	channels := make(map[string]*Channel, len(config.peers))

	for _, p := range config.peers {
		server := NewServer(p, NewLogStroe(), NewRaftState(), config, observeChan)
		servers = append(servers, server)
		channel := NewChannel(server.GetVoteChan(), server.GetAppendChan())
		channels[p] = channel
	}

	trans := NewMemTransporter(channels)
	for _, server := range servers {
		server.SetTransporter(trans)
	}
	return &Cluster{servers}
}

func (c *Cluster) GetServers() []*Server {
	return c.servers
}

func (c *Cluster) Get(name string) *Server {
	for _, server := range c.servers {
		if server.name == name {
			return server
		}
	}
	return nil
}

func (c *Cluster) GetLeader() *Server {
	for _, server := range c.servers {
		if server.GetState() == Leader {
			return server
		}
	}
	return nil
}

func (c *Cluster) Start() {
	for _, server := range c.servers {
		server.Start()
	}
}

func (c *Cluster) Stop() {
	for _, server := range c.servers {
		server.Stop()
	}
}
