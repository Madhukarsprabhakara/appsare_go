package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func handlerFunc(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>Welcome to my awesome site!</h1>")
}
func getPreviousStatus(db *sql.DB, tracker_id int) string {
	var prev_status string
	err := db.QueryRow("SELECT message FROM public.tracker_events where tracker_id = $1  order by id desc limit 1", tracker_id).Scan(&prev_status)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("No previous status found for tracker %d\n", tracker_id)
		} else {
			log.Printf("Error querying previous status for tracker %d: %v\n", tracker_id, err)
		}
		return ""
	}
	return prev_status
}
func compareStatuses(db *sql.DB, tracker_id int, previous_status string, current_status string) {

}
func updateTrackerAnalytics(token string) {
	// Create the POST request
	req, err := http.NewRequest("POST", "https://appsare.com/trackeranalytics", nil)
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return
	}

	// Set the authorization header with the OAuth2 token
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the POST request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from server: %v\n", resp.Status)
	} else {
		fmt.Printf("Tracker analytics updated successfully\n")
	}
}
func sendNotifications(url string, tracker_id int, token string, service_type int) {
	// Create the data to be sent in the POST request
	data := map[string]interface{}{
		"url":          url,
		"tracker_id":   tracker_id,
		"service_type": service_type,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling JSON: %v\n", err)
		return
	}

	// Create the POST request
	req, err := http.NewRequest("POST", "https://appsare.com/snotifications", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return
	}

	// Set the authorization header with the OAuth2 token
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the POST request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from server: %v\n", resp.Status)
	} else {
		fmt.Printf("Notification sent successfully for tracker %d\n", tracker_id)
	}
}
func performTask(db *sql.DB) {
	// Your specific task goes here
	fmt.Println("Performing the task at", time.Now().Format("2006-01-02 15:04:05"))
	start := time.Now()
	// Example list of URLs to check
	rows, err := db.Query("SELECT id, url, team_id FROM public.trackers where is_active != false and is_archived = false")

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	// Create a WaitGroup to synchronize the completion of all goroutines
	var wg sync.WaitGroup

	// Create a custom HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Iterate over the rows and print the results
	for rows.Next() {
		var id int
		var url string
		var team_id int
		err := rows.Scan(&id, &url, &team_id)
		if err != nil {
			log.Fatal(err)
		}
		// Increment the WaitGroup counter
		wg.Add(1)
		go func(id int, url string, team_id int) {
			defer wg.Done()
			startTimeWTz := time.Now().UTC().Format("2006-01-02 15:04:05")
			startTime := time.Now()
			startTimeEpoch := startTime.Unix()
			resp, err := client.Head(url)
			endTime := time.Now()
			endTimeEpoch := endTime.Unix()
			responseTime := time.Since(startTime).Seconds()
			endTimeWTz := time.Now().UTC().Format("2006-01-02 15:04:05")
			if err != nil {
				var previous_status string
				previous_status = getPreviousStatus(db, id)
				fmt.Printf("Previous status for %d: %s\n", id, previous_status)
				//compare previous status with current status and send an alert if there is a change
				if previous_status != "Failed" {

					//send notifications
					fmt.Printf("Sending failed notifications at %s for tracker %d at %s\n", url, id, time.Now().Format("2006-01-02 15:04:05"))
					sendNotifications(url, id, "FwdJGH4yV1dW84CwnGPl2E58CEUKGTPCfwzO3HNhe9476059", 0)

				}
				fmt.Printf("Error checking %s: %v\n", url, err)
				// Insert the event into tracker_events table
				_, err := db.Exec("INSERT INTO tracker_events (tracker_id, response_time, http_status_code, message, start_time, end_time,epoch_start_time, epoch_end_time, exception, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
					id, responseTime, err.Error(), "Failed", startTimeWTz, endTimeWTz, startTimeEpoch, endTimeEpoch, err.Error(), startTimeWTz, startTimeWTz)
				if err != nil {

					log.Printf("Error inserting event for %s: %v\n", url, err)
				}
			} else {
				var previous_status string
				previous_status = getPreviousStatus(db, id)
				fmt.Printf("Previous status for %d: %s\n", id, previous_status)

				//compare previous status with current status and send an alert if there is a change
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				if (resp.StatusCode >= 200 && resp.StatusCode <= 299) || (resp.StatusCode >= 300 && resp.StatusCode <= 399) {
					if previous_status != "Success" {

						//send notifications
						fmt.Printf("Sending success notifications at %s for tracker %d at %s\n", url, id, time.Now().Format("2006-01-02 15:04:05"))
						sendNotifications(url, id, "FwdJGH4yV1dW84CwnGPl2E58CEUKGTPCfwzO3HNhe9476059", 1)

					}
					statusCodeStr := strconv.Itoa(resp.StatusCode)
					_, err := db.Exec("INSERT INTO tracker_events (tracker_id, response_time, http_status_code, message, start_time, end_time, epoch_start_time, epoch_end_time, exception, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
						id, responseTime, statusCodeStr, "Success", startTimeWTz, endTimeWTz, startTimeEpoch, endTimeEpoch, "", startTimeWTz, startTimeWTz)
					if err != nil {
						log.Printf("Error inserting event for %s: %v\n", url, err)
					}
				} else {
					if previous_status != "Failed" {

						//send notifications
						fmt.Printf("Sending failed notifications at %s for tracker %d at %s\n", url, id, time.Now().Format("2006-01-02 15:04:05"))
						sendNotifications(url, id, "FwdJGH4yV1dW84CwnGPl2E58CEUKGTPCfwzO3HNhe9476059", 0)

					}
					statusCodeStr := strconv.Itoa(resp.StatusCode)
					_, err := db.Exec("INSERT INTO tracker_events (tracker_id, response_time, http_status_code, message, start_time, end_time, epoch_start_time, epoch_end_time, exception, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
						id, responseTime, statusCodeStr, "Failed", startTimeWTz, endTimeWTz, startTimeEpoch, endTimeEpoch, "", startTimeWTz, startTimeWTz)
					if err != nil {
						log.Printf("Error inserting event for %s: %v\n", url, err)
					}
				}

				fmt.Printf("Checked %s: %d at %s %d\n", url, resp.StatusCode, timestamp, runtime.NumGoroutine())
				resp.Body.Close()
			}

		}(id, url, team_id)
	}

	// Check for errors from iterating over rows
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	fmt.Println("Done with everything", time.Now().Format("2006-01-02 15:04:05"))
	duration := time.Since(start)
	fmt.Println("Duration:", duration)
}
func performFifteenMinuteTask() {
	// Your specific task goes here
	fmt.Println("Performing the 15-minute task at", time.Now().Format("2006-01-02 15:04:05"))
	updateTrackerAnalytics("FwdJGH4yV1dW84CwnGPl2E58CEUKGTPCfwzO3HNhe9476059")

}
func main() {
	//Connect to the db
	db, err := sql.Open("pgx", "host=192.168.0.3 port=5433 user=postgres password=password dbname=appsare sslmode=disable")
	defer db.Close()
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected!")
	// Create a ticker that triggers every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Create another ticker that triggers every 5 minutes
	ticker15m := time.NewTicker(15 * time.Minute)
	defer ticker15m.Stop()

	// Perform the task immediately at startup
	performTask(db)
	performFifteenMinuteTask()
	go func() {
		for {
			select {
			case <-ticker.C:
				performTask(db)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ticker15m.C:
				performFifteenMinuteTask()
			}
		}
	}()
	// Keep the main function running
	select {}
}
