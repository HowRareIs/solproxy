package solana_proxy

import (
	"gosol/solana_proxy/client"
)

func (this *scheduler) _pick_next() *client.SOLClient {

	min, min_pos := -1, -1
	for num, v := range this.clients {
		if v == nil {
			continue
		}

		info := v.GetInfo()
		if info.Is_disabled {
			this.clients[num] = nil
			continue
		}

		if info.Is_public_node && this.force_private {
			continue
		}
		if !info.Is_public_node && this.force_public {
			continue
		}

		if this.min_block_no > -1 && this.min_block_no <= info.First_available_block {
			continue
		}

		// if the node is public it'll have limits, so get public node at the end!
		_r := int(info.Throttle.GetUsedCapacity() * 10)
		if info.Is_public_node {
			_r += 5000000
		}
		// this node is alternative, use it as last resort only
		if info.Attr&client.CLIENT_ALT > 0 {
			_r += 5000000000
		}

		if min == -1 || _r < min {
			min = _r
			min_pos = num
		}
	}

	if min_pos == -1 {
		return nil
	}
	ret := this.clients[min_pos]
	this.clients[min_pos] = nil
	return ret
}
