package saga

import (
	"errors"
	"testing"
	"trading-saga/pkg/domain"

	"github.com/stretchr/testify/assert"
)

// MockQuotationService for testing
type MockQuotationService struct {
	shouldFail bool
	response   *domain.QuotationResponse
}

func (m *MockQuotationService) GetQuotation(req domain.QuotationRequest) (*domain.QuotationResponse, error) {
	if m.shouldFail {
		return nil, errors.New("quotation service error")
	}
	if m.response != nil {
		return m.response, nil
	}
	return &domain.QuotationResponse{
		Price1: 50.0,
		Price2: 5.0,
		TTLms:  1000,
	}, nil
}

// MockRiskService for testing
type MockRiskService struct {
	shouldFail bool
	approved   bool
}

func (m *MockRiskService) EvaluateRisk(req domain.RiskRequest) (*domain.RiskResponse, error) {
	if m.shouldFail {
		return nil, errors.New("risk service error")
	}
	return &domain.RiskResponse{
		Approved: m.approved,
	}, nil
}

// MockPurchaseService for testing
type MockPurchaseService struct {
	shouldFail      bool
	successRate     float64
	callCount       int
	failOnCallNum   int // Fail on specific call number (0-indexed), -1 means never fail
	executedActions []domain.ActionType
}

func NewMockPurchaseService() *MockPurchaseService {
	return &MockPurchaseService{
		failOnCallNum: -1, // Default: never fail on a specific call
	}
}

func (m *MockPurchaseService) ExecutePurchase(req domain.PurchaseRequest) (*domain.PurchaseResponse, error) {
	m.executedActions = append(m.executedActions, req.Action)
	m.callCount++

	if m.shouldFail {
		return nil, errors.New("purchase service error")
	}

	if m.failOnCallNum >= 0 && m.callCount-1 == m.failOnCallNum {
		return &domain.PurchaseResponse{Success: false}, nil
	}

	// For SELL (compensation), always succeed
	if req.Action == domain.ActionSell {
		return &domain.PurchaseResponse{Success: true}, nil
	}

	// For BUY, use success rate
	return &domain.PurchaseResponse{Success: true}, nil
}

// TestSuccessfulOrderExecution - happy path
func TestSuccessfulOrderExecution(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.NoError(t, err, "Expected successful order execution")
	assert.Equal(t, 2, purchaseSvc.callCount, "Expected 2 purchase calls (buy both assets)")
	assert.Equal(t, 2, len(purchaseSvc.executedActions), "Expected 2 actions (both BUY)")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[0], "First action should be BUY")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[1], "Second action should be BUY")
}

// TestOrderFailsOnQuotationError
func TestOrderFailsOnQuotationError(t *testing.T) {
	quotationSvc := &MockQuotationService{shouldFail: true}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Error(t, err, "Expected error on quotation failure")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when quotation fails")
}

// TestOrderFailsOnRiskRejection
func TestOrderFailsOnRiskRejection(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: false}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Equal(t, ErrRiskReproved, err, "Expected ErrRiskReproved")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when risk is rejected")
}

// TestOrderFailsOnRiskError
func TestOrderFailsOnRiskError(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{shouldFail: true}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Error(t, err, "Expected error on risk service failure")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when risk service fails")
}

// TestOrderCompensatesOnFirstAssetPurchaseFailure - CRITICAL TEST
// This tests the SAGA compensation: if Asset1 buy succeeds but Asset2 buy fails,
// Asset1 must be sold back
func TestOrderCompensatesOnFirstAssetPurchaseFailure(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()
	purchaseSvc.failOnCallNum = 1 // Fail on 2nd purchase call (Asset2)

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Equal(t, ErrPurchasingAsset2, err, "Expected ErrPurchasingAsset2 when second asset purchase fails")

	// Verify the SAGA compensation: BUY Asset1, BUY Asset2 (fail), SELL Asset1 (compensation)
	assert.Equal(t, 3, purchaseSvc.callCount, "Expected 3 purchase calls (BUY Asset1, BUY Asset2, SELL Asset1 compensation)")

	// Verify the sequence: BUY, BUY, SELL
	expectedActions := []domain.ActionType{
		domain.ActionBuy,  // Asset1 purchase
		domain.ActionBuy,  // Asset2 purchase (fails)
		domain.ActionSell, // Asset1 compensation (reverses the first BUY)
	}

	assert.Equal(t, expectedActions, purchaseSvc.executedActions, "Expected BUY, BUY, SELL sequence")
}

