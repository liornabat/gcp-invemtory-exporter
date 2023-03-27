package compute

import (
	"context"
	"fmt"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"google.golang.org/api/compute/v1"
)

type machineType struct {
	Name   string
	CPU    int64
	Memory int64
}

type MachineTypes map[string]*machineType

func FetchMachineTypes(ctx context.Context, projectsId string, zone string, log *logger.Logger) MachineTypes {
	log.Infof("Getting Machine Types inventory for project %s and zone %s", projectsId, zone)
	defer log.Infof("Done getting Machine Types inventory for project %s and zone %s", projectsId, zone)
	machineTypes := make(MachineTypes)
	service, err := compute.NewService(ctx)
	if err != nil {
		return machineTypes
	}
	mt, err := service.MachineTypes.List(projectsId, zone).Do()
	if err != nil {
		return machineTypes
	}
	for _, item := range mt.Items {
		machineTypes[item.Name] = &machineType{
			Name:   item.Name,
			CPU:    item.GuestCpus,
			Memory: item.MemoryMb,
		}
	}
	return machineTypes
}
func (m MachineTypes) GetCPU(name string) string {
	if m[name] != nil {
		return fmt.Sprintf("%d", m[name].CPU)
	}
	return ""
}

func (m MachineTypes) GetMemory(name string) string {
	if m[name] != nil {
		return fmt.Sprintf("%d", m[name].Memory)
	}
	return ""
}
