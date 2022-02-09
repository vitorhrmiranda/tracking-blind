package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
)

type Consult struct {
	Codes []Codes `json:"consulta"`
}

type Tracking struct {
	Events []Event `json:"eventos"`
	Status string  `json:"status"`
}

type Event struct {
	Status string `json:"status"`
}

type Codes struct {
	Code string `json:"codigo"`

	Tracking Tracking `json:"tracking"`
}

func main() {
	// sync
	c := make(chan []Codes)
	wg := sync.WaitGroup{}
	guard := make(chan struct{}, 500)

	// read csv
	f, _ := os.ReadFile("orders.csv")
	r, _ := csv.NewReader(bytes.NewReader(f)).ReadAll()

	go func(r [][]string) {
		defer close(c)

		batch := 50
		codes := make([]Codes, 0, batch)
		for i := 1; i < len(r); i++ {
			codes = append(codes, Codes{Code: r[i][0]})

			if len(codes) >= batch {
				c <- codes
				codes = make([]Codes, 0, batch)
			}
		}
	}(r)

	// requests
	for input := range c {
		wg.Add(1)
		guard <- struct{}{}

		go func(i []Codes, wg *sync.WaitGroup) {
			defer wg.Done()
			defer func() { <-guard }()

			l := log.New(os.Stdout, "", 0)

			input := Consult{Codes: i}
			raw, _ := json.Marshal(input)

			request, _ := http.NewRequest("POST", "http://www.jadlog.com.br/embarcador/api/tracking/consultar", bytes.NewBuffer(raw))

			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer {{token_here}}")

			c := &http.Client{}
			response, err := c.Do(request)
			for err != nil {
				l.Printf("Error: %s\n", err)
			}

			defer response.Body.Close()
			request.Close = true

			consult := &Consult{}
			json.NewDecoder(response.Body).Decode(consult)

			for _, code := range consult.Codes {
				if len(code.Tracking.Events) > 1 {
					l.Printf("\t('%s', '%s'),\n", code.Code, code.Tracking.Status)
				}
			}
		}(input, &wg)
	}

	wg.Wait()
	log.Println("Done")
}
