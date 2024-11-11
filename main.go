package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// stores pairs of ID strings and assigned points
var idScores = make(map[string]int)

type Receipt struct {
	Retailer     string
	PurchaseDate string
	PurchaseTime string
	Total        string
	Items        []Item
}

type Item struct {
	ShortDescription string
	Price            string
}

func calcPoints() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		var receipt Receipt

		dec := json.NewDecoder(req.Body)

		err := dec.Decode(&receipt)
		if err != nil {
			log.Fatal(err)
		}

		var receiptScore int

		// RULE 1: retailer name
		//
		re := regexp.MustCompile(`[^\w]`)
		validRetailerName := re.ReplaceAll([]byte(receipt.Retailer), []byte(""))

		receiptScore += len(validRetailerName)

		fmt.Fprintf(w, "Retailer name points: %d\n", receiptScore)

		// Total rules
		// Check if valid amount: only . or int, x.xx format

		priceFormat := regexp.MustCompile(`^[0-9]+\.[0-9][0-9]$`)

		match := priceFormat.Match([]byte(receipt.Total))
		if !match {
			fmt.Fprintf(w, "Invalid total: %v\n", receipt.Total)
		} else {
			// RULE 2: round total
			round, err := regexp.MatchString("^.*\\.00$", receipt.Total)
			if round {
				// 50 + (25 for Rule 3)
				receiptScore += 75
				io.WriteString(w, "Round total. \n")
			}
			if err != nil {
				io.WriteString(w, "Regex error: "+err.Error()+"\n")
			}

			// RULE 3: total is multiple of 25
			mult, err := regexp.MatchString("^.*\\.(25|50|75)$", receipt.Total)
			if mult {
				receiptScore += 25
				io.WriteString(w, "Total multiple of 25. \n")
			}
			if err != nil {
				io.WriteString(w, "Regex error: "+err.Error()+"\n")
			}
		}

		fmt.Fprintf(w, "Total after 3 rules = %v\n", receiptScore)

		// RULE 4: every two items = 5 points
		itemCount := float64(len(receipt.Items))
		receiptScore += int(5 * (math.Floor(itemCount / 2)))

		fmt.Fprintf(w, "Item count = %v\n", itemCount)

		// RULE 5: lengths of item descriptions
		for _, item := range receipt.Items {
			trimmedLen := len(strings.TrimSpace(item.ShortDescription))

			// validate item price format
			match := priceFormat.Match([]byte(item.Price))
			if !match {
				fmt.Fprintf(w, "Invalid price: %v\n", item.Price)
			} else if math.Mod(float64(trimmedLen), 3) == 0 {
				priceAsNum, err := strconv.ParseFloat(item.Price, 64)

				if err != nil {
					fmt.Fprintf(w, "Failed to parse item price: "+err.Error()+"\n")
				} else {
					receiptScore += int(math.Ceil(priceAsNum * 0.2))
				}
			}
		}

		// RULE 6: date of purchase
		// validate date format xxxx-xx-xx
		dateFormat := regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
		dateMatch := dateFormat.Match([]byte(receipt.PurchaseDate))
		if !dateMatch {
			fmt.Fprintf(w, "Invalid purchase date: %v\n", receipt.PurchaseDate)
		} else {
			day, err := strconv.ParseFloat(string(receipt.PurchaseDate[9]), 64)

			if err != nil {
				fmt.Fprintf(w, "Failed to parse purchase date: "+err.Error()+"\n")
			} else if math.Mod(day, 2) != 0 {
				receiptScore += 6
				fmt.Fprintf(w, "Odd purchase date.\n")
			}
		}

		// RULE 7: time of purchase
		// validate time format xx:xx
		timeFormat := regexp.MustCompile((`^[0-2][0-9]:[0-5][0-9]$`))
		timeMatch := timeFormat.Match([]byte(receipt.PurchaseTime))
		if !timeMatch {
			fmt.Fprintf(w, "Invalid purchase time: "+err.Error()+"\n")
		} else {
			// 14:00 to 16:00
			timeString := strings.ReplaceAll(receipt.PurchaseTime, ":", "")
			time, err := strconv.ParseFloat(timeString, 64)

			if err != nil {
				fmt.Fprintf(w, "Failed to parse purchase time: "+err.Error()+"\n")
			} else if time > 1400 && time < 1600 {
				receiptScore += 10
				fmt.Fprintf(w, "Time of purchase. \n")
			}
		}

		// Store receipt ID and score.
		id := uuid.NewString()
		idScores[id] = receiptScore

	})
}

func getPoints() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		//do stuff here
	})
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("/receipts/process", calcPoints())
	mux.Handle("GET /receipts/{id}/points", getPoints())

	http.ListenAndServe(":8080", mux)

}
