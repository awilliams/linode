package linode

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

const (
	linodeListAction   = "linode.list"
	linodeIPListAction = "linode.ip.list"
)

// LinodeList returns slice of Linodes
func (c *Client) LinodeList() ([]Linode, error) {
	req := c.NewRequest().AddAction(linodeListAction, nil)
	var err error

	responses, err := req.GetJSON()
	if err != nil {
		return nil, err
	}
	if len(responses) != 1 {
		return nil, fmt.Errorf("unexpected number of responses: %d", len(responses))
	}

	var linodes sortedLinodes
	if responses[0].Action != linodeListAction {
		return nil, fmt.Errorf("unexpected api action %s", responses[0].Action)
	}
	if err = json.Unmarshal(responses[0].Data, &linodes); err != nil {
		return nil, err
	}
	sort.Sort(linodes)

	return []Linode(linodes), nil
}

// LinodeIPList returns mapping of LinodeID to slice of its LinodeIPs
func (c *Client) LinodeIPList(linodeIDs []int) (map[int][]LinodeIP, error) {
	req := c.NewRequest()
	var err error
	// batch all requests together
	for _, id := range linodeIDs {
		idVal := strconv.Itoa(id)
		req.AddAction(linodeIPListAction, map[string]string{"LinodeID": idVal})
	}

	responses, err := req.GetJSON()
	if err != nil {
		return nil, err
	}

	m := make(map[int][]LinodeIP, len(responses))
	for _, r := range responses {
		if r.Action != linodeIPListAction {
			return nil, fmt.Errorf("unexpected api action %s", r.Action)
		}
		var ips sortedLinodeIPs
		if err = json.Unmarshal(r.Data, &ips); err != nil {
			return nil, err
		}
		if len(ips) > 0 {
			sort.Sort(ips)
			m[ips[0].LinodeID] = []LinodeIP(ips)
		}
	}

	return m, nil
}

// Linode represent a Linode as returned by the API
type Linode struct {
	ID           int    `json:"LINODEID"`
	Status       int    `json:"STATUS"`
	Label        string `json:"LABEL"`
	DisplayGroup string `json:"LPM_DISPLAYGROUP"`
	RAM          int    `json:"TOTALRAM"`
}

// IsRunning returns true if Status == 1
func (l Linode) IsRunning() bool {
	return l.Status == 1
}

// LinodeIP respresents a Linode.IP as returned by the API
type LinodeIP struct {
	LinodeID int    `json:"LINODEID"`
	Public   int    `json:"ISPUBLIC"`
	IP       string `json:"IPADDRESS"`
}

// IsPublic returns true if IP is public
func (i LinodeIP) IsPublic() bool {
	return i.Public == 1
}

// Sort LinodeIPs by private IPs first
type sortedLinodeIPs []LinodeIP

func (sorted sortedLinodeIPs) Len() int {
	return len(sorted)
}
func (sorted sortedLinodeIPs) Swap(i, j int) {
	sorted[i], sorted[j] = sorted[j], sorted[i]
}

func (sorted sortedLinodeIPs) Less(i, j int) bool {
	return sorted[i].Public < sorted[j].Public
}

// Sort Linodes by DisplayGroup then Label
type sortedLinodes []Linode

func (sorted sortedLinodes) Len() int {
	return len(sorted)
}
func (sorted sortedLinodes) Swap(i, j int) {
	sorted[i], sorted[j] = sorted[j], sorted[i]
}

func (sorted sortedLinodes) Less(i, j int) bool {
	if sorted[i].DisplayGroup == sorted[j].DisplayGroup {
		return sorted[i].Label < sorted[j].Label
	}
	return sorted[i].DisplayGroup < sorted[j].DisplayGroup
}
