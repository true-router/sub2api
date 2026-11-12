package handler

// PendingUsageRecords includes queued and running persistence tasks.
func (h *GatewayHandler) PendingUsageRecords() uint64 {
	if h == nil || h.usageRecordWorkerPool == nil {
		return 0
	}
	stats := h.usageRecordWorkerPool.Stats()
	if stats.CompletedTasks > stats.SubmittedTasks {
		return 1
	}
	return stats.SubmittedTasks - stats.CompletedTasks
}
