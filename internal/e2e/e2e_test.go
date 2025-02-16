package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	handler "github.com/qosmioo/merch-store/internal/delivery/http"
	"github.com/qosmioo/merch-store/internal/entity"
	"github.com/qosmioo/merch-store/pkg/jwt"
	"go.uber.org/zap"
)

type EmployeeData struct {
	ID          int
	Name        string
	Password    string
	Coins       int
	Inventory   []entity.Inventory
	CoinHistory entity.CoinHistory
}

type FakeEmployeeUsecase struct {
	mu                  sync.Mutex
	nextID              int
	employeesByUsername map[string]*EmployeeData
	employeesByID       map[int]*EmployeeData
}

func NewFakeEmployeeUsecase() *FakeEmployeeUsecase {
	return &FakeEmployeeUsecase{
		nextID:              1,
		employeesByUsername: make(map[string]*EmployeeData),
		employeesByID:       make(map[int]*EmployeeData),
	}
}

func (f *FakeEmployeeUsecase) Authenticate(username, password string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	emp, ok := f.employeesByUsername[username]
	if ok {
		if emp.Password != password {
			return "", errors.New("invalid credentials")
		}
	} else {
		emp = &EmployeeData{
			ID:          f.nextID,
			Name:        username,
			Password:    password,
			Coins:       1000,
			Inventory:   []entity.Inventory{},
			CoinHistory: entity.CoinHistory{},
		}
		f.employeesByUsername[username] = emp
		f.employeesByID[f.nextID] = emp
		f.nextID++
	}
	return jwt.GenerateJWT(username)
}

func (f *FakeEmployeeUsecase) GetEmployeeIDByUsername(username string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	emp, ok := f.employeesByUsername[username]
	if !ok {
		return 0, errors.New("employee not found")
	}
	return emp.ID, nil
}

func (f *FakeEmployeeUsecase) GetEmployeeInfo(employeeID int) (entity.InfoResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	emp, ok := f.employeesByID[employeeID]
	if !ok {
		return entity.InfoResponse{}, errors.New("employee not found")
	}
	return entity.InfoResponse{
		Coins:       emp.Coins,
		Inventory:   emp.Inventory,
		CoinHistory: emp.CoinHistory,
	}, nil
}

func (f *FakeEmployeeUsecase) TransferCoins(fromEmployeeID, toEmployeeID, amount int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	fromEmp, ok := f.employeesByID[fromEmployeeID]
	if !ok {
		return errors.New("from employee not found")
	}
	toEmp, ok := f.employeesByID[toEmployeeID]
	if !ok {
		return errors.New("to employee not found")
	}
	if fromEmp.Coins < amount {
		return errors.New("insufficient coins")
	}
	fromEmp.Coins -= amount
	toEmp.Coins += amount

	fromEmp.CoinHistory.Sent = append(fromEmp.CoinHistory.Sent, entity.Transaction{
		UserID: toEmployeeID,
		Amount: amount,
	})
	toEmp.CoinHistory.Received = append(toEmp.CoinHistory.Received, entity.Transaction{
		UserID: fromEmployeeID,
		Amount: amount,
	})
	return nil
}

func (f *FakeEmployeeUsecase) BuyMerch(employeeID int, itemName string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	emp, ok := f.employeesByID[employeeID]
	if !ok {
		return errors.New("employee not found")
	}
	price := 50
	if emp.Coins < price {
		return errors.New("insufficient coins")
	}
	emp.Coins -= price
	updated := false
	for i, item := range emp.Inventory {
		if item.Type == itemName {
			emp.Inventory[i].Quantity++
			updated = true
			break
		}
	}
	if !updated {
		emp.Inventory = append(emp.Inventory, entity.Inventory{
			Type:     itemName,
			Quantity: 1,
		})
	}
	return nil
}

func setupServer(usecase *FakeEmployeeUsecase) *httptest.Server {
	router := mux.NewRouter()
	h := handler.NewHandler(usecase, zap.NewNop())
	h.RegisterRoutes(router)
	return httptest.NewServer(router)
}

func TestAuthNewUser(t *testing.T) {
	fake := NewFakeEmployeeUsecase()
	server := setupServer(fake)
	defer server.Close()

	reqBody, _ := json.Marshal(map[string]string{
		"username": "alice",
		"password": "password",
	})
	res, err := http.Post(server.URL+"/api/auth", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Не удалось выполнить запрос: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Ожидался статус 200, получен %d", res.StatusCode)
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("Ошибка декодирования ответа: %v", err)
	}
	if resp.Token == "" {
		t.Fatalf("Ожидался непустой токен, получен пустой")
	}
}

