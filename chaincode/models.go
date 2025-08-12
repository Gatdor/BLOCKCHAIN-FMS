package main

// Fisher represents a registered fisher
type Fisher struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	GovtID string `json:"govtId"`
	Role   string `json:"role"` // e.g., "fisher"
}

// Catch represents a fishing catch log
type Catch struct {
	CatchID  string  `json:"catchId"`
	FisherID string  `json:"fisherId"`
	Species  string  `json:"species"`
	WeightKg float64 `json:"weightKg"`
	Date     string  `json:"date"`
}

// Batch represents a processed batch of catches
type Batch struct {
	BatchID     string   `json:"batchId"`
	CatchIDs    []string `json:"catchIds"`
	ProcessorID string   `json:"processorId"`
	Date        string   `json:"date"`
	QRCodeURL   string   `json:"qrCodeUrl"`
}

// Order represents a buyer order
type Order struct {
	OrderID string `json:"orderId"`
	BatchID string `json:"batchId"`
	BuyerID string `json:"buyerId"`
	Status  string `json:"status"` // e.g., "placed", "shipped"
	Date    string `json:"date"`
}
