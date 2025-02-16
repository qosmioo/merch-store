package usecase

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/qosmioo/merch-store/internal/entity"
	"github.com/qosmioo/merch-store/internal/repository"
	"github.com/qosmioo/merch-store/pkg/jwt"
	"go.uber.org/zap"
)

type EmployeeUsecase interface {
	GetEmployeeInfo(employeeID int) (entity.InfoResponse, error)
	TransferCoins(fromEmployeeID, toEmployeeID, amount int) error
	BuyMerch(employeeID int, itemName string) error
	Authenticate(username, password string) (string, error)
	GetEmployeeIDByUsername(username string) (int, error)
}

type employeeUsecase struct {
	employeeRepo repository.EmployeeRepository
	logger       *zap.Logger
}

func NewEmployeeUsecase(employeeRepo repository.EmployeeRepository, logger *zap.Logger) EmployeeUsecase {
	return &employeeUsecase{employeeRepo: employeeRepo, logger: logger}
}

func (u *employeeUsecase) GetEmployeeInfo(employeeID int) (entity.InfoResponse, error) {
	u.logger.Info("GetEmployeeInfo called", zap.Int("employeeID", employeeID))
	employee, err := u.employeeRepo.GetEmployeeByID(employeeID)
	if err != nil {
		u.logger.Error("Error getting employee by ID", zap.Error(err))
		return entity.InfoResponse{}, err
	}

	inventory, err := u.employeeRepo.GetEmployeeInventory(employeeID)
	if err != nil {
		u.logger.Error("Error getting employee inventory", zap.Error(err))
		return entity.InfoResponse{}, err
	}

	coinHistory, err := u.employeeRepo.GetEmployeeCoinHistory(employeeID)
	if err != nil {
		u.logger.Error("Error getting employee coin history", zap.Error(err))
		return entity.InfoResponse{}, err
	}

	u.logger.Info("Successfully retrieved employee info", zap.Int("employeeID", employeeID))
	return entity.InfoResponse{
		Coins:       employee.Coins,
		Inventory:   inventory,
		CoinHistory: coinHistory,
	}, nil
}

func (u *employeeUsecase) TransferCoins(fromEmployeeID, toEmployeeID, amount int) (err error) {
	u.logger.Info("TransferCoins called", zap.Int("fromEmployeeID", fromEmployeeID), zap.Int("toEmployeeID", toEmployeeID), zap.Int("amount", amount))

	var tx pgx.Tx
	if tx, err = u.employeeRepo.BeginTransaction(); err != nil {
		u.logger.Error("Error starting transaction", zap.Error(err))
		return err
	}
	defer func(err *error) {
		if *err != nil {
			tx.Rollback(context.Background())
		} else {
			tx.Commit(context.Background())
		}
	}(&err)

	fromEmployee, err := u.employeeRepo.GetEmployeeByIDTx(tx, fromEmployeeID)
	if err != nil {
		u.logger.Error("Error getting from employee by ID", zap.Error(err))
		return err
	}

	if fromEmployee.Coins < amount {
		u.logger.Warn("Insufficient coins", zap.Int("fromEmployeeID", fromEmployeeID), zap.Int("amount", amount))
		err = errors.New("insufficient coins")
		return err
	}

	toEmployee, err := u.employeeRepo.GetEmployeeByIDTx(tx, toEmployeeID)
	if err != nil {
		u.logger.Error("Error getting to employee by ID", zap.Error(err))
		return err
	}

	// Обновление количества монет
	err = u.employeeRepo.UpdateEmployeeCoinsTx(tx, fromEmployeeID, fromEmployee.Coins-amount)
	if err != nil {
		u.logger.Error("Error updating from employee coins", zap.Error(err))
		return err
	}

	err = u.employeeRepo.UpdateEmployeeCoinsTx(tx, toEmployeeID, toEmployee.Coins+amount)
	if err != nil {
		u.logger.Error("Error updating to employee coins", zap.Error(err))
		return err
	}

	err = u.employeeRepo.RecordTransactionTx(tx, fromEmployeeID, toEmployeeID, amount)
	if err != nil {
		u.logger.Error("Error recording transaction", zap.Error(err))
		return err
	}

	u.logger.Info("Successfully transferred coins", zap.Int("fromEmployeeID", fromEmployeeID), zap.Int("toEmployeeID", toEmployeeID), zap.Int("amount", amount))
	return nil
}

func (u *employeeUsecase) BuyMerch(employeeID int, itemName string) error {
	u.logger.Info("BuyMerch called", zap.Int("employeeID", employeeID), zap.String("itemName", itemName))
	price, err := u.employeeRepo.GetMerchPrice(itemName)
	if err != nil {
		u.logger.Error("Error getting merch price", zap.Error(err))
		return err
	}

	employee, err := u.employeeRepo.GetEmployeeByID(employeeID)
	if err != nil {
		u.logger.Error("Error getting employee by ID", zap.Error(err))
		return err
	}

	if employee.Coins < price {
		u.logger.Warn("Insufficient coins for purchase", zap.Int("employeeID", employeeID), zap.Int("price", price))
		return errors.New("insufficient coins")
	}

	err = u.employeeRepo.UpdateEmployeeCoins(employeeID, employee.Coins-price)
	if err != nil {
		u.logger.Error("Error updating employee coins after purchase", zap.Error(err))
		return err
	}

	err = u.employeeRepo.AddToInventory(employeeID, itemName)
	if err != nil {
		u.logger.Error("Error adding item to inventory", zap.Error(err))
		return err
	}

	u.logger.Info("Successfully purchased merch", zap.Int("employeeID", employeeID), zap.String("itemName", itemName))
	return nil
}

func (u *employeeUsecase) Authenticate(username, password string) (string, error) {
	u.logger.Info("Authenticate called", zap.String("username", username))
	employee, err := u.employeeRepo.GetEmployeeByUsername(username)
	if err != nil {
		u.logger.Warn("User not found, creating new user", zap.String("username", username))
		newEmployee := entity.Employee{
			Name:     username,
			Password: password,
			Coins:    1000,
		}
		err = u.employeeRepo.CreateEmployee(newEmployee)
		if err != nil {
			u.logger.Error("Failed to create new user", zap.Error(err))
			return "", errors.New("failed to create new user")
		}
		token, err := jwt.GenerateJWT(username)
		if err != nil {
			u.logger.Error("Error generating JWT", zap.Error(err))
			return "", err
		}
		u.logger.Info("Successfully authenticated new user", zap.String("username", username))
		return token, nil
	}

	if employee.Password != password {
		u.logger.Warn("Invalid credentials", zap.String("username", username))
		return "", errors.New("invalid credentials")
	}

	token, err := jwt.GenerateJWT(username)
	if err != nil {
		u.logger.Error("Error generating JWT", zap.Error(err))
		return "", err
	}

	u.logger.Info("Successfully authenticated user", zap.String("username", username))
	return token, nil
}

func (u *employeeUsecase) GetEmployeeIDByUsername(username string) (int, error) {
	u.logger.Info("GetEmployeeIDByUsername called", zap.String("username", username))
	employeeID, err := u.employeeRepo.GetEmployeeIDByUsername(username)
	if err != nil {
		u.logger.Error("Error getting employee ID by username", zap.Error(err))
		return 0, err
	}
	u.logger.Info("Successfully retrieved employee ID", zap.String("username", username), zap.Int("employeeID", employeeID))
	return employeeID, nil
}
