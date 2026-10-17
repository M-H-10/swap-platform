package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/M-H-10/swap-platform/backend-go/internal/domain"
	"github.com/M-H-10/swap-platform/backend-go/internal/middleware"
	"github.com/M-H-10/swap-platform/backend-go/internal/service"
)

// WalletHandler exposes the wallet routes.
type WalletHandler struct {
	wallets *service.WalletService
}

// NewWalletHandler creates the handler.
func NewWalletHandler(wallets *service.WalletService) *WalletHandler {
	return &WalletHandler{wallets: wallets}
}

type walletResponse struct {
	Balance       int64     `json:"balance"`        // kopecks the user can use
	FrozenBalance int64     `json:"frozen_balance"` // kopecks locked in active swaps
	UpdatedAt     time.Time `json:"updated_at"`
}

type transactionResponse struct {
	ID           int64     `json:"id"`
	SwapID       *string   `json:"swap_id"`
	Type         string    `json:"type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	FrozenAfter  int64     `json:"frozen_after"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

func toWalletResponse(w domain.Wallet) walletResponse {
	return walletResponse{Balance: w.Balance, FrozenBalance: w.FrozenBalance, UpdatedAt: w.UpdatedAt}
}

// Get handles GET /api/wallet.
func (h *WalletHandler) Get(c fiber.Ctx) error {
	w, err := h.wallets.Get(c.Context(), middleware.UserID(c))
	if err != nil {
		return err
	}
	return c.JSON(toWalletResponse(w))
}

// Transactions handles GET /api/wallet/transactions?limit=&offset=
func (h *WalletHandler) Transactions(c fiber.Ctx) error {
	limit, err := queryInt(c, "limit")
	if err != nil {
		return err
	}
	offset, err := queryInt(c, "offset")
	if err != nil {
		return err
	}
	list, total, err := h.wallets.Transactions(c.Context(), middleware.UserID(c), limit, offset)
	if err != nil {
		return err
	}
	items := make([]transactionResponse, len(list))
	for i, t := range list {
		items[i] = transactionResponse{
			ID: t.ID, SwapID: t.SwapID, Type: string(t.Type), Amount: t.Amount,
			BalanceAfter: t.BalanceAfter, FrozenAfter: t.FrozenAfter,
			Description: t.Description, CreatedAt: t.CreatedAt,
		}
	}
	return c.JSON(fiber.Map{"items": items, "total": total})
}

// Deposit handles POST /api/wallet/deposit (development only).
func (h *WalletHandler) Deposit(c fiber.Ctx) error {
	var req struct {
		Amount int64 `json:"amount"` // kopecks
	}
	if err := readJSON(c, &req); err != nil {
		return err
	}
	w, err := h.wallets.Deposit(c.Context(), middleware.UserID(c), req.Amount)
	if err != nil {
		return err
	}
	return c.JSON(toWalletResponse(w))
}
