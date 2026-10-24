package context

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	curlerrors "github.com/cdwiegand/go-curling/errors"
)

// Close releases bodies and the deadline after the final body has been consumed.
func (resp *CurlResponses) Close() {
	if resp == nil {
		return
	}
	if len(resp.Responses) > 0 {
		last := resp.Responses[len(resp.Responses)-1]
		if last.Timings != nil && last.Timings.BodyDone.IsZero() {
			last.Timings.BodyDone = time.Now()
		}
	}
	for _, r := range resp.Responses {
		if r.HttpResponse != nil && r.HttpResponse.Body != nil {
			r.HttpResponse.Body.Close()
		}
	}
	if resp.cancel != nil {
		resp.cancel()
		resp.cancel = nil
	}
}

func sameOrigin(a, b *url.URL) bool {
	port := func(u *url.URL) string {
		if p := u.Port(); p != "" {
			return p
		}
		if strings.EqualFold(u.Scheme, "https") {
			return "443"
		}
		return "80"
	}
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Hostname(), b.Hostname()) && port(a) == port(b)
}

func replayRequest(request *http.Request, c context.Context) (*http.Request, error) {
	r := request.Clone(c)
	if request.Body != nil && request.Body != http.NoBody {
		if request.GetBody == nil {
			return nil, fmt.Errorf("request body cannot be replayed")
		}
		body, err := request.GetBody()
		if err != nil {
			return nil, err
		}
		r.Body = body
	}
	return r, nil
}

// GetCompleteResponse fetches headers and follows redirects. Callers must consume
// the final body and Close the result. Request/status retries happen here.
func (ctx *CurlContext) GetCompleteResponse(index int, client *http.Client, request *http.Request) (*CurlResponses, *curlerrors.CurlError) {
	return ctx.completeTransfer(index, client, request, false)
}

// PerformTransfer includes output handling, so body-read failures can also be
// retried. File output is rolled back before replaying a failed attempt.
func (ctx *CurlContext) PerformTransfer(index int, client *http.Client, request *http.Request) (*CurlResponses, *curlerrors.CurlError) {
	return ctx.completeTransfer(index, client, request, true)
}

func (ctx *CurlContext) completeTransfer(index int, client *http.Client, request *http.Request, emit bool) (*CurlResponses, *curlerrors.CurlError) {
	started := time.Now()
	if request.Body != nil {
		defer request.Body.Close()
	}
	for retry := 0; ; retry++ {
		resp, err := ctx.getRedirectChain(client, request)
		resp.OperationStart, resp.Error = started, err
		last := resp.Responses[len(resp.Responses)-1]
		retryable := last.HttpResponse != nil && ctx.canStatusCodeRetry(last.HttpResponse.StatusCode)
		if last.Error != nil {
			retryable = ctx.canRetryError(last.Error)
		}
		delay := ctx.retryDelay(retry, last.HttpResponse)
		retryAllowed := ctx.retryAllowed(retry, started, delay)
		var offsets map[string]int64
		emitted := false
		if emit && err == nil && !(retryable && retryAllowed) {
			var offsetErr error
			offsets, offsetErr = ctx.outputOffsets(index, resp)
			if offsetErr != nil {
				err = curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_WRITE_FILE, offsetErr)
			} else {
				// Cleanup belongs to the final failed attempt, after deciding
				// whether output will be rolled back and retried.
				remove := ctx.RemoveOnError
				ctx.RemoveOnError = false
				errs := ctx.ProcessResponseToOutputs(index, resp, request)
				ctx.RemoveOnError = remove
				emitted = true
				if errs.HasError() {
					err = errs.Errors[len(errs.Errors)-1]
				}
			}
			if err != nil {
				retryable = ctx.RetryAllErrors || err.ExitCode == curlerrors.ERROR_TIMEOUT
			}
			retryAllowed = ctx.retryAllowed(retry, started, delay)
		}
		resp.Error = err
		if err != nil {
			resp.IsError = true
		}
		if !retryable || !retryAllowed {
			if err != nil && emitted && ctx.RemoveOnError {
				errs := ctx.removeBodyOutputs()
				if errs.HasError() {
					err = errs.Errors[len(errs.Errors)-1]
					resp.Error = err
				}
			}
			if emit {
				resp.Close()
			}
			return resp, err
		}
		resp.Close()
		if emitted {
			if rollbackErr := ctx.rollbackBodyOutputs(offsets); rollbackErr != nil {
				err = curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_WRITE_FILE, rollbackErr)
				resp.Error = err
				if ctx.RemoveOnError {
					ctx.removeBodyOutputs()
				}
				return resp, err
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-request.Context().Done():
			timer.Stop()
			err = transferError(request.Context().Err())
			resp.Error = err
			return resp, err
		case <-timer.C:
		}
	}
}

