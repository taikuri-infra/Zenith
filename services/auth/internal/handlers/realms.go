package handlers

import (
	"github.com/dotechhq/zenith/services/auth/internal/crypto"
	"github.com/dotechhq/zenith/services/auth/internal/models"
	"github.com/dotechhq/zenith/services/auth/internal/storage"
	"github.com/gofiber/fiber/v2"
)

type RealmHandler struct {
	store storage.Store
}

func NewRealmHandler(store storage.Store) *RealmHandler {
	return &RealmHandler{store: store}
}

type CreateRealmRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

func (h *RealmHandler) Create(c *fiber.Ctx) error {
	var req CreateRealmRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	realm := &models.Realm{
		ID:          req.Name,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Enabled:     true,
	}

	if err := h.store.CreateRealm(realm); err != nil {
		return fiber.NewError(fiber.StatusConflict, "realm already exists")
	}

	return c.Status(fiber.StatusCreated).JSON(realm)
}

func (h *RealmHandler) Get(c *fiber.Ctx) error {
	id := c.Params("realm")
	realm, err := h.store.GetRealm(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "realm not found")
	}
	return c.JSON(realm)
}

func (h *RealmHandler) List(c *fiber.Ctx) error {
	realms, err := h.store.ListRealms()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list realms")
	}
	if realms == nil {
		realms = []*models.Realm{}
	}
	return c.JSON(fiber.Map{"items": realms, "total": len(realms)})
}

func (h *RealmHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("realm")
	if err := h.store.DeleteRealm(id); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "realm not found")
	}
	return c.JSON(fiber.Map{"message": "realm deleted"})
}

// Client management within a realm

type CreateClientRequest struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	RedirectURIs []string `json:"redirect_uris"`
	Scopes       []string `json:"scopes"`
}

func (h *RealmHandler) CreateClient(c *fiber.Ctx) error {
	realmID := c.Params("realm")

	var req CreateClientRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if req.Type == "" {
		req.Type = "public"
	}

	clientID := crypto.GenerateID()
	var secret string
	if req.Type == "confidential" {
		var err error
		secret, err = crypto.GenerateSecret(32)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "failed to generate secret")
		}
	}

	client := &models.Client{
		ID:           clientID,
		RealmID:      realmID,
		Name:         req.Name,
		Type:         req.Type,
		Secret:       secret,
		RedirectURIs: req.RedirectURIs,
		Scopes:       req.Scopes,
		Enabled:      true,
	}

	if err := h.store.CreateClient(client); err != nil {
		return fiber.NewError(fiber.StatusConflict, "client already exists")
	}

	response := fiber.Map{
		"id":            client.ID,
		"name":          client.Name,
		"type":          client.Type,
		"redirect_uris": client.RedirectURIs,
		"scopes":        client.Scopes,
	}
	if secret != "" {
		response["secret"] = secret
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

func (h *RealmHandler) ListClients(c *fiber.Ctx) error {
	realmID := c.Params("realm")
	clients, err := h.store.ListClients(realmID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list clients")
	}
	if clients == nil {
		clients = []*models.Client{}
	}
	return c.JSON(fiber.Map{"items": clients, "total": len(clients)})
}

// User management within a realm

func (h *RealmHandler) ListUsers(c *fiber.Ctx) error {
	realmID := c.Params("realm")
	users, err := h.store.ListUsers(realmID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to list users")
	}
	if users == nil {
		users = []*models.User{}
	}
	return c.JSON(fiber.Map{"items": users, "total": len(users)})
}

func (h *RealmHandler) GetUser(c *fiber.Ctx) error {
	realmID := c.Params("realm")
	userID := c.Params("userId")

	user, err := h.store.GetUser(realmID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	return c.JSON(user)
}

func (h *RealmHandler) DeleteUser(c *fiber.Ctx) error {
	realmID := c.Params("realm")
	userID := c.Params("userId")

	if err := h.store.DeleteUser(realmID, userID); err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	return c.JSON(fiber.Map{"message": "user deleted"})
}
