package repository

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/qosmioo/merch-store/internal/entity"
	"go.uber.org/zap"
)

const (
	queryGetEmployeeByID         = "SELECT id, name, coins FROM employees WHERE id = $1"
	queryUpdateEmployeeCoins     = "UPDATE employees SET coins = $1 WHERE id = $2"
	queryGetEmployeeIDByUsername = "SELECT id FROM employees WHERE name = $1"
	queryGetEmployeeByUsername   = "SELECT id, name, coins, password FROM employees WHERE name = $1"
	queryGetEmployeeInventory    = "SELECT type, quantity FROM inventory WHERE employee_id = $1"
	queryGetEmployeeCoinHistoryR = "SELECT from_user_id, amount FROM transactions WHERE to_user_id = $1"
	queryGetEmployeeCoinHistoryS = "SELECT to_user_id, amount FROM transactions WHERE from_user_id = $1"
	queryRecordTransaction       = "INSERT INTO transactions (from_user_id, to_user_id, amount) VALUES ($1, $2, $3)"
	queryAddToInventory          = "INSERT INTO inventory (employee_id, type, quantity) VALUES ($1, $2, 1) ON CONFLICT (employee_id, type) DO UPDATE SET quantity = inventory.quantity + 1"
	queryCreateEmployee          = "INSERT INTO employees (name, password, coins) VALUES ($1, $2, $3)"
)

type EmployeeRepository interface {
	GetEmployeeByID(employeeID int) (entity.Employee, error)
	UpdateEmployeeCoins(employeeID, newAmount int) error
	GetEmployeeIDByUsername(username string) (int, error)
	GetEmployeeByUsername(username string) (entity.Employee, error)
	GetEmployeeInventory(employeeID int) ([]entity.Inventory, error)
	GetEmployeeCoinHistory(employeeID int) (entity.CoinHistory, error)
	RecordTransaction(fromEmployeeID, toEmployeeID, amount int) error
	AddToInventory(employeeID int, itemName string) error
	GetMerchPrice(itemName string) (int, error)
	CreateEmployee(employee entity.Employee) error
	BeginTransaction() (pgx.Tx, error)
	GetEmployeeByIDTx(tx pgx.Tx, employeeID int) (entity.Employee, error)
	UpdateEmployeeCoinsTx(tx pgx.Tx, employeeID, newAmount int) error
	RecordTransactionTx(tx pgx.Tx, fromEmployeeID, toEmployeeID, amount int) error
}

type employeeRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewEmployeeRepository(db *pgxpool.Pool, logger *zap.Logger) EmployeeRepository {
	return &employeeRepository{db: db, logger: logger}
}

func (r *employeeRepository) GetEmployeeByID(employeeID int) (entity.Employee, error) {
	r.logger.Info("Fetching employee by ID", zap.Int("employeeID", employeeID))
	var employee entity.Employee
	err := r.db.QueryRow(context.Background(), queryGetEmployeeByID, employeeID).Scan(&employee.ID, &employee.Name, &employee.Coins)
	if err != nil {
		r.logger.Error("Error fetching employee", zap.Error(err))
		return entity.Employee{}, err
	}
	r.logger.Info("Successfully fetched employee", zap.Any("employee", employee))
	return employee, nil
}

func (r *employeeRepository) UpdateEmployeeCoins(employeeID, newAmount int) error {
	r.logger.Info("Updating employee coins", zap.Int("employeeID", employeeID), zap.Int("newAmount", newAmount))
	_, err := r.db.Exec(context.Background(), queryUpdateEmployeeCoins, newAmount, employeeID)
	if err != nil {
		r.logger.Error("Error updating employee coins", zap.Error(err))
	}
	return err
}

func (r *employeeRepository) GetEmployeeIDByUsername(username string) (int, error) {
	r.logger.Info("Fetching employee ID by username", zap.String("username", username))
	var employeeID int
	err := r.db.QueryRow(context.Background(), queryGetEmployeeIDByUsername, username).Scan(&employeeID)
	if err != nil {
		r.logger.Error("Error fetching employee ID", zap.Error(err))
		return 0, err
	}
	r.logger.Info("Successfully fetched employee ID", zap.String("username", username), zap.Int("employeeID", employeeID))
	return employeeID, nil
}

