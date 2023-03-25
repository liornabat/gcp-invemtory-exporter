package gcp_inventory_exporter

import (
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/liornabat/gcp_inventory_exporter/compute"
	"github.com/liornabat/gcp_inventory_exporter/config"
	"github.com/liornabat/gcp_inventory_exporter/network"
	"github.com/liornabat/gcp_inventory_exporter/pkg/logger"
	"github.com/liornabat/gcp_inventory_exporter/pkg/xls"
	"github.com/liornabat/gcp_inventory_exporter/project"
	"github.com/liornabat/gcp_inventory_exporter/storage"
	"net/http"
	"time"
)

func init() {
	functions.HTTP("ExportInventory", processInventory)
}
func setResponse(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Write([]byte(msg))
}

func setErrorResponse(w http.ResponseWriter, code int, err error) {
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}

func processInventory(w http.ResponseWriter, r *http.Request) {
	log := logger.NewLogger("ExportInventory", "debug")
	log.Infof("ExportInventory Started")
	cfg := config.DefaultConfig
	if err := cfg.Validate(); err != nil {
		log.Errorf("Failed to validate config: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	storageClient, err := storage.NewStorage(r.Context(), cfg.ExportProjectId)
	if err != nil {
		log.Errorf("Failed to create storage client: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	defer storageClient.Close()
	err = storageClient.BucketExistsOrCreate(r.Context(), cfg.ExportBucketName)
	if err != nil {
		log.Errorf("Failed to create bucket: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	xlsFile := xls.NewXls()

	projects, err := project.GetProjects(r.Context(), log)
	if err != nil {
		log.Errorf("Failed to get projects: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	compute, err := compute.GetComputeInventory(r.Context(), projects, cfg.Zones, log)
	if err != nil {
		log.Errorf("Failed to get compute inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.SetDataToSheet("Compute", compute); err != nil {
		log.Errorf("Failed to add compute sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	vpc, err := network.GetVPCInventory(r.Context(), projects, cfg.Regions, log)
	if err != nil {
		log.Errorf("Failed to get vpc inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.SetDataToSheet("VPC", vpc); err != nil {
		log.Errorf("Failed to add vpc sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	ipAddress, err := network.GetIPAddressInventory(r.Context(), projects, cfg.Zones, log)
	if err != nil {
		log.Errorf("Failed to get ip address inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.SetDataToSheet("IP Addresses", ipAddress); err != nil {
		log.Errorf("Failed to add ip address sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	routes, err := network.GetRoutesInventory(r.Context(), projects, log)
	if err != nil {
		log.Errorf("Failed to get routes inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.SetDataToSheet("Routes", routes); err != nil {
		log.Errorf("Failed to add routes sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	peering, err := network.GetPreeingInventory(r.Context(), projects, log)
	if err != nil {
		log.Errorf("Failed to get peering inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.SetDataToSheet("VPC Peering", peering); err != nil {
		log.Errorf("Failed to add peering sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	firewall, err := network.GetFirewallInventory(r.Context(), projects, log)
	if err != nil {
		log.Errorf("Failed to get firewall inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.SetDataToSheet("Firewall", firewall); err != nil {
		log.Errorf("Failed to add firewall sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.DeleteSheet("Sheet1"); err != nil {
		log.Errorf("Failed to delete default sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	cloudStore, err := storageClient.GetStorageInventory(r.Context(), projects, log)
	if err != nil {
		log.Errorf("Failed to get cloud storage inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	if err := xlsFile.SetDataToSheet("Cloud Storage", cloudStore); err != nil {
		log.Errorf("Failed to add cloud storage sheet: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	objectName := fmt.Sprintf("inventory-%s.xlsx", time.Now().Format("2006-01-02-15-04-05"))
	objectData, err := xlsFile.GetBytes()
	if err != nil {
		log.Errorf("Failed to get xls bytes: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	err = storageClient.SaveFile(r.Context(), cfg.ExportBucketName, objectName, objectData)
	if err != nil {
		log.Errorf("Failed to save compute inventory: %s", err.Error())
		setErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
	log.Infof("Inventory exported to gs://%s/%s", cfg.ExportBucketName, objectName)
	setResponse(w, http.StatusOK, fmt.Sprintf("Inventory exported to gs://%s/%s", cfg.ExportBucketName, objectName))
}
