package arrow

// Test hooks for verifying zero-middleware dispatch paths.

func ResetZeroMiddlewareDispatchCounters() {
	resetZeroMiddlewareDispatchCounters()
}

func ZeroMiddlewareRouterDispatches() uint64 {
	return zeroMiddlewareRouterDispatches.Load()
}

func ZeroMiddlewarePipelineDispatches() uint64 {
	return zeroMiddlewarePipelineDispatches.Load()
}