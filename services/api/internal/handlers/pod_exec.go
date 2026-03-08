package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/dotechhq/zenith/services/api/internal/adapters/k8sclient"
	"github.com/dotechhq/zenith/services/api/internal/entities"
	"github.com/dotechhq/zenith/services/api/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// PodExecHandler provides SSH-to-pod terminal access via WebSocket.
type PodExecHandler struct {
	k8sClient   k8sclient.Client
	appRepo     ports.AppRepository
	planRepo    ports.UserPlanRepository
	userRepo    ports.UserRepository
	sessionRepo ports.PodExecSessionRepository
	s3          ports.ObjectStorage
	s3Bucket    string
	namespace   string
}

// NewPodExecHandler creates a new PodExecHandler.
func NewPodExecHandler(
	k8sClient k8sclient.Client,
	appRepo ports.AppRepository,
	planRepo ports.UserPlanRepository,
	userRepo ports.UserRepository,
	sessionRepo ports.PodExecSessionRepository,
	s3 ports.ObjectStorage,
	s3Bucket string,
	namespace string,
) *PodExecHandler {
	return &PodExecHandler{
		k8sClient:   k8sClient,
		appRepo:     appRepo,
		planRepo:    planRepo,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		s3:          s3,
		s3Bucket:    s3Bucket,
		namespace:   namespace,
	}
}

// UpgradeCheck is Fiber middleware that pre-checks WebSocket upgrade eligibility.
func (h *PodExecHandler) UpgradeCheck(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

// HandleExec handles the WebSocket connection for pod exec.
// GET /api/v1/apps/:appId/pods/:podName/exec (WebSocket)
func (h *PodExecHandler) HandleExec() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		userID := c.Locals("user_id")
		if userID == nil {
			c.WriteMessage(websocket.TextMessage, []byte(`{"error":"unauthorized"}`))
			return
		}
		uid := userID.(string)

		appID := c.Params("appId")
		podName := c.Params("podName")

		// Validate ownership
		app, err := h.appRepo.GetApp(context.Background(), appID)
		if err != nil || app == nil || app.UserID != uid {
			c.WriteMessage(websocket.TextMessage, []byte(`{"error":"app not found"}`))
			return
		}

		// Check plan: Business+ only
		plan, err := h.planRepo.GetUserPlan(context.Background(), uid)
		if err != nil || (plan.Tier != entities.PlanBusiness && plan.Tier != entities.PlanEnterprise) {
			c.WriteMessage(websocket.TextMessage, []byte(`{"error":"pod exec requires Business plan or higher"}`))
			return
		}

		// Verify it's a real k8s client
		realClient, ok := h.k8sClient.(*k8sclient.RealClient)
		if !ok {
			// Memory mode: echo back input for demo
			h.handleMemoryExec(c, uid, app, podName)
			return
		}

		// Get user info for audit
		user, _ := h.userRepo.GetByID(context.Background(), uid)
		userEmail := ""
		if user != nil {
			userEmail = user.Email
		}

		// Create session record
		sessionID := uuid.New().String()
		session := &entities.PodExecSession{
			ID:        sessionID,
			UserID:    uid,
			UserEmail: userEmail,
			AppID:     appID,
			AppName:   app.Name,
			PodName:   podName,
			Container: app.Name,
			Command:   "/bin/sh",
			Status:    entities.PodSessionActive,
			IPAddress: c.RemoteAddr().String(),
			StartedAt: time.Now(),
		}
		h.sessionRepo.CreateSession(context.Background(), session)

		slog.Info("pod exec session started", "session_id", sessionID[:8], "user_id", uid[:8], "app", app.Name, "pod", podName)

		// Execute pod exec and bridge to WebSocket
		var recording bytes.Buffer
		h.execAndBridge(c, realClient, podName, &recording)

		// Session ended — save recording
		endTime := time.Now()
		duration := int(endTime.Sub(session.StartedAt).Seconds())
		recordingKey := ""

		if recording.Len() > 0 && h.s3 != nil {
			recordingKey = fmt.Sprintf("pod-sessions/%s/%s.cast", uid, sessionID)
			if err := h.s3.PutObject(context.Background(), h.s3Bucket, recordingKey, "application/x-asciicast", bytes.NewReader(recording.Bytes()), int64(recording.Len())); err != nil {
				slog.Warn("failed to save recording", "error", err)
				recordingKey = ""
			}
		}

		h.sessionRepo.EndSession(context.Background(), sessionID, recordingKey)
		slog.Info("pod exec session ended", "session_id", sessionID[:8], "duration_seconds", duration, "recording", recordingKey)
	})
}