func TestAuthExistingIncorrect(t *testing.T) {
	fake := NewFakeEmployeeUsecase()
	_, err := fake.Authenticate("bob", "secret")
	if err != nil {
		t.Fatalf("Ошибка при предварительной аутентификации: %v", err)
	}
	server := setupServer(fake)
	defer server.Close()

	reqBody, _ := json.Marshal(map[string]string{
		"username": "bob",
		"password": "wrong",
	})
	res, err := http.Post(server.URL+"/api/auth", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Не удалось выполнить запрос: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Ожидался статус 401, получен %d", res.StatusCode)
	}
}

func TestGetInfoAuthorized(t *testing.T) {
	fake := NewFakeEmployeeUsecase()
	token, err := fake.Authenticate("charlie", "password")
	if err != nil {
		t.Fatalf("Ошибка аутентификации: %v", err)
	}
	server := setupServer(fake)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", server.URL+"/api/info", nil)
	if err != nil {
		t.Fatalf("Ошибка создания запроса: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Ожидался статус 200, получен %d", res.StatusCode)
	}

	var info entity.InfoResponse
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		t.Fatalf("Ошибка декодирования ответа: %v", err)
	}
	if info.Coins != 1000 {
		t.Fatalf("Ожидалось 1000 монет, получено %d", info.Coins)
	}
}

func TestSendCoin(t *testing.T) {
	fake := NewFakeEmployeeUsecase()
	tokenDave, err := fake.Authenticate("dave", "pass")
	if err != nil {
		t.Fatalf("Ошибка аутентификации dave: %v", err)
	}
	_, err = fake.Authenticate("eva", "pass")
	if err != nil {
		t.Fatalf("Ошибка аутентификации eva: %v", err)
	}
	server := setupServer(fake)
	defer server.Close()

	client := &http.Client{}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"toUser": "eva",
		"amount": 200,
	})
	req, err := http.NewRequest("POST", server.URL+"/api/sendCoin", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Ошибка создания запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenDave)

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Ожидался статус 200, получен %d", res.StatusCode)
	}

	daveID, _ := fake.GetEmployeeIDByUsername("dave")
	evaID, _ := fake.GetEmployeeIDByUsername("eva")
	daveInfo, _ := fake.GetEmployeeInfo(daveID)
	evaInfo, _ := fake.GetEmployeeInfo(evaID)

	if daveInfo.Coins != 800 {
		t.Fatalf("Ожидалось 800 монет у dave, получено %d", daveInfo.Coins)
	}
	if evaInfo.Coins != 1200 {
		t.Fatalf("Ожидалось 1200 монет у eva, получено %d", evaInfo.Coins)
	}
}

func TestSendCoinInsufficientFunds(t *testing.T) {
	fake := NewFakeEmployeeUsecase()
	tokenFrank, err := fake.Authenticate("frank", "pass")
	if err != nil {
		t.Fatalf("Ошибка аутентификации frank: %v", err)
	}
	_, err = fake.Authenticate("gina", "pass")
	if err != nil {
		t.Fatalf("Ошибка аутентификации gina: %v", err)
	}
	frankID, _ := fake.GetEmployeeIDByUsername("frank")
	fake.mu.Lock()
	fake.employeesByID[frankID].Coins = 100
	fake.mu.Unlock()

	server := setupServer(fake)
	defer server.Close()

	client := &http.Client{}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"toUser": "gina",
		"amount": 200,
	})
	req, err := http.NewRequest("POST", server.URL+"/api/sendCoin", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Ошибка создания запроса: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tokenFrank)

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("Ожидался статус 500, получен %d", res.StatusCode)
	}
}

func TestBuyMerch(t *testing.T) {
	fake := NewFakeEmployeeUsecase()
	tokenHarry, err := fake.Authenticate("harry", "pass")
	if err != nil {
		t.Fatalf("Ошибка аутентификации harry: %v", err)
	}
	server := setupServer(fake)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", server.URL+"/api/buy/item1", nil)
	if err != nil {
		t.Fatalf("Ошибка создания запроса: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenHarry)

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("Ожидался статус 200, получен %d", res.StatusCode)
	}

	harryID, _ := fake.GetEmployeeIDByUsername("harry")
	info, _ := fake.GetEmployeeInfo(harryID)
	if info.Coins != 950 {
		t.Fatalf("Ожидалось 950 монет у harry, получено %d", info.Coins)
	}
	if len(info.Inventory) == 0 || info.Inventory[0].Type != "item1" || info.Inventory[0].Quantity != 1 {
		t.Fatalf("Ожидался товар item1 в инвентаре с количеством 1")
	}
}

func TestAccessWithoutToken(t *testing.T) {
	fake := NewFakeEmployeeUsecase()
	server := setupServer(fake)
	defer server.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", server.URL+"/api/info", nil)
	if err != nil {
		t.Fatalf("Ошибка создания запроса: %v", err)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Ожидался статус 401, получен %d", res.StatusCode)
	}
}
