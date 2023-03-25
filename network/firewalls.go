package network

import (
	"context"
	"fmt"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"github.com/liornabat/gcp_inventory_exporter/project"
	"google.golang.org/api/compute/v1"
	"strings"
	"sync"
)

var firewallHeader = []string{
	"Project",
	"Name",
	"Network",
	"Priority",
	"Source Ranges",
	"Allowed",
	"Denied",
	"Creation Timestamp",
}

func GetFirewallInventory(ctx context.Context, projectsId []*project.Project, log *logger.Logger) ([][]string, error) {
	log.Infof("Getting Firewall inventory")
	defer log.Infof("Done getting Firewall inventory")
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, err
	}
	var inventory [][]string
	inventory = append(inventory, firewallHeader)
	mutex := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(projectsId))
	for _, projectId := range projectsId {
		go func(projectId *project.Project) {
			defer wg.Done()
			var localInventory [][]string
			log.Infof("Getting firewall inventory for project %s", projectId.Name)
			req := service.Firewalls.List(projectId.ID)
			if err := req.Pages(ctx, func(page *compute.FirewallList) error {
				for _, route := range page.Items {
					localInventory = append(localInventory, []string{
						projectId.Name,
						route.Name,
						removeUrlPrefix(route.Network),
						fmt.Sprintf("%d", route.Priority),
						strings.Join(route.SourceRanges, ","),
						allowToString(route.Allowed),
						denyToString(route.Denied),
						route.CreationTimestamp,
					})
				}
				return nil
			}); err != nil {
				log.Errorf("Failed to get firewall inventory for project %s , error: %s", projectId.Name, err.Error())
			}
			mutex.Lock()
			inventory = append(inventory, localInventory...)
			mutex.Unlock()
		}(projectId)
	}
	wg.Wait()
	return inventory, nil
}

func denyToString(denied []*compute.FirewallDenied) string {
	var deniedString []string
	for _, deny := range denied {
		deniedString = append(deniedString, fmt.Sprintf("%s:%s", deny.IPProtocol, strings.Join(deny.Ports, ",")))
	}
	return strings.Join(deniedString, ",")
}

func allowToString(allowed []*compute.FirewallAllowed) string {
	var allowedString []string
	for _, allow := range allowed {
		allowedString = append(allowedString, fmt.Sprintf("%s:%s", allow.IPProtocol, strings.Join(allow.Ports, ",")))
	}
	return strings.Join(allowedString, ",")
}
