/*
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// Fisher represents a fisher registered in the system
type Fisher struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	GovtID string `json:"govtId"`
	Role   string `json:"role"`
}

// Catch represents a fish catch record
type Catch struct {
	CatchID  string  `json:"catchId"`
	FisherID string  `json:"fisherId"`
	Species  string  `json:"species"`
	WeightKg float64 `json:"weightKg"`
	Date     string  `json:"date"`
}

// Batch represents a batch of catches processed together
type Batch struct {
	BatchID     string   `json:"batchId"`
	CatchIDs    []string `json:"catchIds"`
	ProcessorID string   `json:"processorId"`
	Date        string   `json:"date"`
	QRCodeURL   string   `json:"qrCodeUrl"`
}

// Order represents an order placed for a batch
type Order struct {
	OrderID string `json:"orderId"`
	BatchID string `json:"batchId"`
	BuyerID string `json:"buyerId"`
	Status  string `json:"status"`
	Date    string `json:"date"`
}

// SmartContract provides functions for managing the fisheries system
type SmartContract struct {
	contractapi.Contract
}

// RegisterFisher allows an authority to register a new fisher (stored in private data)
func (s *SmartContract) RegisterFisher(ctx contractapi.TransactionContextInterface, id, name, govtId string) error {
	if !s.hasRole(ctx, "authority") {
		return fmt.Errorf("only authority can register fishers")
	}

	fisher := Fisher{
		ID:     id,
		Name:   name,
		GovtID: govtId,
		Role:   "fisher",
	}

	fisherBytes, err := json.Marshal(fisher)
	if err != nil {
		return fmt.Errorf("failed to marshal fisher: %v", err)
	}

	// Store in private data collection "FisherCollection"
	return ctx.GetStub().PutPrivateData("FisherCollection", "FISHER_"+id, fisherBytes)
}

// GetFisher retrieves a fisher by ID from private data collection
func (s *SmartContract) GetFisher(ctx contractapi.TransactionContextInterface, fisherID string) (*Fisher, error) {
	fisherBytes, err := ctx.GetStub().GetPrivateData("FisherCollection", "FISHER_"+fisherID)
	if err != nil {
		return nil, fmt.Errorf("failed to read fisher %s: %v", fisherID, err)
	}
	if fisherBytes == nil {
		return nil, fmt.Errorf("fisher %s does not exist", fisherID)
	}

	var fisher Fisher
	err = json.Unmarshal(fisherBytes, &fisher)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal fisher data: %v", err)
	}

	return &fisher, nil
}

// LogCatch logs a new catch record
// weightKgStr is string because chaincode args are passed as strings; converted inside
func (s *SmartContract) LogCatch(ctx contractapi.TransactionContextInterface, catchId, fisherId, species, weightKgStr, date string) error {
	// Uncomment when ready to enforce access control
	/*
		if !s.hasRole(ctx, "fisher") || !s.isCaller(ctx, fisherId) {
			return fmt.Errorf("only the fisher can log their catch")
		}
	*/

	weightKg, err := strconv.ParseFloat(weightKgStr, 64)
	if err != nil {
		return fmt.Errorf("invalid weightKg value '%s': %v", weightKgStr, err)
	}

	catch := Catch{
		CatchID:  catchId,
		FisherID: fisherId,
		Species:  species,
		WeightKg: weightKg,
		Date:     date,
	}

	catchBytes, err := json.Marshal(catch)
	if err != nil {
		return fmt.Errorf("failed to marshal catch data: %v", err)
	}

	return ctx.GetStub().PutState("CATCH_"+catchId, catchBytes)
}

// CreateBatch creates a new batch record from catches
func (s *SmartContract) CreateBatch(ctx contractapi.TransactionContextInterface, batchId string, catchIds []string, processorId, date string) error {
	if !s.hasRole(ctx, "processor") {
		return fmt.Errorf("only processor can create batches")
	}

	batch := Batch{
		BatchID:     batchId,
		CatchIDs:    catchIds,
		ProcessorID: processorId,
		Date:        date,
		QRCodeURL:   fmt.Sprintf("https://getreech.example.org/batch/%s", batchId),
	}

	batchBytes, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch data: %v", err)
	}

	return ctx.GetStub().PutState("BATCH_"+batchId, batchBytes)
}

// TrackBatch retrieves batch details
func (s *SmartContract) TrackBatch(ctx contractapi.TransactionContextInterface, batchId string) (string, error) {
	batchBytes, err := ctx.GetStub().GetState("BATCH_" + batchId)
	if err != nil {
		return "", fmt.Errorf("failed to get batch %s: %v", batchId, err)
	}
	if batchBytes == nil {
		return "", fmt.Errorf("batch %s not found", batchId)
	}
	return string(batchBytes), nil
}

// PlaceOrder places a new order for a batch
func (s *SmartContract) PlaceOrder(ctx contractapi.TransactionContextInterface, orderId, batchId, buyerId, date string) error {
	if !s.hasRole(ctx, "buyer") {
		return fmt.Errorf("only buyer can place orders")
	}

	order := Order{
		OrderID: orderId,
		BatchID: batchId,
		BuyerID: buyerId,
		Status:  "placed",
		Date:    date,
	}

	orderBytes, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order data: %v", err)
	}

	return ctx.GetStub().PutState("ORDER_"+orderId, orderBytes)
}

// GenerateReport generates a JSON report of catches between dates
func (s *SmartContract) GenerateReport(ctx contractapi.TransactionContextInterface, startDate, endDate string) (string, error) {
	if !s.hasRole(ctx, "authority") {
		return "", fmt.Errorf("only authority can generate reports")
	}

	resultsIterator, err := ctx.GetStub().GetStateByRange("CATCH_", "CATCH_~")
	if err != nil {
		return "", fmt.Errorf("failed to get catches by range: %v", err)
	}
	defer resultsIterator.Close()

	var catches []Catch
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return "", fmt.Errorf("failed during results iteration: %v", err)
		}

		var catch Catch
		err = json.Unmarshal(queryResponse.Value, &catch)
		if err != nil {
			return "", fmt.Errorf("failed to unmarshal catch data: %v", err)
		}

		if catch.Date >= startDate && catch.Date <= endDate {
			catches = append(catches, catch)
		}
	}

	reportBytes, err := json.Marshal(catches)
	if err != nil {
		return "", fmt.Errorf("failed to marshal report data: %v", err)
	}

	return string(reportBytes), nil
}

// hasRole checks if the caller has the specified role attribute
func (s *SmartContract) hasRole(ctx contractapi.TransactionContextInterface, role string) bool {
	val, found, err := ctx.GetClientIdentity().GetAttributeValue("role")
	if err != nil || !found {
		return false
	}
	return val == role
}

// isCaller checks if the caller's enrollment ID matches the provided ID
func (s *SmartContract) isCaller(ctx contractapi.TransactionContextInterface, id string) bool {
	enrollmentID, found, err := ctx.GetClientIdentity().GetAttributeValue("hf.EnrollmentID")
	if err != nil || !found {
		return false
	}
	return enrollmentID == id
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating chaincode: %v\n", err)
		return
	}

	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting chaincode: %v\n", err)
	}
}
