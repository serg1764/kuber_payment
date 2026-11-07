package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"payment-service/internal/repository"
	"payment-service/internal/service"
)

type PaymentHandler struct {
	service *service.PaymentService
}

type CreatePaymentResponse struct {
	repository.Payment
	NotificationStatus string `json:"notification_status"`
	Information        string `json:"information"`
}

func NewPaymentHandler(service *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) GetPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := h.service.GetPayments(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch payments")
		return
	}

	writeJSON(w, http.StatusOK, payments)
}

func (h *PaymentHandler) GetPaymentByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid payment id")
		return
	}

	payment, err := h.service.GetPaymentByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrPaymentNotFound) {
			writeError(w, http.StatusNotFound, "payment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch payment")
		return
	}

	writeJSON(w, http.StatusOK, payment)
}

func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req repository.CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	req.Provider = strings.TrimSpace(req.Provider)
	req.Status = strings.TrimSpace(req.Status)
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency == "" {
		req.Currency = "EUR"
	}

	if req.OrderID <= 0 || req.Amount <= 0 || req.Provider == "" || req.Status == "" || len(req.Currency) != 3 {
		writeError(w, http.StatusBadRequest, "invalid payment payload")
		return
	}

	payment, notificationStatus, err := h.service.CreatePayment(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create payment")
		return
	}

	writeJSON(w, http.StatusCreated, CreatePaymentResponse{
		Payment:            payment,
		NotificationStatus: string(notificationStatus),
		Information:        "Что-то написали!!! 888 !!! - 888 - !!! - v4",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