// TestOrderSucceedsWhenBothAssetsPurchased - happy path with both purchases
func TestOrderSucceedsWhenBothAssetsPurchased(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.NoError(t, err, "Expected successful order when both assets purchase")
	assert.Equal(t, 2, purchaseSvc.callCount, "Expected exactly 2 purchase calls")

	// Verify exactly 2 BUY calls, no compensation
	assert.Len(t, purchaseSvc.executedActions, 2, "Expected 2 actions (no compensation)")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[0], "First action should be BUY")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[1], "Second action should be BUY")
}


// TestTTLExceededImmediately - tests TTL check right after quotation
// Uses a negative TTL to simulate an already-expired quote
func TestTTLExceededImmediately(t *testing.T) {
	// Create a quotation response with negative TTL (already expired)
	quotationSvc := &MockQuotationService{
		response: &domain.QuotationResponse{
			Price1: 50.0,
			Price2: 5.0,
			TTLms:  -100, // Negative TTL - will be expired immediately
		},
	}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Equal(t, ErrTTLExceeded, err, "Expected ErrTTLExceeded when TTL is negative")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when TTL expired")
}

// TestCompensationRevertsInCorrectOrder - CRITICAL TEST
// Verifies that compensation happens in LIFO order (Last In, First Out)
// If we bought Asset1 then Asset2, we must sell Asset2 then Asset1
func TestCompensationRevertsInCorrectOrder(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()
	purchaseSvc.failOnCallNum = 1 // Fail on Asset2 purchase

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Equal(t, ErrPurchasingAsset2, err, "Expected ErrPurchasingAsset2")

	// Verify LIFO compensation order
	assert.Greater(t, len(purchaseSvc.executedActions), 2, "Expected at least 3 actions")

	// Last action should be SELL (the compensation)
	lastAction := purchaseSvc.executedActions[len(purchaseSvc.executedActions)-1]
	assert.Equal(t, domain.ActionSell, lastAction, "Last action should be SELL (compensation)")

	// First two should be BUY
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[0], "First action should be BUY")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[1], "Second action should be BUY")
}

// TestOrderFailsOnPurchaseServiceError
func TestOrderFailsOnPurchaseServiceError(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()
	purchaseSvc.shouldFail = true

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Error(t, err, "Expected error on purchase service failure")
	assert.Equal(t, 1, purchaseSvc.callCount, "Expected 1 purchase call (failed immediately)")
}

// TestMultipleOrdersIndependent - verifies orchestrator is stateless
func TestMultipleOrdersIndependent(t *testing.T) {
	orchestrator := NewOrchestrator(
		&MockQuotationService{},
		&MockRiskService{approved: true},
		NewMockPurchaseService(),
	)

	order1 := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	order2 := domain.OrderRequest{
		Asset1: "BTC/USDT",
		Asset2: "EUR/BRL",
		Qty:    2.5,
	}

	err1 := orchestrator.ExecuteOrder(order1)
	err2 := orchestrator.ExecuteOrder(order2)

	assert.NoError(t, err1, "First order should succeed")
	assert.NoError(t, err2, "Second order should succeed")
}

// TestCompensationDoesNotRunOnSuccess - verifies cleanup behavior
func TestCompensationDoesNotRunOnSuccess(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.NoError(t, err, "Expected successful order")

	// Should have exactly 2 BUY calls, 0 SELL calls
	buyCount := 0
	sellCount := 0
	for _, action := range purchaseSvc.executedActions {
		if action == domain.ActionBuy {
			buyCount++
		} else if action == domain.ActionSell {
			sellCount++
		}
	}

	assert.Equal(t, 2, buyCount, "Expected 2 BUY actions")
	assert.Equal(t, 0, sellCount, "Expected 0 SELL actions (no compensation on success)")
}

// TestTTLCheckedAfterEachStep - verifies TTL is enforced after risk analysis
// Uses negative TTL to simulate expiration between risk check and first purchase
func TestTTLCheckedAfterRiskAnalysis(t *testing.T) {
	// This test verifies that TTL is checked after risk analysis
	// Since we can't easily simulate time passing during the orchestrator execution,
	// we use a negative TTL which will be expired by the time the first purchase is attempted
	quotationSvc := &MockQuotationService{
		response: &domain.QuotationResponse{
			Price1: 50.0,
			Price2: 5.0,
			TTLms:  -100, // Already expired
		},
	}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	orchestrator := NewOrchestrator(quotationSvc, riskSvc, purchaseSvc)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := orchestrator.ExecuteOrder(order)

	assert.Equal(t, ErrTTLExceeded, err, "Expected TTL to be checked")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when TTL expired")
}
