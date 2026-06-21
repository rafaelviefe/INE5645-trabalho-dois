package application

import (
	"errors"
	"testing"
	"trading-saga/pkg/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockQuotationService struct {
	shouldFail bool
	response   *domain.QuotationResponse
}

func (m *MockQuotationService) Get(req domain.QuotationRequest) (*domain.QuotationResponse, error) {
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

type MockRiskService struct {
	shouldFail bool
	approved   bool
}

func (m *MockRiskService) Evaluate(req domain.RiskRequest) (*domain.RiskResponse, error) {
	if m.shouldFail {
		return nil, errors.New("risk service error")
	}
	return &domain.RiskResponse{
		Approved: m.approved,
	}, nil
}

type MockPurchaseService struct {
	shouldFail      bool
	successRate     float64
	callCount       int
	failOnCallNum   int
	executedActions []domain.ActionType
}

func NewMockPurchaseService() *MockPurchaseService {
	return &MockPurchaseService{
		failOnCallNum: -1,
	}
}

func (m *MockPurchaseService) Execute(req domain.TradeExecution) (*domain.TradeResponse, error) {
	m.executedActions = append(m.executedActions, req.Action)
	m.callCount++

	if m.shouldFail {
		return nil, errors.New("purchase service error")
	}

	if m.failOnCallNum >= 0 && m.callCount-1 == m.failOnCallNum {
		return &domain.TradeResponse{Success: false}, nil
	}

	if req.Action == domain.ActionSell {
		return &domain.TradeResponse{Success: true}, nil
	}

	return &domain.TradeResponse{Success: true}, nil
}

func TestSuccessfulOrderExecution(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.NoError(t, err, "Expected successful order execution")
	assert.Equal(t, 2, purchaseSvc.callCount, "Expected 2 purchase calls (buy both assets)")
	assert.Equal(t, 2, len(purchaseSvc.executedActions), "Expected 2 actions (both BUY)")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[0], "First action should be BUY")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[1], "Second action should be BUY")
}

func TestOrderFailsOnQuotationError(t *testing.T) {
	quotationSvc := &MockQuotationService{shouldFail: true}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.Error(t, err, "Expected error on quotation failure")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when quotation fails")
}

func TestOrderFailsOnRiskRejection(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: false}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.True(t, errors.Is(err, ErrRiskReproved), "Expected ErrRiskReproved")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when risk is rejected")
}

func TestOrderFailsOnRiskError(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{shouldFail: true}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.Error(t, err, "Expected error on risk service failure")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when risk service fails")
}

func TestOrderCompensatesOnFirstAssetPurchaseFailure(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()
	purchaseSvc.failOnCallNum = 1

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPurchasingAsset2), "Expected ErrPurchasingAsset2 when second asset purchase fails")

	assert.Equal(t, 3, purchaseSvc.callCount, "Expected 3 purchase calls (BUY Asset1, BUY Asset2, SELL Asset1 compensation)")

	expectedActions := []domain.ActionType{
		domain.ActionBuy,
		domain.ActionBuy,
		domain.ActionSell,
	}

	assert.Equal(t, expectedActions, purchaseSvc.executedActions, "Expected BUY, BUY, SELL sequence")
}

func TestOrderSucceedsWhenBothAssetsPurchased(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.NoError(t, err, "Expected successful order when both assets purchase")
	assert.Equal(t, 2, purchaseSvc.callCount, "Expected exactly 2 purchase calls")
	assert.Len(t, purchaseSvc.executedActions, 2, "Expected 2 actions (no compensation)")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[0], "First action should be BUY")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[1], "Second action should be BUY")
}

func TestTTLExceededImmediately(t *testing.T) {
	quotationSvc := &MockQuotationService{
		response: &domain.QuotationResponse{
			Price1: 50.0,
			Price2: 5.0,
			TTLms:  -100,
		},
	}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.True(t, errors.Is(err, ErrTTLExceeded), "Expected ErrTTLExceeded when TTL is negative")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when TTL expired")
}

func TestCompensationRevertsInCorrectOrder(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()
	purchaseSvc.failOnCallNum = 1

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPurchasingAsset2), "Expected ErrPurchasingAsset2")
	assert.Greater(t, len(purchaseSvc.executedActions), 2, "Expected at least 3 actions")

	lastAction := purchaseSvc.executedActions[len(purchaseSvc.executedActions)-1]
	assert.Equal(t, domain.ActionSell, lastAction, "Last action should be SELL (compensation)")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[0], "First action should be BUY")
	assert.Equal(t, domain.ActionBuy, purchaseSvc.executedActions[1], "Second action should be BUY")
}

func TestOrderFailsOnPurchaseServiceError(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()
	purchaseSvc.shouldFail = true

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.Error(t, err, "Expected error on purchase service failure")
	assert.Equal(t, 1, purchaseSvc.callCount, "Expected 1 purchase call (failed immediately)")
}

func TestMultipleOrdersIndependent(t *testing.T) {
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

	err1 := ExecuteOrder(order1,
		&MockQuotationService{},
		&MockRiskService{approved: true},
		NewMockPurchaseService(),
	)
	err2 := ExecuteOrder(order2,
		&MockQuotationService{},
		&MockRiskService{approved: true},
		NewMockPurchaseService(),
	)

	assert.NoError(t, err1, "First order should succeed")
	assert.NoError(t, err2, "Second order should succeed")
}

func TestCompensationDoesNotRunOnSuccess(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.NoError(t, err, "Expected successful order")

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

func TestOrderFailsOnFirstAssetPurchaseWithoutCompensation(t *testing.T) {
	quotationSvc := &MockQuotationService{}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()
	purchaseSvc.failOnCallNum = 0

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPurchasingAsset1), "Expected ErrPurchasingAsset1 when first asset purchase fails")

	assert.Equal(t, 1, purchaseSvc.callCount, "Expected 1 purchase call (failed immediately on Asset 1)")

	expectedActions := []domain.ActionType{
		domain.ActionBuy,
	}
	assert.Equal(t, expectedActions, purchaseSvc.executedActions, "Expected only BUY (no compensation since nothing was bought)")
}

func TestTTLCheckedAfterRiskAnalysis(t *testing.T) {
	quotationSvc := &MockQuotationService{
		response: &domain.QuotationResponse{
			Price1: 50.0,
			Price2: 5.0,
			TTLms:  -100,
		},
	}
	riskSvc := &MockRiskService{approved: true}
	purchaseSvc := NewMockPurchaseService()

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err := ExecuteOrder(order, quotationSvc, riskSvc, purchaseSvc)

	assert.True(t, errors.Is(err, ErrTTLExceeded), "Expected TTL to be checked")
	assert.Equal(t, 0, purchaseSvc.callCount, "Expected 0 purchase calls when TTL expired")
}
