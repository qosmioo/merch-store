package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/qosmioo/merch-store/internal/usecase"
	"github.com/qosmioo/merch-store/pkg/jwt"
	"go.uber.org/zap"
)

type Handler struct {
	employeeUsecase usecase.EmployeeUsecase
	logger          *zap.Logger
}

func NewHandler(employeeUsecase usecase.EmployeeUsecase, logger *zap.Logger) *Handler {
	return &Handler{employeeUsecase: employeeUsecase, logger: logger}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/info", jwt.AuthMiddleware(h.GetInfo)).Methods("GET")
	router.HandleFunc("/api/sendCoin", jwt.AuthMiddleware(h.SendCoin)).Methods("POST")
	router.HandleFunc("/api/buy/{item}", jwt.AuthMiddleware(h.BuyItem)).Methods("GET")
	router.HandleFunc("/api/auth", h.Authenticate).Methods("POST")
}

func (h *Handler) GetInfo(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("GetInfo called")
	claims, ok := r.Context().Value("claims").(*jwt.Claims)
	if !ok {
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}

	employeeID, err := h.employeeUsecase.GetEmployeeIDByUsername(claims.Username)
	if err != nil {
		h.logger.Error("Error getting employee ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	info, err := h.employeeUsecase.GetEmployeeInfo(employeeID)
	if err != nil {
		h.logger.Error("Error getting employee info", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
	h.logger.Info("Successfully retrieved employee info", zap.Any("info", info))
}

func (h *Handler) SendCoin(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("SendCoin called")
	var request struct {
		ToUser string `json:"toUser"`
		Amount int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	claims, ok := r.Context().Value("claims").(*jwt.Claims)
	if !ok {
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}

	fromEmployeeID, err := h.employeeUsecase.GetEmployeeIDByUsername(claims.Username)
	if err != nil {
		h.logger.Error("Error getting sender employee ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	toEmployeeID, err := h.employeeUsecase.GetEmployeeIDByUsername(request.ToUser)
	if err != nil {
		h.logger.Error("Recipient not found", zap.Error(err))
		http.Error(w, "Получатель не найден", http.StatusBadRequest)
		return
	}

	err = h.employeeUsecase.TransferCoins(fromEmployeeID, toEmployeeID, request.Amount)
	if err != nil {
		h.logger.Error("Error transferring coins", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	h.logger.Info("Successfully transferred coins", zap.String("from", claims.Username), zap.String("to", request.ToUser), zap.Int("amount", request.Amount))
}

func (h *Handler) BuyItem(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("BuyItem called")
	vars := mux.Vars(r)
	itemName := vars["item"]

	claims, ok := r.Context().Value("claims").(*jwt.Claims)
	if !ok {
		http.Error(w, "Неавторизован", http.StatusUnauthorized)
		return
	}

	employeeID, err := h.employeeUsecase.GetEmployeeIDByUsername(claims.Username)
	if err != nil {
		h.logger.Error("Error getting employee ID", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = h.employeeUsecase.BuyMerch(employeeID, itemName)
	if err != nil {
		h.logger.Error("Error buying item", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	h.logger.Info("Successfully purchased item", zap.String("item", itemName), zap.String("by", claims.Username))
}

func (h *Handler) Authenticate(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Authenticate called")
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	token, err := h.employeeUsecase.Authenticate(request.Username, request.Password)
	if err != nil {
		h.logger.Error("Authentication failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	response := struct {
		Token string `json:"token"`
	}{Token: token}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	h.logger.Info("Successfully authenticated user", zap.String("username", request.Username))
}
