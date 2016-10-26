package main

import (
	"log"
	"net/http"
	"sync"
	"time"
	"encoding/json"
	"strings"
)

type PriceTracker struct {
	frequency time.Duration
	prices    map[string]float64
	pricesMux sync.RWMutex
	// process flow
	quit      chan struct{}
	wg        sync.WaitGroup
}


// Start a small daemon that regularly gets quotes for Ether prices in various currencies.
func StartPriceTracker(frequency time.Duration) *PriceTracker {
	pt := &PriceTracker{
		frequency: frequency,
		prices: make(map[string]float64),
		quit: make(chan struct{}),
	}
	go pt.run()
	return pt
}

func (pt *PriceTracker) Get(currency string) float64 {
	pt.pricesMux.RLock()
	price := pt.prices[currency]
	pt.pricesMux.RUnlock()
	return price
}

func (pt *PriceTracker) GetAll() map[string]float64 {
	pt.pricesMux.RLock()
	ret := pt.prices
	pt.pricesMux.RUnlock()
	return ret
}

func (pt *PriceTracker) run() {
	pt.wg.Add(1)
	defer pt.wg.Done()
	ticker := time.NewTicker(pt.frequency)
	defer ticker.Stop()
	pt.update()
	for {
		select {
		case <-ticker.C:
			pt.update()
		case <-pt.quit:
			log.Println("Price Tracker is closing...")
			return
		}
	}
}

func (pt *PriceTracker) update() {
	prices, err := pt.fetch();
	if err != nil {
		log.Printf("ERROR fetching prices: %v\n", err)
		return
	}
	pt.pricesMux.Lock()
	pt.prices = prices
	pt.pricesMux.Unlock()
}

func (pt *PriceTracker) Stop() {
	pt.quit <- struct{}{}
	pt.wg.Wait()
}

func (pt *PriceTracker) fetch() (map[string]float64, error) {
	resp, err := http.Get("https://coinmarketcap-nexuist.rhcloud.com/api/eth")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var response apiResponse
	json.NewDecoder(resp.Body).Decode(&response)
	result := make(map[string]float64)
	for _, cur := range CURRENCIES {
		result[cur] = response.Price[strings.ToLower(cur)]
		log.Printf("New price for %v: %.4f\n", cur, response.Price[strings.ToLower(cur)])
	}
	return result, nil
}

type apiResponse struct {
	Price map[string]float64
}
