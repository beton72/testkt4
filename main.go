
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	mu          sync.Mutex
	logFilePath = "ecommerce.log"
)

type Product struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
}

type CartItem struct {
	Product  Product `json:"product"`
	Quantity int     `json:"quantity"`
}

type Order struct {
	Items       []CartItem `json:"items"`
	TotalAmount float64    `json:"total_amount"`
	PaymentType string     `json:"payment_type"`
	Address     string     `json:"address"`
}

var products = []Product{
	{ID: 1, Name: "Laptop", Category: "Electronics", Price: 1000.0},
	{ID: 2, Name: "Phone", Category: "Electronics", Price: 500.0},
	{ID: 3, Name: "Shoes", Category: "Fashion", Price: 50.0},
}
var cart = []CartItem{}

func initLogFile() {
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	log.SetOutput(file)
}

func searchProducts(c *gin.Context) {
	query := c.Query("q")
	category := c.Query("category")
	var results []Product
	for _, p := range products {
		if (query == "" || contains(p.Name, query)) && (category == "" || p.Category == category) {
			results = append(results, p)
		}
	}
	c.JSON(http.StatusOK, results)
}

func addToCart(c *gin.Context) {
	id := c.Query("id")
	quantity := c.Query("quantity")
	var product *Product
	for _, p := range products {
		if fmt.Sprintf("%d", p.ID) == id {
			product = &p
			break
		}
	}
	if product == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}
	mu.Lock()
	cart = append(cart, CartItem{Product: *product, Quantity: parseQuantity(quantity)})
	mu.Unlock()
	log.Printf("Added %d of %s to the cart", parseQuantity(quantity), product.Name)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Added %d of %s to the cart", parseQuantity(quantity), product.Name)})
}

func checkout(c *gin.Context) {
	var request struct {
		PaymentType string `json:"payment_type"`
		Address     string `json:"address"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if request.PaymentType == "" || request.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment type and address are required"})
		return
	}
	mu.Lock()
	total := calculateTotal()
	order := Order{
		Items:       cart,
		TotalAmount: total,
		PaymentType: request.PaymentType,
		Address:     request.Address,
	}
	cart = []CartItem{} // Clear the cart
	mu.Unlock()
	logOrder(order)
	c.JSON(http.StatusOK, gin.H{"message": "Order placed successfully!", "total": order.TotalAmount})
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr
}

func parseQuantity(quantity string) int {
	if quantity == "" {
		return 1
	}
	var q int
	fmt.Sscanf(quantity, "%d", &q)
	return q
}

func calculateTotal() float64 {
	total := 0.0
	for _, item := range cart {
		total += item.Product.Price * float64(item.Quantity)
	}
	return total
}

func logOrder(order Order) {
	log.Printf("Order Details: %+v\n", order)
}

func main() {
	initLogFile()
	r := gin.Default()
	r.GET("/search", searchProducts)
	r.POST("/add", addToCart)
	r.POST("/checkout", checkout)
	log.Println("Server running on port 8080...")
	r.Run(":8080")
}
