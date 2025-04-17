package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pterodactyl/wings/system"
)

// NodeStatsResponse represents the formatted response structure for node statistics
type NodeStatsResponse struct {
	RAMUsed       uint64  `json:"ram_used"`
	RAMTotal      uint64  `json:"ram_total"`
	RAMPercent    float64 `json:"ram_percent"`
	SwapUsed      uint64  `json:"swap_used"`
	SwapTotal     uint64  `json:"swap_total"`
	SwapPercent   float64 `json:"swap_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskRead      uint64  `json:"disk_read"`
	DiskWrite     uint64  `json:"disk_write"`
	DiskReadRate  float64 `json:"disk_read_rate"`
	DiskWriteRate float64 `json:"disk_write_rate"`
	NetIn         uint64  `json:"net_in"`
	NetOut        uint64  `json:"net_out"`
	NetInRate     float64 `json:"net_in_rate"`
	NetOutRate    float64 `json:"net_out_rate"`
	CPUUsed       float64 `json:"cpu_used"`
	CPUThreads    int     `json:"cpu_threads"`
	CPUModel      string  `json:"cpu_model"`
}

// calculatePercentage safely calculates a percentage value
func calculatePercentage(used, total uint64) float64 {
	if total > 0 {
		return float64(used) / float64(total) * 100
	}
	return 0
}

// GetNodeStats handles the request to retrieve node system statistics
// @Summary Get node system metrics
// @Description Returns system metrics including CPU, memory, disk, and network usage
// @Produce json
// @Success 200 {object} NodeStatsResponse
// @Failure 500 {object} gin.H
// @Router /api/system/stats [get]
func GetNodeStats(c *gin.Context) {
	stats, err := system.GetNodeMetrics()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error": "The metrics could not be obtained for the node",
		})
		return
	}

	// Calculate resource percentages
	ramPercent := calculatePercentage(stats.MemoryUsed, stats.MemoryTotal)
	swapPercent := calculatePercentage(stats.SwapUsed, stats.SwapTotal)
	diskPercent := calculatePercentage(stats.DiskUsed, stats.DiskTotal)

	response := NodeStatsResponse{
		RAMUsed:       stats.MemoryUsed,
		RAMTotal:      stats.MemoryTotal,
		RAMPercent:    ramPercent,
		SwapUsed:      stats.SwapUsed,
		SwapTotal:     stats.SwapTotal,
		SwapPercent:   swapPercent,
		DiskUsed:      stats.DiskUsed,
		DiskTotal:     stats.DiskTotal,
		DiskPercent:   diskPercent,
		DiskRead:      stats.DiskRead,
		DiskWrite:     stats.DiskWrite,
		DiskReadRate:  stats.DiskReadRate,
		DiskWriteRate: stats.DiskWriteRate,
		NetIn:         stats.NetIn,
		NetOut:        stats.NetOut,
		NetInRate:     stats.NetInRate,
		NetOutRate:    stats.NetOutRate,
		CPUUsed:       stats.CPUUsed,
		CPUThreads:    stats.CPUThreads,
		CPUModel:      stats.CPUModel,
	}

	c.JSON(http.StatusOK, response)
}