func (r *employeeRepository) GetEmployeeByUsername(username string) (entity.Employee, error) {
	var employee entity.Employee
	err := r.db.QueryRow(context.Background(), queryGetEmployeeByUsername, username).Scan(&employee.ID, &employee.Name, &employee.Coins, &employee.Password)
	if err != nil {
		return entity.Employee{}, err
	}
	return employee, nil
}

func (r *employeeRepository) GetEmployeeInventory(employeeID int) ([]entity.Inventory, error) {
	rows, err := r.db.Query(context.Background(), queryGetEmployeeInventory, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inventory []entity.Inventory
	for rows.Next() {
		var item entity.Inventory
		if err := rows.Scan(&item.Type, &item.Quantity); err != nil {
			return nil, err
		}
		inventory = append(inventory, item)
	}
	return inventory, nil
}

func (r *employeeRepository) GetEmployeeCoinHistory(employeeID int) (entity.CoinHistory, error) {
	var history entity.CoinHistory

	receivedRows, err := r.db.Query(context.Background(), queryGetEmployeeCoinHistoryR, employeeID)
	if err != nil {
		return history, err
	}
	defer receivedRows.Close()

	for receivedRows.Next() {
		var transaction entity.Transaction
		if err := receivedRows.Scan(&transaction.UserID, &transaction.Amount); err != nil {
			return history, err
		}
		history.Received = append(history.Received, transaction)
	}

	sentRows, err := r.db.Query(context.Background(), queryGetEmployeeCoinHistoryS, employeeID)
	if err != nil {
		return history, err
	}
	defer sentRows.Close()

	for sentRows.Next() {
		var transaction entity.Transaction
		if err := sentRows.Scan(&transaction.UserID, &transaction.Amount); err != nil {
			return history, err
		}
		history.Sent = append(history.Sent, transaction)
	}

	return history, nil
}

func (r *employeeRepository) RecordTransaction(fromEmployeeID, toEmployeeID, amount int) error {
	_, err := r.db.Exec(context.Background(), queryRecordTransaction, fromEmployeeID, toEmployeeID, amount)
	return err
}

func (r *employeeRepository) AddToInventory(employeeID int, itemName string) error {
	_, err := r.db.Exec(context.Background(), queryAddToInventory, employeeID, itemName)
	return err
}

func (r *employeeRepository) GetMerchPrice(itemName string) (int, error) {
	r.logger.Info("Fetching merch price", zap.String("itemName", itemName))
	query := "SELECT price FROM merch WHERE name = $1"
	var price int
	err := r.db.QueryRow(context.Background(), query, itemName).Scan(&price)
	if err != nil {
		r.logger.Error("Error fetching merch price", zap.Error(err))
		return 0, err
	}
	r.logger.Info("Successfully fetched merch price", zap.Int("price", price))
	return price, nil
}

func (r *employeeRepository) CreateEmployee(employee entity.Employee) error {
	_, err := r.db.Exec(context.Background(), queryCreateEmployee, employee.Name, employee.Password, employee.Coins)
	return err
}

func (r *employeeRepository) BeginTransaction() (pgx.Tx, error) {
	return r.db.Begin(context.Background())
}

func (r *employeeRepository) GetEmployeeByIDTx(tx pgx.Tx, employeeID int) (entity.Employee, error) {
	r.logger.Info("Fetching employee by ID in transaction", zap.Int("employeeID", employeeID))
	var employee entity.Employee
	err := tx.QueryRow(context.Background(), queryGetEmployeeByID, employeeID).Scan(&employee.ID, &employee.Name, &employee.Coins)
	if err != nil {
		r.logger.Error("Error fetching employee in transaction", zap.Error(err))
		return entity.Employee{}, err
	}
	return employee, nil
}

func (r *employeeRepository) UpdateEmployeeCoinsTx(tx pgx.Tx, employeeID, newAmount int) error {
	r.logger.Info("Updating employee coins in transaction", zap.Int("employeeID", employeeID), zap.Int("newAmount", newAmount))
	_, err := tx.Exec(context.Background(), queryUpdateEmployeeCoins, newAmount, employeeID)
	if err != nil {
		r.logger.Error("Error updating employee coins in transaction", zap.Error(err))
	}
	return err
}

func (r *employeeRepository) RecordTransactionTx(tx pgx.Tx, fromEmployeeID, toEmployeeID, amount int) error {
	_, err := tx.Exec(context.Background(), queryRecordTransaction, fromEmployeeID, toEmployeeID, amount)
	return err
}
