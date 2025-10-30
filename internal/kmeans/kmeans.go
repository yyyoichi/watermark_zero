package kmeans

import "math"

// OneDimKmeans performs k-means clustering on one-dimensional data with k=2.
// It classifies input values into two clusters (high and low) using an iterative
// algorithm that finds optimal cluster centers.
//
// The algorithm initializes cluster centers to min and max values, then iteratively
// assigns points to clusters based on distance to centers and updates cluster centers
// to the mean of assigned points. It continues until convergence when centers stabilize
// within tolerance.
//
// The returned slice contains classification results where true indicates the high
// cluster and false indicates the low cluster.
func OneDimKmeans(averages []float64) []bool {
	var isClass01 []bool
	var center = func() [2]float64 {
		var min, max float64 = averages[0], averages[0]
		for _, v := range averages {
			if min > v {
				min = v
			}
			if max < v {
				max = v
			}
		}
		return [2]float64{min, max}
	}()
	etol := math.Pow10(-6)
	for range 300 {
		isClass01 = make([]bool, len(averages))
		threshold := (center[0] + center[1]) / 2.
		var higts, lows AverageStore
		for i, avr := range averages {
			if threshold <= avr {
				isClass01[i] = true
				higts.Add(avr)
			} else {
				lows.Add(avr)
			}
		}
		center = [2]float64{higts.Average(), lows.Average()}
		if diff := math.Abs((center[0]+center[1])/2. - threshold); diff < etol {
			break
		}
	}

	return isClass01
}
