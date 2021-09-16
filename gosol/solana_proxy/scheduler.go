package solana_proxy

import (
	"gosol/solana_proxy/client"
)

type scheduler struct {
	min_block_no  int
	clients       []*client.SOLClient
	force_public  bool
	force_private bool
}

func MakeScheduler() *scheduler {

	mu.RLock()
	tmp := make([]*client.SOLClient, len(clients))
	copy(tmp, clients)
	mu.RUnlock()

	ret := &scheduler{min_block_no: -1, clients: tmp}
	ret.force_public = false
	ret.force_private = false
	return ret
}

func (this *scheduler) SetMinBlock(min_block_no int) {
	this.min_block_no = min_block_no
}

func (this *scheduler) ForcePublic(f bool) {
	this.force_public = f
	if this.force_public && this.force_private {
		this.force_public = false
		this.force_private = false
	}
}
func (this *scheduler) ForcePrivate(f bool) {
	this.force_private = f
	if this.force_public && this.force_private {
		this.force_public = false
		this.force_private = false
	}
}

/* Gets client, prioritize private client */
func (this *scheduler) GetAnyClient() *client.SOLClient {
	return this._pick_next()
}

/* Get public client only */
func (this *scheduler) GetPublicClient() *client.SOLClient {

	// we forced something, so override the client returned
	if this.force_public || this.force_private {
		return this._pick_next()
	}

	this.force_public = true
	ret := this._pick_next()
	this.force_public = false
	return ret
}

func (this *scheduler) GetAll(is_public bool, include_disabled bool) []*client.SOLClient {

	ret := make([]*client.SOLClient, 0, len(this.clients))
	for _, v := range this.clients {
		info := v.GetInfo()
		if info.Is_disabled && include_disabled == false {
			continue
		}
		if is_public != info.Is_public_node {
			continue
		}
		if this.min_block_no > -1 && this.min_block_no <= info.First_available_block {
			continue
		}
		ret = append(ret, v)
	}
	return ret
}
