package main

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func setupStub(t *testing.T) (*shimtest.MockStub, *contractapi.MockTransactionContext) {
	stub := shimtest.NewMockStub("testChaincode", &SmartContract{})
	ctx := contractapi.NewMockTransactionContext(stub)
	return stub, ctx
}

func TestRegisterFisher(t *testing.T) {
	stub, ctx := setupStub(t)
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "authority")

	// Success case
	err := (&SmartContract{}).RegisterFisher(ctx, "F001", "John Doe", "GOV123")
	if err != nil {
		t.Errorf("RegisterFisher failed: %v", err)
	}
	fisherBytes, _ := stub.GetPrivateData("FisherCollection", "FISHER_F001")
	if fisherBytes == nil {
		t.Error("Fisher F001 should exist")
	}

	// Duplicate ID
	err = (&SmartContract{}).RegisterFisher(ctx, "F001", "Jane Doe", "GOV456")
	if err == nil {
		t.Error("RegisterFisher should fail on duplicate ID")
	}

	// Unauthorized access
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "fisher")
	err = (&SmartContract{}).RegisterFisher(ctx, "F002", "Jane Doe", "GOV456")
	if err == nil || err.Error() != "only authority can register fishers" {
		t.Error("RegisterFisher should fail for non-authority")
	}
}

func TestLogCatch(t *testing.T) {
	stub, ctx := setupStub(t)
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "fisher")
	ctx.GetClientIdentity().SetID("F001")

	// Success case
	err := (&SmartContract{}).LogCatch(ctx, "C001", "F001", "Tilapia", 10.5, "2025-08-09")
	if err != nil {
		t.Errorf("LogCatch failed: %v", err)
	}
	catchBytes, _ := stub.GetState("CATCH_C001")
	if catchBytes == nil {
		t.Error("Catch C001 should exist")
	}

	// Invalid weight
	err = (&SmartContract{}).LogCatch(ctx, "C002", "F001", "Tilapia", -1.0, "2025-08-09")
	if err == nil || err.Error() != "weight must be positive" {
		t.Error("LogCatch should fail for invalid weight")
	}

	// Unauthorized fisher
	ctx.GetClientIdentity().SetID("F002")
	err = (&SmartContract{}).LogCatch(ctx, "C003", "F001", "Tilapia", 5.0, "2025-08-09")
	if err == nil || err.Error() != "only the fisher can log their catch" {
		t.Error("LogCatch should fail for unauthorized fisher")
	}

	// Non-fisher role
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "processor")
	ctx.GetClientIdentity().SetID("F001")
	err = (&SmartContract{}).LogCatch(ctx, "C004", "F001", "Tilapia", 5.0, "2025-08-09")
	if err == nil || err.Error() != "only the fisher can log their catch" {
		t.Error("LogCatch should fail for non-fisher role")
	}
}

func TestCreateBatch(t *testing.T) {
	stub, ctx := setupStub(t)
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "processor")

	// Success case
	catchIds := []string{"C001", "C002"}
	err := (&SmartContract{}).CreateBatch(ctx, "B001", catchIds, "P001", "2025-08-09")
	if err != nil {
		t.Errorf("CreateBatch failed: %v", err)
	}
	batchBytes, _ := stub.GetState("BATCH_B001")
	if batchBytes == nil {
		t.Error("Batch B001 should exist")
	}

	// Unauthorized access
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "fisher")
	err = (&SmartContract{}).CreateBatch(ctx, "B002", catchIds, "P001", "2025-08-09")
	if err == nil || err.Error() != "only processor can create batches" {
		t.Error("CreateBatch should fail for non-processor")
	}
}

func TestTrackBatch(t *testing.T) {
	stub, ctx := setupStub(t)
	batch := Batch{BatchID: "B001", CatchIDs: []string{"C001"}, ProcessorID: "P001", Date: "2025-08-09", QRCodeURL: "https://getreech.example.org/batch/B001"}
	batchBytes, _ := json.Marshal(batch)
	stub.PutState("BATCH_B001", batchBytes)

	// Success case
	result, err := (&SmartContract{}).TrackBatch(ctx, "B001")
	if err != nil {
		t.Errorf("TrackBatch failed: %v", err)
	}
	if result == "" {
		t.Error("TrackBatch should return batch data")
	}

	// Non-existent batch
	_, err = (&SmartContract{}).TrackBatch(ctx, "B002")
	if err == nil || err.Error() != "batch B002 not found" {
		t.Error("TrackBatch should fail for non-existent batch")
	}
}

func TestPlaceOrder(t *testing.T) {
	stub, ctx := setupStub(t)
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "buyer")

	// Success case
	err := (&SmartContract{}).PlaceOrder(ctx, "O001", "B001", "BUY001", "2025-08-09")
	if err != nil {
		t.Errorf("PlaceOrder failed: %v", err)
	}
	orderBytes, _ := stub.GetState("ORDER_O001")
	if orderBytes == nil {
		t.Error("Order O001 should exist")
	}

	// Unauthorized access
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "fisher")
	err = (&SmartContract{}).PlaceOrder(ctx, "O002", "B001", "BUY001", "2025-08-09")
	if err == nil || err.Error() != "only buyer can place orders" {
		t.Error("PlaceOrder should fail for non-buyer")
	}
}

func TestGenerateReport(t *testing.T) {
	stub, ctx := setupStub(t)
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "authority")

	// Seed data
	catch1 := Catch{CatchID: "C001", FisherID: "F001", Species: "Tilapia", WeightKg: 10.5, Date: "2025-08-09"}
	catch2 := Catch{CatchID: "C002", FisherID: "F002", Species: "Nile Perch", WeightKg: 15.0, Date: "2025-08-10"}
	catch1Bytes, _ := json.Marshal(catch1)
	catch2Bytes, _ := json.Marshal(catch2)
	stub.PutState("CATCH_C001", catch1Bytes)
	stub.PutState("CATCH_C002", catch2Bytes)

	// Success case
	result, err := (&SmartContract{}).GenerateReport(ctx, "2025-08-09", "2025-08-10")
	if err != nil {
		t.Errorf("GenerateReport failed: %v", err)
	}
	if result == "" {
		t.Error("GenerateReport should return report data")
	}

	// Unauthorized access
	ctx.GetClientIdentity().SetAttributeValue("hf.Role", "fisher")
	_, err = (&SmartContract{}).GenerateReport(ctx, "2025-08-09", "2025-08-10")
	if err == nil || err.Error() != "only authority can generate reports" {
		t.Error("GenerateReport should fail for non-authority")
	}
}
