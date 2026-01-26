package main

import (
	"sort"
)

// CalculatePercentile calculates the nth percentile of a slice of latency values
// Uses a more efficient approach than full sorting for large datasets
func CalculatePercentile(latencies []uint64, percentile float64) uint64 {
	if len(latencies) == 0 {
		return 0
	}

	if len(latencies) <= 1000 {
		// For small datasets, just sort and get the value
		return calculatePercentileSorted(latencies, percentile)
	}

	// For larger datasets, use selection algorithm to avoid full sort
	return calculatePercentileSelection(latencies, percentile)
}

// calculatePercentileSorted sorts the data and returns the percentile value
func calculatePercentileSorted(latencies []uint64, percentile float64) uint64 {
	sorted := make([]uint64, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	index := float64(len(sorted)-1) * (percentile / 100.0)
	ceilIndex := int(index)
	if ceilIndex >= len(sorted) {
		ceilIndex = len(sorted) - 1
	}
	return sorted[ceilIndex]
}

// calculatePercentileSelection uses quickselect algorithm for efficiency
// This avoids the O(n log n) cost of full sorting when we only need one percentile
func calculatePercentileSelection(latencies []uint64, percentile float64) uint64 {
	data := make([]uint64, len(latencies))
	copy(data, latencies)

	k := int(float64(len(data)-1) * (percentile / 100.0))
	if k >= len(data) {
		k = len(data) - 1
	}

	return quickSelect(data, k)
}

// quickSelect implements the Quickselect algorithm to find the k-th smallest element
// Average time complexity: O(n), worst case: O(n^2) but rare in practice
func quickSelect(arr []uint64, k int) uint64 {
	left := 0
	right := len(arr) - 1

	for {
		if left == right {
			return arr[left]
		}

		pivotIndex := partition(arr, left, right)

		if k == pivotIndex {
			return arr[k]
		} else if k < pivotIndex {
			right = pivotIndex - 1
		} else {
			left = pivotIndex + 1
		}
	}
}

// partition partitions the array around a pivot and returns the pivot's final position
func partition(arr []uint64, left, right int) int {
	// Choose middle element as pivot to avoid worst-case behavior
	pivotIndex := left + (right-left)/2
	pivot := arr[pivotIndex]

	// Move pivot to end
	arr[pivotIndex], arr[right] = arr[right], arr[pivotIndex]
	storeIndex := left

	for i := left; i < right; i++ {
		if arr[i] < pivot {
			arr[storeIndex], arr[i] = arr[i], arr[storeIndex]
			storeIndex++
		}
	}

	// Move pivot to its final place
	arr[storeIndex], arr[right] = arr[right], arr[storeIndex]
	return storeIndex
}

// CalculateMultiplePercentiles efficiently calculates multiple percentiles at once
// This is more efficient than calling CalculatePercentile multiple times
func CalculateMultiplePercentiles(latencies []uint64, percentiles []float64) map[float64]uint64 {
	result := make(map[float64]uint64)

	if len(latencies) == 0 {
		for _, p := range percentiles {
			result[p] = 0
		}
		return result
	}

	if len(latencies) <= 1000 {
		// For small datasets, sort once and get all percentiles
		sorted := make([]uint64, len(latencies))
		copy(sorted, latencies)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

		for _, percentile := range percentiles {
			index := float64(len(sorted)-1) * (percentile / 100.0)
			ceilIndex := int(index)
			if ceilIndex >= len(sorted) {
				ceilIndex = len(sorted) - 1
			}
			result[percentile] = sorted[ceilIndex]
		}
	} else {
		// For larger datasets, calculate each percentile separately
		// In practice, this is still efficient since our datasets are typically < 10,000 items
		for _, percentile := range percentiles {
			result[percentile] = calculatePercentileSelection(latencies, percentile)
		}
	}

	return result
}
