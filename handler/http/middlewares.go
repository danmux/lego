package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/stairlin/lego/stats"
)

// Middleware is a function called on the HTTP stack before an action
type Middleware func(CallFunc) CallFunc

func buildMiddlewareChain(l []Middleware, a Action) CallFunc {
	if len(l) == 0 {
		return a.Call
	}

	c := a.Call
	for i := len(l) - 1; i >= 0; i-- {
		c = l[i](c)
	}

	return c
}

// mwDraining blocks request when the handler is draining
func mwDraining(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		c.Ctx.Trace("http.mw.draining.call")
		if c.isDraining() {
			return c.Head(http.StatusServiceUnavailable)
		}
		return next(c)
	}
}

// mwDraining blocks request when the handler is draining
func mwLogging(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		c.Ctx.Trace("http.mw.logging.call")

		c.Ctx.Tracef("h.http.req.start", "%s %T", c.Req.Method, c.Req.URL)
		c.Ctx.Trace("h.http.req.ua", c.Req.Header.Get("User-Agent"))

		r := next(c)

		c.Ctx.Tracef("h.http.req.end", "status=<%v> duration=<%v>", r.Status(), time.Since(c.StartAt))

		return r
	}
}

// mwStats sends request/response stats
func mwStats(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		c.Ctx.Trace("http.mw.stats.call")

		r := next(c)

		// Stats
		metric := stats.Metric{
			Key: "http",
			Values: map[string]interface{}{
				"duration": time.Since(c.StartAt).Nanoseconds() / 1000, // us
			},
			T: time.Now(),
			Meta: map[string]string{
				"action": fmt.Sprintf("%T", c.action),
				"status": fmt.Sprintf("%v", r.Status()),
			},
		}
		c.Ctx.Stats().Add(&metric)

		return r
	}
}

func mwPanic(next CallFunc) CallFunc {
	return func(c *Context) Renderer {
		c.Ctx.Trace("http.mw.panic.call")

		p := false
		var r Renderer

		// Wrap call to the next middleware
		func() {
			defer func() {
				if err := recover(); err != nil {
					p = true
					c.Ctx.Error("PANIC!", err)
				}
			}()

			r = next(c)
		}()

		if p {
			return c.Head(http.StatusInternalServerError)
		}
		return r
	}
}
