package atrust

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/nnu-connect/nnu-connect/log"
)

type nodeGroup struct {
	wan []string
	lan []string
}

func (c *Client) probeNodes(ctx context.Context) map[string]string {
	best := make(map[string]string)
	for group, nodes := range c.NodeGroups {
		node := c.probeNodeGroup(ctx, nodes.wan)
		if node == "" {
			node = c.probeNodeGroup(ctx, nodes.lan)
		}
		if node != "" {
			best[group] = node
			log.Printf("aTrust TLS node for group %s: %s", group, node)
		}
	}
	return best
}

func (c *Client) probeNodeGroup(ctx context.Context, nodes []string) string {
	if len(nodes) == 0 || ctx.Err() != nil {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	results := make(chan string, len(nodes))
	for _, node := range nodes {
		go func() {
			conn, err := c.underlayDialer.DialTLSContext(ctx, "tcp", node, &tls.Config{InsecureSkipVerify: true})
			if err != nil {
				log.DebugPrintf("aTrust node %s TLS probe failed: %v", node, err)
				results <- ""
				return
			}
			results <- node
			_ = conn.Close()
		}()
	}
	for range nodes {
		if node := <-results; node != "" {
			return node
		}
	}
	return ""
}

func (c *Client) updateBestNodes(ctx context.Context, updateBestNodesInterval int) {
	ticker := time.NewTicker(time.Duration(updateBestNodesInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		bestNodes := c.probeNodes(ctx)
		c.BestNodesRWMutex.Lock()
		for group, node := range bestNodes {
			c.BestNodes[group] = node
		}
		c.BestNodesRWMutex.Unlock()
	}
}
