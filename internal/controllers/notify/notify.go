package notify

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	ns "github.com/eric2788/bilirec/internal/services/notify"
	"github.com/gofiber/fiber/v3"
)

type Controller struct {
	notifySvc *ns.Service
}

func NewController(app *fiber.App, notifySvc *ns.Service) *Controller {
	c := &Controller{notifySvc: notifySvc}
	group := app.Group("/notify")
	group.Get("/stream", c.stream)
	return c
}

func (c *Controller) stream(ctx fiber.Ctx) error {
	ctx.Set(fiber.HeaderContentType, "text/event-stream")
	ctx.Set(fiber.HeaderCacheControl, "no-cache")
	ctx.Set(fiber.HeaderConnection, "keep-alive")

	// force nginx to not buffer the response
	ctx.Set("X-Accel-Buffering", "no")

	_, ch, unsubscribe := c.notifySvc.Subscribe(32)

	requestCtx := ctx.RequestCtx()
	requestCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer unsubscribe()
		writeEvent := func(event string, payload []byte) bool {
			if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
				return false
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return false
			}
			return w.Flush() == nil
		}

		if !writeEvent("ping", []byte(`{"ok":true}`)) {
			return
		}

		heartbeat := time.NewTicker(25 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case <-requestCtx.Done():
				return
			case <-heartbeat.C:
				if !writeEvent("ping", []byte(`{"ok":true}`)) {
					return
				}
			case evt, ok := <-ch:
				if !ok {
					return
				}
				payload, err := json.Marshal(evt)
				if err != nil {
					continue
				}
				if !writeEvent("notification", payload) {
					return
				}
			}
		}
	})

	return nil
}
