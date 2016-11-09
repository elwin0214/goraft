package raft

type Cluster struct {
	servers      []*Server
	electionChan chan *ElectMessage
}

func newCluster(peers []string) *Cluster {
	config := newConfig(peers)
	servers := make([]*Server, 0, len(config.peers))
	channels := make(map[string]*Channel, len(config.peers))
	electionChan := make(chan *ElectMessage, 16)
	for _, p := range config.peers {
		server := newServer(p, newLogStroe(), config, electionChan)
		servers = append(servers, server)
		channel := newChannel(server.GetVoteChan(), server.GetAppendChan())
		channels[p] = channel
	}

	trans := newMemTransporter(channels)
	for _, server := range servers {
		server.SetTransporter(trans)
	}
	return &Cluster{servers, electionChan}
}

func (c *Cluster) GetServers() []*Server {
	return c.servers
}

func (c *Cluster) waitElected() *ElectMessage {
	return <-c.electionChan
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
		if server.GetRole() == Leader {
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
