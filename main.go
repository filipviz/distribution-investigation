package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"sync"
	"time"

	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

const STEP = 20
const MAX = 8_000
const TESTS_PER = 100_000
const LOG_RESULTS = false
const PLOT = false
const REGRESSION = false

type TrialResults struct {
	avg float64
	x   int
}

func main() {
	start := time.Now()
	results := make(chan TrialResults, MAX)
	var trialGroup sync.WaitGroup
	for i := STEP; i <= MAX; i += STEP {
		trialGroup.Add(1)
		go trialFor(i, results, &trialGroup)
	}
	go func() {
		trialGroup.Wait()
		close(results)
	}()

	resultsSlice := make([]TrialResults, MAX/STEP)
	i := 0
	for trialResult := range results {
		resultsSlice[i] = trialResult
		i++
	}
	slices.SortFunc(resultsSlice, func(a, b TrialResults) int {
		if a.x < b.x {
			return -1
		}
		return 1
	})
	fmt.Printf("Took %v to run %d trials, each with %d tests.\n", time.Since(start), MAX/STEP, TESTS_PER)

	for _, res := range resultsSlice {
		fmt.Printf("%f, ", float64(res.x))
	}
	fmt.Print("\n\n")
	for _, res := range resultsSlice {
		fmt.Printf("%f, ", res.avg)
	}
	fmt.Print("\n\n")

	if PLOT {
		// Plot
		fmt.Println("Plotting...")
		p := plot.New()
		p.Title.Text = "Random Maximum vs. Average Number of Loops"
		p.X.Label.Text = "Maximum Random Number"
		p.Y.Label.Text = "Average Number of Loops"
		pts := make(plotter.XYs, len(resultsSlice))
		for i, res := range resultsSlice {
			pts[i].X = float64(res.x)
			pts[i].Y = res.avg
		}
		err := plotutil.AddLinePoints(p, "Points", pts)
		if err != nil {
			fmt.Printf("Could not add line points: %v\n", err)
			os.Exit(1)
		}

		if err = p.Save(4*vg.Inch, 4*vg.Inch, "results.png"); err != nil {
			fmt.Println("Error saving plot to file: ", err)
			os.Exit(1)
		}
	}

	if REGRESSION {
		// Regression
		fmt.Println("Running regression...")
		xVals := make([]float64, len(resultsSlice))
		yVals := make([]float64, len(resultsSlice))
		logXVals := make([]float64, len(resultsSlice))
		logYVals := make([]float64, len(resultsSlice))
		for i, res := range resultsSlice {
			if res.avg < 0.0 {
				fmt.Printf("error: cannot perform regression on negative y value (avg) for result %+v\n", res)
				os.Exit(1)
			}
			xVals[i] = float64(res.x)
			yVals[i] = res.avg
			logXVals[i] = math.Log(float64(res.x))
			logYVals[i] = math.Log(res.avg)
		}

		aLog, bLog := stat.LinearRegression(logXVals, yVals, nil, false)
		aExp, bExp := stat.LinearRegression(xVals, logYVals, nil, false)
		aExp = math.Exp(aExp)

		fmt.Printf("Log model: y = %f * log(x) + %f\n", aLog, bLog)
		fmt.Printf("Exp model: y = %f * exp(%f*x)\n", aExp, bExp)
	}

	if LOG_RESULTS {
		fmt.Printf("%8s: %-8s\n", "Rand Max", "Average")
		for _, res := range resultsSlice {
			fmt.Printf("%8d: %-8f\n", res.x, res.avg)
		}
	}
	fmt.Printf("Took %v to finish.\n", time.Since(start))
}

func trialFor(x int, results chan TrialResults, trialGroup *sync.WaitGroup) {
	defer trialGroup.Done()

	var testGroup sync.WaitGroup
	r := make(chan int, TESTS_PER)
	for i := 0; i < TESTS_PER; i++ {
		testGroup.Add(1)
		go test(x, r, &testGroup)
	}
	go func() {
		testGroup.Wait()
		close(r)
	}()

	total := 0
	for count := range r {
		total += count
	}

	final := float64(total) / TESTS_PER
	results <- TrialResults{final, x}
}

func test(x int, r chan int, wg *sync.WaitGroup) {
	defer wg.Done()
	count := 0
	for i := 0; i < rand.Intn(x)+1; i++ {
		count++
	}
	r <- count
}
