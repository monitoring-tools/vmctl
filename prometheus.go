package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/victoriametrics/vmctl/prometheus"
	"github.com/victoriametrics/vmctl/vm"
)

type prometheusProcessor struct {
	// prometheus client fetches and reads
	// snapshot blocks
	cl *prometheus.Client
	// importer performs import requests
	// for timeseries data returned from
	// snapshot blocks
	im *vm.Importer
	// cc stands for concurrency
	// and defines number of concurrently
	// running snapshot block readers
	cc int
}

func (pp *prometheusProcessor) run() error {
	blocks, s, err := pp.cl.Explore()
	if err != nil {
		return fmt.Errorf("explore failed: %s", err)
	}
	if len(blocks) < 1 {
		return fmt.Errorf("found no blocks to import")
	}
	question := fmt.Sprintf("Found %d blocks to import. Continue?", len(blocks))
	if !prompt(question) {
		return nil
	}

	bar := pb.StartNew(int(s.Series))
	blockReadersCh := make(chan tsdb.BlockReader)
	errCh := make(chan error, pp.cc)
	//go report(bar)
	var wg sync.WaitGroup
	wg.Add(pp.cc)
	for i := 0; i < pp.cc; i++ {
		go func() {
			defer wg.Done()
			for br := range blockReadersCh {
				if err := pp.do(bar, br); err != nil {
					errCh <- fmt.Errorf("read failed for block %q: %s", br.Meta().ULID, err)
					return
				}
			}
		}()
	}

	// any error breaks the import
	for _, br := range blocks {
		select {
		case promErr := <-errCh:
			close(blockReadersCh)
			return fmt.Errorf("prometheus error: %s", promErr)
		case vmErr := <-pp.im.Errors():
			close(blockReadersCh)
			var errTS string
			for _, ts := range vmErr.Batch {
				errTS += fmt.Sprintf("%s for timestamps range %d - %d\n",
					ts.String(), ts.Timestamps[0], ts.Timestamps[len(ts.Timestamps)-1])
			}
			return fmt.Errorf("Import process failed for: \n%swith error: %s", errTS, vmErr.Err)
		case blockReadersCh <- br:
		}
	}

	close(blockReadersCh)
	wg.Wait()
	// wait for all buffers to flush
	pp.im.Close()
	bar.Finish()
	log.Println("Import finished!")
	log.Print(pp.im.Stats())
	return nil
}

func report(bar *pb.ProgressBar)  {
	for {
		val := bar.Current()
		time.Sleep(time.Second)
		log.Printf("speed: %0.2fKTS/s", float64(bar.Current()-val)/1000)
	}
}
func (pp *prometheusProcessor) do(bar *pb.ProgressBar, b tsdb.BlockReader) error {
	ss, err := pp.cl.Read(b)
	if err != nil {
		return fmt.Errorf("failed to read block: %s", err)
	}
	if ss.Err() != nil {
		return fmt.Errorf("unexpected series set error: %s", err)
	}
	var labels []vm.LabelPair
	var timestamps []int64
	var values []float64
	for ss.Next() {
		var name string
		series := ss.At()

		labels = labels[:0]
		for _, label := range series.Labels() {
			if label.Name == "__name__" {
				name = label.Value
				continue
			}
			labels = append(labels, vm.LabelPair{
				Name:  label.Name,
				Value: label.Value,
			})
		}
		if name == "" {
			return fmt.Errorf("failed to find `__name__` label in labelset for block %v", b.Meta().ULID)
		}

		timestamps = timestamps[:0]
		values = values[:0]
		it := series.Iterator()
		for it.Next() {
			t, v := it.At()
			timestamps = append(timestamps, t)
			values = append(values, v)
		}
		if it.Err() != nil {
			return ss.Err()
		}
		pp.im.Input() <- &vm.TimeSeries{
			Name:       name,
			LabelPairs: append([]vm.LabelPair{}, labels...),
			Timestamps: append([]int64{}, timestamps...),
			Values:     append([]float64{}, values...),
		}
		bar.Increment()
	}
	return nil
}