// execAndBridge bridges a WebSocket connection to a k8s pod exec SPDY stream.
func (h *PodExecHandler) execAndBridge(ws *websocket.Conn, client *k8sclient.RealClient, podName string, recording *bytes.Buffer) {
	req := client.Clientset().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(h.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: []string{"/bin/sh"},
			Stdin:   true,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(client.RESTConfig(), "POST", req.URL())
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"error":"exec failed: %v"}`, err)))
		return
	}

	// Create bidirectional pipes
	stdinR, stdinW := io.Pipe()

	// Asciinema recording header
	header := asciinemaHeader(80, 24)
	recording.Write(header)

	streamDone := make(chan struct{})
	outputWriter := &wsOutputWriter{ws: ws, recording: recording, startTime: time.Now()}

	// Stream k8s exec output → WebSocket
	go func() {
		defer close(streamDone)
		err := exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
			Stdin:  stdinR,
			Stdout: outputWriter,
			Stderr: outputWriter,
			Tty:    true,
		})
		if err != nil {
			slog.Error("pod exec stream error", "error", err)
		}
	}()

	// Read WebSocket input → k8s exec stdin
	go func() {
		defer stdinW.Close()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			// Parse input message: {"type":"input","data":"..."}
			var input struct {
				Type string `json:"type"`
				Data string `json:"data"`
			}
			if json.Unmarshal(msg, &input) == nil && input.Type == "input" {
				stdinW.Write([]byte(input.Data))
			} else {
				// Raw data fallback
				stdinW.Write(msg)
			}
		}
	}()

	<-streamDone
}

// handleMemoryExec provides a fake terminal for demo/dev mode.
func (h *PodExecHandler) handleMemoryExec(ws *websocket.Conn, uid string, app *entities.App, podName string) {
	sessionID := uuid.New().String()
	session := &entities.PodExecSession{
		ID:        sessionID,
		UserID:    uid,
		AppID:     app.ID,
		AppName:   app.Name,
		PodName:   podName,
		Container: app.Name,
		Command:   "/bin/sh",
		Status:    entities.PodSessionActive,
		StartedAt: time.Now(),
	}
	h.sessionRepo.CreateSession(context.Background(), session)

	// Send welcome message
	ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Connected to %s (%s) [demo mode]\r\n$ ", podName, app.Name)))

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		var input struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if json.Unmarshal(msg, &input) == nil && input.Type == "input" {
			// Echo back with a demo response
			cmd := strings.TrimSpace(input.Data)
			if cmd == "\r" || cmd == "\n" {
				ws.WriteMessage(websocket.TextMessage, []byte("\r\n$ "))
			} else {
				ws.WriteMessage(websocket.TextMessage, []byte(cmd))
			}
		}
	}

	h.sessionRepo.EndSession(context.Background(), sessionID, "")
}

// ListSessions returns SSH session audit records for the current user.
// GET /api/v1/pod-sessions
func (h *PodExecHandler) ListSessions(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	sessions, total, err := h.sessionRepo.ListByUser(context.Background(), userID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"sessions": sessions, "total": total})
}

// AdminListSessions returns all pod exec sessions (admin only).
// GET /api/v1/admin/pod-sessions
func (h *PodExecHandler) AdminListSessions(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	sessions, total, err := h.sessionRepo.ListAll(context.Background(), limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"sessions": sessions, "total": total})
}

// GetRecordingURL returns a presigned download URL for a session recording.
// GET /api/v1/pod-sessions/:sessionId/recording
func (h *PodExecHandler) GetRecordingURL(c *fiber.Ctx) error {
	userID, _ := c.Locals("user_id").(string)
	sessionID := c.Params("sessionId")

	session, err := h.sessionRepo.GetSession(context.Background(), sessionID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "session not found")
	}
	if session.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "not your session")
	}
	if session.RecordingKey == "" {
		return fiber.NewError(fiber.StatusNotFound, "no recording available")
	}
	if h.s3 == nil {
		return fiber.NewError(fiber.StatusNotFound, "recordings not configured")
	}

	url, err := h.s3.GeneratePresignedDownloadURL(context.Background(), h.s3Bucket, session.RecordingKey, 15*time.Minute)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate download URL")
	}

	return c.JSON(fiber.Map{"url": url, "expires_in": 900})
}

// wsOutputWriter writes k8s exec output to both WebSocket and recording buffer.
type wsOutputWriter struct {
	ws        *websocket.Conn
	recording *bytes.Buffer
	startTime time.Time
}

func (w *wsOutputWriter) Write(p []byte) (int, error) {
	// Send to WebSocket
	if err := w.ws.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}

	// Record in asciinema format: [elapsed, "o", "data"]
	elapsed := time.Since(w.startTime).Seconds()
	data, _ := json.Marshal(string(p))
	fmt.Fprintf(w.recording, "[%.6f, \"o\", %s]\n", elapsed, data)

	return len(p), nil
}

// asciinemaHeader creates the asciinema v2 header.
func asciinemaHeader(width, height int) []byte {
	header := map[string]interface{}{
		"version":   2,
		"width":     width,
		"height":    height,
		"timestamp": time.Now().Unix(),
		"env": map[string]string{
			"SHELL": "/bin/sh",
			"TERM":  "xterm-256color",
		},
	}
	data, _ := json.Marshal(header)
	return append(data, '\n')
}