func (ctx *CurlContext) canRetryError(err error) bool {
	if ctx.RetryAllErrors || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return true
	}
	// Winsock returns WSAECONNREFUSED (10061), distinct from Go's
	// portable syscall.ECONNREFUSED value on Windows.
	return ctx.RetryConnRefused && (errors.Is(err, syscall.ECONNREFUSED) ||
		(runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(10061))))
}

func (ctx *CurlContext) retryAllowed(retry int, started time.Time, delay time.Duration) bool {
	if retry >= ctx.MaxRetries {
		return false
	}
	if ctx.RetryMaxTime > 0 {
		budget := time.Duration(ctx.RetryMaxTime * float64(time.Second))
		elapsed := time.Since(started)
		if elapsed >= budget || delay >= budget-elapsed {
			return false
		}
	}
	return true
}

func (ctx *CurlContext) getRedirectChain(client *http.Client, original *http.Request) (result *CurlResponses, cerr *curlerrors.CurlError) {
	result = &CurlResponses{OperationStart: time.Now()}
	c := original.Context()
	if ctx.MaxTime > 0 {
		c, result.cancel = context.WithTimeout(c, time.Duration(float64(ctx.MaxTime)*float64(time.Second)))
	}
	defer func() {
		result.Error = cerr
		if cerr != nil {
			result.IsError = true
			result.Close()
		}
	}()
	r, err := replayRequest(original, c)
	if err != nil {
		result.Responses = append(result.Responses, &CurlResponse{Error: err})
		return result, curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_READ_FILE, err)
	}
	for {
		response := GetCurlResponse(client, r)
		result.Responses = append(result.Responses, response)
		if response.Error != nil {
			return result, transferError(response.Error)
		}
		result.IsError = response.HttpResponse.StatusCode >= 400
		code := response.HttpResponse.StatusCode
		if !ctx.FollowRedirects || response.NextUrl == nil || !(code == 301 || code == 302 || code == 303 || code == 307 || code == 308) {
			return result, nil
		}
		if ctx.MaxRedirects >= 0 && result.NumRedirects >= ctx.MaxRedirects {
			return result, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_TOO_MANY_REDIRECTS, "Maximum number of redirects exceeded")
		}
		if response.NextUrl.Scheme != "http" && response.NextUrl.Scheme != "https" {
			return result, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_URL, "Redirect uses an unsupported protocol")
		}
		// Retain headers for -D/-i, but release the body before making another request.
		response.HttpResponse.Body.Close()
		dropBody := (code == 303 && r.Method != "HEAD" && !ctx.Allow303Post) ||
			(r.Method == "POST" && ((code == 301 && !ctx.Allow301Post) || (code == 302 && !ctx.Allow302Post)))
		var next *http.Request
		if dropBody {
			next = r.Clone(c)
			next.Method, next.Body, next.GetBody, next.ContentLength = "GET", nil, nil, 0
			next.Header.Del("Content-Type")
			next.Header.Del("Content-Length")
			next.TransferEncoding = nil
		} else {
			next, err = replayRequest(r, c)
			if err != nil {
				return result, curlerrors.NewCurlErrorFromError(curlerrors.ERROR_CANNOT_READ_FILE, err)
			}
		}
		next.URL = response.NextUrl
		next.Host = ""
		if ctx.RedirectsKeepAuthenticationHeaders || sameOrigin(original.URL, next.URL) {
			for _, name := range []string{"Authorization", "Cookie"} {
				next.Header.Del(name)
				for _, value := range original.Header.Values(name) {
					next.Header.Add(name, value)
				}
			}
		} else {
			next.Header.Del("Authorization")
			next.Header.Del("Cookie")
			next.URL.User = nil
		}
		r = next
		result.NumRedirects++
	}
}

func (ctx *CurlContext) retryDelay(retry int, response *http.Response) time.Duration {
	delay := time.Duration(ctx.RetryDelaySeconds) * time.Second
	if delay == 0 {
		delay = time.Second
		for i := 0; i < retry && delay < 600*time.Second; i++ {
			delay *= 2
		}
		if delay > 600*time.Second {
			delay = 600 * time.Second
		}
	}
	if response != nil {
		value := response.Header.Get("Retry-After")
		var after time.Duration
		if seconds, err := strconv.ParseInt(value, 10, 64); err == nil && seconds > 0 {
			if seconds > int64((1<<63-1)/int64(time.Second)) {
				after = time.Duration(1<<63 - 1)
			} else {
				after = time.Duration(seconds) * time.Second
			}
		} else if date, err := http.ParseTime(value); err == nil {
			after = time.Until(date)
		}
		if after > delay {
			delay = after
		}
	}
	return delay
}
