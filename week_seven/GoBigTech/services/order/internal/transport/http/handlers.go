package http

import (
	"encoding/json"
	"net/http"

	orderapi "github.com/bulbahal/GoBigTech/services/order/api"
	"github.com/bulbahal/GoBigTech/services/order/internal/service"
)

type Handler struct {
	Service service.OrderService
}

func (h *Handler) PostOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body orderapi.CreateOrder
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if body.UserId == "" || len(body.Items) == 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	items := make([]service.OrderItem, len(body.Items))
	for i, it := range body.Items {
		items[i] = service.OrderItem{
			ProductID: it.ProductId,
			Quantity:  int(it.Quantity),
		}
	}

	order, err := h.Service.CreateOrder(ctx, body.UserId, items)
	if err != nil {
		http.Error(w, "order processing error: "+err.Error(), http.StatusBadGateway)
		return
	}

	respItems := make([]orderapi.OrderItem, len(order.Items))
	for i, it := range order.Items {
		respItems[i] = orderapi.OrderItem{
			ProductId: it.ProductID,
			Quantity:  int32(it.Quantity),
		}
	}

	resp := orderapi.Order{
		Id:     order.ID,
		UserId: order.UserID,
		Status: orderapi.OrderStatus(order.Status),
		Items:  respItems,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetOrdersId(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	order, err := h.Service.GetOrder(ctx, id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	respItems := make([]orderapi.OrderItem, len(order.Items))
	for i, it := range order.Items {
		respItems[i] = orderapi.OrderItem{
			ProductId: it.ProductID,
			Quantity:  int32(it.Quantity),
		}
	}

	resp := orderapi.Order{
		Id:     order.ID,
		UserId: order.UserID,
		Status: orderapi.OrderStatus(order.Status),
		Items:  respItems,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
