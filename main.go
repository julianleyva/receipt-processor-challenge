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
var idScores = make(map[string]int64)

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

		// Check if Items has at least 1 item
		if len(receipt.Items) == 0 {
			invalidReceipt(w)
			return
		}

		var receiptScore int

		// RULE 1: retailer name
		//
		retailerFormat := regexp.MustCompile(`^[\w\s\-&]+$`)
		retailerMatch := retailerFormat.Match([]byte(receipt.Retailer))
		if !retailerMatch {
			invalidReceipt(w)
			return
		}

		receiptScore += len(receipt.Retailer)

		// Total rules
		// Check if valid amount
		priceFormat := regexp.MustCompile(`^\d+\.\d{2}$`)

		match := priceFormat.Match([]byte(receipt.Total))
		if !match {
			invalidReceipt(w)
			return
		} else {
			// RULE 2: round total
			round, err := regexp.MatchString(`^.*\.00$`, receipt.Total)
			if round {
				// 50 + (25 for Rule 3)
				receiptScore += 75
			}
			if err != nil {
				io.WriteString(w, "Regex error: "+err.Error()+"\n")
			}

			// RULE 3: total is multiple of 25
			mult, err := regexp.MatchString(`^.*\.(25|50|75)$`, receipt.Total)
			if mult {
				receiptScore += 25
			}
			if err != nil {
				io.WriteString(w, "Regex error: "+err.Error()+"\n")
			}
		}

		// RULE 4: every two items = 5 points
		itemCount := float64(len(receipt.Items))
		receiptScore += int(5 * (math.Floor(itemCount / 2)))

		// RULE 5: lengths of item descriptions
		for _, item := range receipt.Items {

			descFormat := regexp.MustCompile(`^[\\w\\s\\-]+$`)
			descMatch := descFormat.Match([]byte(item.ShortDescription))
			if !descMatch {
				invalidReceipt(w)
				return
			}

			trimmedLen := len(strings.TrimSpace(item.ShortDescription))

			// validate item price format
			match := priceFormat.Match([]byte(item.Price))
			if !match {
				invalidReceipt(w)
				return
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
			invalidReceipt(w)
			return
		} else {
			day, err := strconv.ParseFloat(string(receipt.PurchaseDate[9]), 64)

			if err != nil {
				fmt.Fprintf(w, "Failed to parse purchase date: "+err.Error()+"\n")
			} else if math.Mod(day, 2) != 0 {
				receiptScore += 6
			}
		}

		// RULE 7: time of purchase
		// validate time format xx:xx
		timeFormat := regexp.MustCompile((`^[0-2][0-9]:[0-5][0-9]$`))
		timeMatch := timeFormat.Match([]byte(receipt.PurchaseTime))
		if !timeMatch {
			invalidReceipt(w)
			return
		} else {
			// 14:00 to 16:00
			timeString := strings.ReplaceAll(receipt.PurchaseTime, ":", "")
			time, err := strconv.ParseFloat(timeString, 64)

			if err != nil {
				fmt.Fprintf(w, "Failed to parse purchase time: "+err.Error()+"\n")
			} else if time > 1400 && time < 1600 {
				receiptScore += 10
			}
		}

		// Store receipt ID and score.
		id := uuid.NewString()
		idScores[id] = int64(receiptScore)

		// Prepare JSON response.
		error := json.NewEncoder(w).Encode(map[string]string{"id": id})
		if error != nil {
			fmt.Fprintf(w, "JSON encoder error: "+err.Error()+"\n")
		}

	})
}

func invalidReceipt(w http.ResponseWriter) {
	w.WriteHeader(400)
	fmt.Fprintf(w, "The receipt is invalid.\n")
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
