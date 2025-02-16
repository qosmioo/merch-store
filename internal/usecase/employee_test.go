package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/qosmioo/merch-store/internal/entity"
	"github.com/qosmioo/merch-store/internal/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGetEmployeeInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockEmployeeRepository(ctrl)
	logger := zap.NewNop()
	usecase := NewEmployeeUsecase(mockRepo, logger)

	employeeID := 1
	expectedEmployee := entity.Employee{ID: employeeID, Coins: 100}
	expectedInventory := []entity.Inventory{{Type: "item1", Quantity: 1}}
	expectedHistory := entity.CoinHistory{}

	mockRepo.EXPECT().GetEmployeeByID(employeeID).Return(expectedEmployee, nil)
	mockRepo.EXPECT().GetEmployeeInventory(employeeID).Return(expectedInventory, nil)
	mockRepo.EXPECT().GetEmployeeCoinHistory(employeeID).Return(expectedHistory, nil)

	info, err := usecase.GetEmployeeInfo(employeeID)
	assert.NoError(t, err)
	assert.Equal(t, expectedEmployee.Coins, info.Coins)
	assert.Equal(t, expectedInventory, info.Inventory)
	assert.Equal(t, expectedHistory, info.CoinHistory)
}

func TestTransferCoins_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockEmployeeRepository(ctrl)
	mockTx := repository.NewMockTransaction(ctrl)
	logger := zap.NewNop()
	usecase := NewEmployeeUsecase(mockRepo, logger)

	fromEmployeeID := 1
	toEmployeeID := 2
	amount := 50
	fromEmployee := entity.Employee{ID: fromEmployeeID, Coins: 100}
	toEmployee := entity.Employee{ID: toEmployeeID, Coins: 50}

	mockRepo.EXPECT().BeginTransaction().Return(mockTx, nil)
	mockRepo.EXPECT().GetEmployeeByIDTx(mockTx, fromEmployeeID).Return(fromEmployee, nil)
	mockRepo.EXPECT().GetEmployeeByIDTx(mockTx, toEmployeeID).Return(toEmployee, nil)
	mockRepo.EXPECT().UpdateEmployeeCoinsTx(mockTx, fromEmployeeID, fromEmployee.Coins-amount).Return(nil)
	mockRepo.EXPECT().UpdateEmployeeCoinsTx(mockTx, toEmployeeID, toEmployee.Coins+amount).Return(nil)
	mockRepo.EXPECT().RecordTransactionTx(mockTx, fromEmployeeID, toEmployeeID, amount).Return(nil)
	mockTx.EXPECT().Commit(context.Background()).Return(nil)

	err := usecase.TransferCoins(fromEmployeeID, toEmployeeID, amount)
	assert.NoError(t, err)
}

func TestTransferCoins_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockEmployeeRepository(ctrl)
	mockTx := repository.NewMockTransaction(ctrl)
	logger := zap.NewNop()
	usecase := NewEmployeeUsecase(mockRepo, logger)

	fromEmployeeID := 1
	toEmployeeID := 2
	amount := 150
	fromEmployee := entity.Employee{ID: fromEmployeeID, Coins: 100}

	mockRepo.EXPECT().BeginTransaction().Return(mockTx, nil)
	mockRepo.EXPECT().GetEmployeeByIDTx(mockTx, fromEmployeeID).Return(fromEmployee, nil)
	mockTx.EXPECT().Rollback(context.Background()).Return(nil)

	err := usecase.TransferCoins(fromEmployeeID, toEmployeeID, amount)
	assert.Error(t, err)
	assert.Equal(t, "insufficient coins", err.Error())
}

func TestTransferCoins_NonExistentRecipient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockEmployeeRepository(ctrl)
	mockTx := repository.NewMockTransaction(ctrl)
	logger := zap.NewNop()
	usecase := NewEmployeeUsecase(mockRepo, logger)

	fromEmployeeID := 1
	toEmployeeID := 999
	amount := 50
	fromEmployee := entity.Employee{ID: fromEmployeeID, Coins: 100}

	mockRepo.EXPECT().BeginTransaction().Return(mockTx, nil)
	mockRepo.EXPECT().GetEmployeeByIDTx(mockTx, fromEmployeeID).Return(fromEmployee, nil)
	mockRepo.EXPECT().GetEmployeeByIDTx(mockTx, toEmployeeID).Return(entity.Employee{}, errors.New("employee not found"))
	mockTx.EXPECT().Rollback(context.Background()).Return(nil)

	err := usecase.TransferCoins(fromEmployeeID, toEmployeeID, amount)
	assert.Error(t, err)
	assert.Equal(t, "employee not found", err.Error())
}

func TestBuyMerch_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockEmployeeRepository(ctrl)
	logger := zap.NewNop()
	usecase := NewEmployeeUsecase(mockRepo, logger)

	employeeID := 1
	itemName := "item1"
	employee := entity.Employee{ID: employeeID, Coins: 30}

	mockRepo.EXPECT().GetMerchPrice(itemName).Return(50, nil)
	mockRepo.EXPECT().GetEmployeeByID(employeeID).Return(employee, nil)

	err := usecase.BuyMerch(employeeID, itemName)
	assert.Error(t, err)
	assert.Equal(t, "insufficient coins", err.Error())
}

func TestAuthenticate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockEmployeeRepository(ctrl)
	logger := zap.NewNop()
	usecase := NewEmployeeUsecase(mockRepo, logger)

	username := "testuser"
	password := "password"
	employee := entity.Employee{Name: username, Password: password, Coins: 1000}

	mockRepo.EXPECT().GetEmployeeByUsername(username).Return(employee, nil)

	token, err := usecase.Authenticate(username, password)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestAuthenticate_InvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repository.NewMockEmployeeRepository(ctrl)
	logger := zap.NewNop()
	usecase := NewEmployeeUsecase(mockRepo, logger)

	username := "testuser"
	password := "wrongpassword"
	employee := entity.Employee{Name: username, Password: "correctpassword", Coins: 1000}

	mockRepo.EXPECT().GetEmployeeByUsername(username).Return(employee, nil)

	token, err := usecase.Authenticate(username, password)
	assert.Error(t, err)
	assert.Equal(t, "invalid credentials", err.Error())
	assert.Empty(t, token)
}
