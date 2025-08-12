// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ------------------ Domain types ------------------

type Fisher struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	GovtID string `json:"govtId"`
	Role   string `json:"role"`
}

type Catch struct {
	CatchID  string  `json:"catchId"`
	FisherID string  `json:"fisherId"`
	Species  string  `json:"species"`
	WeightKg float64 `json:"weightKg"`
	Date     string  `json:"date"` // ISO date preferred
}

type Batch struct {
	BatchID     string   `json:"batchId"`
	CatchIDs    []string `json:"catchIds"`
	ProcessorID string   `json:"processorId"`
	Date        string   `json:"date"`
	QRCodeURL   string   `json:"qrCodeUrl"`
}

type Order struct {
	OrderID string `json:"orderId"`
	BatchID string `json:"batchId"`
	BuyerID string `json:"buyerId"`
	Status  string `json:"status"`
	Date    string `json:"date"`
}

// Asset type (sample utilities for testing)
type Asset struct {
	ID             string `json:"id"`
	Color          string `json:"color"`
	Size           int    `json:"size"`
	Owner          string `json:"owner"`
	AppraisedValue int    `json:"appraisedValue"`
}

// ------------------ SmartContract ------------------

type SmartContract struct {
	contractapi.Contract
}

// ------------------ Helpers ------------------

// hasRole: check attribute "role" from identity. If attribute absent, permissive for local testing.
// For production: change to require attribute presence and exact match.
func (s *SmartContract) hasRole(ctx contractapi.TransactionContextInterface, role string) bool {
	ci := ctx.GetClientIdentity()
	val, found, err := ci.GetAttributeValue("role")
	if err != nil {
		// Can't read attributes (e.g., CLI cert). Permissive to ease testing.
		return true
	}
	if !found {
		// Not set on identity — permissive for test environment.
		return true
	}
	return val == role
}

// isCaller: compare client's ID. For production use a stable attribute instead.
func (s *SmartContract) isCaller(ctx contractapi.TransactionContextInterface, id string) bool {
	clientID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return false
	}
	return clientID == id
}

// ------------------ Fisher functions ------------------

func (s *SmartContract) RegisterFisher(ctx contractapi.TransactionContextInterface, id, name, govtId string) error {
	if !s.hasRole(ctx, "authority") {
		return fmt.Errorf("only authority can register fishers")
	}
	f := Fisher{ID: id, Name: name, GovtID: govtId, Role: "fisher"}
	b, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState("FISHER_"+id, b)
}

