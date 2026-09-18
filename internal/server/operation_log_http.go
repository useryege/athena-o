package server

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/record"
	utilsession "github.com/useryege/athena/util/session"
)

type operationLogHTTPRoute struct {
	entry        event.Entry
	pattern      *regexp.Regexp
	placeholders []string
}

var operationLogHTTPRoutes = buildOperationLogHTTPRoutes()

func buildOperationLogHTTPRoutes() []operationLogHTTPRoute {
	var routes []operationLogHTTPRoute
	for _, entry := range event.Catalog() {
		if entry.Transport != "http" || entry.HTTPMethod == "" || entry.Route == "" {
			continue
		}
		route := entry.Route
		if purpose := strings.Index(route, " ("); purpose >= 0 {
			route = route[:purpose]
		}
		placeholders := make([]string, 0, 2)
		pattern := regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]*)\}`).ReplaceAllStringFunc(route, func(value string) string {
			name := strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}")
			placeholders = append(placeholders, name)
			return `([^/]+)`
		})
		routes = append(routes, operationLogHTTPRoute{
			entry:        entry,
			pattern:      regexp.MustCompile("^" + pattern + "$"),
			placeholders: placeholders,
		})
	}
	return routes
}

func operationLogHTTPRouteForRequest(request *http.Request) (event.Entry, map[string]string, bool) {
	preferredAction := ""
	if request.URL.Path == "/auth/google/callback" {
		state := request.URL.Query().Get("state")
		for _, candidate := range []struct {
			prefix string
			action string
		}{
			{prefix: "wcob.", action: "worm.cash_out_batch.authorize"},
			{prefix: "wco.", action: "worm.cash_out.authorize"},
			{prefix: "wex.", action: "worm.execution.authorize"},
			{prefix: "wc.", action: "worm.connection.authorize"},
			{prefix: "ws.", action: "wallet.reveal.authorize"},
		} {
			if strings.HasPrefix(state, candidate.prefix) && authregistration.ValidOpaqueValue(strings.TrimPrefix(state, candidate.prefix)) {
				preferredAction = candidate.action
				break
			}
		}
	}
	for _, route := range operationLogHTTPRoutes {
		if !strings.EqualFold(route.entry.HTTPMethod, request.Method) {
			continue
		}
		if preferredAction != "" && route.entry.ActionCode != preferredAction {
			continue
		}
		matches := route.pattern.FindStringSubmatch(request.URL.Path)
		if matches == nil {
			continue
		}
		values := make(map[string]string, len(route.placeholders))
		for i, name := range route.placeholders {
			values[name] = matches[i+1]
		}
		return route.entry, values, true
	}
	return event.Entry{}, nil, false
}

// operationLogHTTPMiddleware creates the one recorder for native HTTP and
// authentication routes. It observes only the status and response-write
// boundary; handlers remain responsible for recording durable business facts.
func (server *AthenaServer) operationLogHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// grpc-web is already observed at the gRPC facade boundary. Do not
		// create a second native HTTP operation for the same business call.
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(request.Header.Get("Content-Type"))), "application/grpc-web") {
			next.ServeHTTP(writer, request)
			return
		}
		entry, _, ok := operationLogHTTPRouteForRequest(request)
		if !ok || server == nil || server.operationLogProducer == nil {
			next.ServeHTTP(writer, request)
			return
		}
		ctx, recorder := server.beginOperationLog(request.Context(), entry.Entry)
		if recorder == nil {
			next.ServeHTTP(writer, request.WithContext(ctx))
			return
		}
		ctx = record.WithRecorder(ctx, recorder)
		recorder.Start()
		recorder.Dispatched()
		tracked := &operationLogHTTPResponseWriter{ResponseWriter: writer, recorder: recorder}
		defer func() {
			if recovered := recover(); recovered != nil {
				server.bindHTTPOperationLogActor(request, recorder)
				recorder.ObserveHTTP(http.StatusInternalServerError)
				recorder.Finish()
				panic(recovered)
			}
			server.bindHTTPOperationLogActor(request, recorder)
			finishOperationLogHTTP(recorder, tracked.statusCode())
		}()
		next.ServeHTTP(tracked, request.WithContext(ctx))
	})
}

func finishOperationLogHTTP(recorder *record.Recorder, statusCode int) {
	if recorder == nil {
		return
	}
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	recorder.ObserveHTTP(statusCode)
	if statusCode >= http.StatusBadRequest {
		switch statusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			recorder.Result(event.Denied, "")
		case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusPreconditionFailed, http.StatusUnprocessableEntity, http.StatusTooManyRequests:
			recorder.Result(event.Failed, "")
		}
	}
	recorder.Finish()
}

// bindHTTPOperationLogActor runs after the handler has had a chance to set a
// response cookie. The request itself remains immutable, so this second
// trusted lookup upgrades the initial anonymous marker without accepting any
// actor data supplied by the browser.
func (server *AthenaServer) bindHTTPOperationLogActor(request *http.Request, recorder *record.Recorder) {
	if server == nil || recorder == nil || request == nil {
		return
	}
	_, credential, err := server.authenticateRealmLoginCookie(request, false)
	if err != nil && server.DisableAuth {
		if accountID, accountErr := server.developmentAccountIDFromHTTPRequest(request, false); accountErr == nil {
			credential = accountcredentials.AuthenticatedCredential{AccountID: accountID, Capability: accountcredentials.CapabilityDevelopment, JTI: "development:" + accountID, IdentityBinding: "development:" + accountID, AccessRevision: 1}
			err = nil
		}
	}
	if err != nil || credential.AccountID == "" {
		return
	}
	ctx := utilsession.WithAuthenticatedCredential(request.Context(), credential)
	server.bindOperationLogActor(ctx, recorder)
}

type operationLogHTTPResponseWriter struct {
	http.ResponseWriter
	recorder    *record.Recorder
	status      int
	writeFailed bool
}

func (writer *operationLogHTTPResponseWriter) WriteHeader(statusCode int) {
	if writer.status == 0 {
		writer.status = statusCode
	}
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *operationLogHTTPResponseWriter) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	n, err := writer.ResponseWriter.Write(data)
	if err != nil {
		writer.writeFailed = true
		writer.recorder.ResponseWriteFailed()
	}
	return n, err
}

func (writer *operationLogHTTPResponseWriter) statusCode() int { return writer.status }

// StatusCode exposes the observed response status to native authentication
// handlers that need to classify a deferred failure. It contains no response
// body or user data.
func (writer *operationLogHTTPResponseWriter) StatusCode() int { return writer.status }

func (writer *operationLogHTTPResponseWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func (writer *operationLogHTTPResponseWriter) Flush() {
	if flusher, ok := writer.ResponseWriter.(http.Flusher); ok {
		if writer.status == 0 {
			writer.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}

func (writer *operationLogHTTPResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := writer.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (writer *operationLogHTTPResponseWriter) Push(target string, options *http.PushOptions) error {
	pusher, ok := writer.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, options)
}

func (writer *operationLogHTTPResponseWriter) ReadFrom(source io.Reader) (int64, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	if reader, ok := writer.ResponseWriter.(io.ReaderFrom); ok {
		n, err := reader.ReadFrom(source)
		if err != nil {
			writer.writeFailed = true
			writer.recorder.ResponseWriteFailed()
		}
		return n, err
	}
	n, err := io.Copy(writer.ResponseWriter, source)
	if err != nil {
		writer.writeFailed = true
		writer.recorder.ResponseWriteFailed()
	}
	return n, err
}

var _ http.ResponseWriter = (*operationLogHTTPResponseWriter)(nil)

func operationLogHTTPStatus(writer http.ResponseWriter) int {
	if statusWriter, ok := writer.(interface{ StatusCode() int }); ok {
		if statusCode := statusWriter.StatusCode(); statusCode != 0 {
			return statusCode
		}
	}
	return http.StatusInternalServerError
}
