package main

import (
	"fmt"
	"net/http"
	"time"
)

// TestStruct demonstrates struct definition and methods
type TestStruct struct {
	Name string
	ID   int
}

// String implements the Stringer interface
func (t TestStruct) String() string {
	return fmt.Sprintf("TestStruct{Name: %s, ID: %d}", t.Name, t.ID)
}

// TestHTTPClient demonstrates HTTP client functionality
func TestHTTPClient() error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get("https://api.github.com")
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()
	
	fmt.Printf("GitHub API responded with status: %s\n", resp.Status)
	return nil
}

// TestConcurrency demonstrates goroutines and channels
func TestConcurrency() {
	results := make(chan string, 3)
	
	// Start 3 goroutines
	for i := 1; i <= 3; i++ {
		go func(id int) {
			time.Sleep(time.Duration(id) * time.Second)
			results <- fmt.Sprintf("Worker %d completed", id)
		}(i)
	}
	
	// Collect results
	for i := 0; i < 3; i++ {
		fmt.Println(<-results)
	}
}

func main() {
	fmt.Println("Testing Go development environment...")
	
	// Test struct creation and methods
	test := TestStruct{Name: "DevContainer Test", ID: 1}
	fmt.Println(test)
	
	// Test HTTP connectivity (important for Copilot)
	if err := TestHTTPClient(); err != nil {
		fmt.Printf("HTTP test failed: %v\n", err)
	}
	
	// Test concurrency
	fmt.Println("Testing concurrency...")
	TestConcurrency()
	
	fmt.Println("Development environment test completed!")
}
