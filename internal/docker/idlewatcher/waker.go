package idlewatcher

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	gphttp "github.com/yusing/go-proxy/internal/net/http"
)

type Waker struct {
	*watcher

	client *http.Client
	rp     *gphttp.ReverseProxy
}

func NewWaker(w *watcher, rp *gphttp.ReverseProxy) *Waker {
	tr := &http.Transport{}
	if w.NoTLSVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Waker{
		watcher: w,
		client: &http.Client{
			Timeout:   1 * time.Second,
			Transport: tr,
		},
		rp: rp,
	}
}

func (w *Waker) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	w.wake(w.rp.ServeHTTP, rw, r)
}

func (w *Waker) wake(next http.HandlerFunc, rw http.ResponseWriter, r *http.Request) {
	// pass through if container is ready
	if w.ready.Load() {
		next(rw, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), w.WakeTimeout)
	defer cancel()

	isCheckRedirect := r.Header.Get(headerCheckRedirect) != ""
	if !isCheckRedirect {
		// Send a loading response to the client
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Write(w.makeRespBody("%s waking up...", w.ContainerName))
		return
	}

	// wake the container and reset idle timer
	// also wait for another wake request
	w.wakeCh <- struct{}{}

	if <-w.wakeDone != nil {
		http.Error(rw, "Error sending wake request", http.StatusInternalServerError)
		return
	}

	// maybe another request came in while we were waiting for the wake
	if w.ready.Load() {
		if isCheckRedirect {
			rw.WriteHeader(http.StatusOK)
		} else {
			next(rw, r)
		}
		return
	}

	for {
		select {
		case <-ctx.Done():
			http.Error(rw, "Waking timed out", http.StatusGatewayTimeout)
			return
		default:
		}

		wakeReq, err := http.NewRequestWithContext(
			ctx,
			http.MethodHead,
			w.URL.String(),
			nil,
		)
		if err != nil {
			w.l.Errorf("new request err to %s: %s", r.URL, err)
			http.Error(rw, "Internal server error", http.StatusInternalServerError)
			return
		}

		// we don't care about the response
		_, err = w.client.Do(wakeReq)
		if err == nil {
			w.ready.Store(true)
			rw.WriteHeader(http.StatusOK)
			return
		}

		// retry until the container is ready or timeout
		time.Sleep(100 * time.Millisecond)
	}
}