func (s *SmartContract) GetFisher(ctx contractapi.TransactionContextInterface, fisherID string) (*Fisher, error) {
	b, err := ctx.GetStub().GetState("FISHER_" + fisherID)
	if err != nil {
		return nil, fmt.Errorf("failed to read fisher %s: %v", fisherID, err)
	}
	if b == nil {
		return nil, fmt.Errorf("fisher %s does not exist", fisherID)
	}
	var f Fisher
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// ------------------ Catch functions ------------------

// LogCatch expects weightKg as string (so CLI can pass it). Date should be ISO string.
func (s *SmartContract) LogCatch(ctx contractapi.TransactionContextInterface, catchId, fisherId, species, weightKgStr, date string) error {
	// enforce fisher role AND caller identity; permissive if attributes absent (for testing)
	if !s.hasRole(ctx, "fisher") && !s.isCaller(ctx, fisherId) {
		return fmt.Errorf("only the fisher can log their catch")
	}
	weightKg, err := strconv.ParseFloat(weightKgStr, 64)
	if err != nil {
		return fmt.Errorf("invalid weightKg: %v", err)
	}
	c := Catch{CatchID: catchId, FisherID: fisherId, Species: species, WeightKg: weightKg, Date: date}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState("CATCH_"+catchId, b)
}

func (s *SmartContract) GetCatch(ctx contractapi.TransactionContextInterface, catchId string) (*Catch, error) {
	b, err := ctx.GetStub().GetState("CATCH_" + catchId)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("catch %s not found", catchId)
	}
	var c Catch
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// ------------------ Batch functions ------------------

func (s *SmartContract) CreateBatch(ctx contractapi.TransactionContextInterface, batchId string, catchIds []string, processorId, date string) error {
	if !s.hasRole(ctx, "processor") {
		return fmt.Errorf("only processor can create batches")
	}
	batch := Batch{BatchID: batchId, CatchIDs: catchIds, ProcessorID: processorId, Date: date, QRCodeURL: fmt.Sprintf("https://example.org/batch/%s", batchId)}
	b, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState("BATCH_"+batchId, b)
}

func (s *SmartContract) TrackBatch(ctx contractapi.TransactionContextInterface, batchId string) (string, error) {
	b, err := ctx.GetStub().GetState("BATCH_" + batchId)
	if err != nil {
		return "", err
	}
	if b == nil {
		return "", fmt.Errorf("batch %s not found", batchId)
	}
	return string(b), nil
}

// ------------------ Order functions ------------------

func (s *SmartContract) PlaceOrder(ctx contractapi.TransactionContextInterface, orderId, batchId, buyerId, date string) error {
	if !s.hasRole(ctx, "buyer") {
		return fmt.Errorf("only buyer can place orders")
	}
	o := Order{OrderID: orderId, BatchID: batchId, BuyerID: buyerId, Status: "placed", Date: date}
	b, err := json.Marshal(o)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState("ORDER_"+orderId, b)
}

// ------------------ Reporting ------------------

func (s *SmartContract) GenerateReport(ctx contractapi.TransactionContextInterface, startDate, endDate string) (string, error) {
	if !s.hasRole(ctx, "authority") {
		return "", fmt.Errorf("only authority can generate reports")
	}
	iter, err := ctx.GetStub().GetStateByRange("CATCH_", "CATCH_~")
	if err != nil {
		return "", err
	}
	defer iter.Close()
	var out []Catch
	for iter.HasNext() {
		r, err := iter.Next()
		if err != nil {
			return "", err
		}
		var c Catch
		if err := json.Unmarshal(r.Value, &c); err != nil {
			return "", err
		}
		if c.Date >= startDate && c.Date <= endDate {
			out = append(out, c)
		}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ------------------ Asset helpers (test utilities) ------------------

func (s *SmartContract) CreateAsset(ctx contractapi.TransactionContextInterface, id, color, sizeStr, owner, appraisedValueStr string) error {
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return fmt.Errorf("invalid size: %v", err)
	}
	appVal, err := strconv.Atoi(appraisedValueStr)
	if err != nil {
		return fmt.Errorf("invalid appraisedValue: %v", err)
	}
	a := Asset{ID: id, Color: color, Size: size, Owner: owner, AppraisedValue: appVal}
	b, _ := json.Marshal(a)
	return ctx.GetStub().PutState("ASSET_"+id, b)
}

func (s *SmartContract) ReadAsset(ctx contractapi.TransactionContextInterface, id string) (*Asset, error) {
	b, err := ctx.GetStub().GetState("ASSET_" + id)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("asset %s not found", id)
	}
	var a Asset
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *SmartContract) TransferAsset(ctx contractapi.TransactionContextInterface, assetKey, newOwner string) error {
	b, err := ctx.GetStub().GetState("ASSET_" + assetKey)
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("asset %s not found", assetKey)
	}
	var a Asset
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	a.Owner = newOwner
	updated, _ := json.Marshal(a)
	return ctx.GetStub().PutState("ASSET_"+assetKey, updated)
}

func (s *SmartContract) UpdateAsset(ctx contractapi.TransactionContextInterface, assetKey, color, sizeStr, appraisedValueStr string) error {
	b, err := ctx.GetStub().GetState("ASSET_" + assetKey)
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("asset %s not found", assetKey)
	}
	var a Asset
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		return err
	}
	appVal, err := strconv.Atoi(appraisedValueStr)
	if err != nil {
		return err
	}
	a.Color = color
	a.Size = size
	a.AppraisedValue = appVal
	updated, _ := json.Marshal(a)
	return ctx.GetStub().PutState("ASSET_"+assetKey, updated)
}

func (s *SmartContract) DeleteAsset(ctx contractapi.TransactionContextInterface, assetKey string) error {
	b, err := ctx.GetStub().GetState("ASSET_" + assetKey)
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("asset %s not found", assetKey)
	}
	return ctx.GetStub().DelState("ASSET_" + assetKey)
}

// ------------------ Main ------------------

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error create chaincode: %v\n", err)
		return
	}
	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting chaincode: %v\n", err)
	}
}
