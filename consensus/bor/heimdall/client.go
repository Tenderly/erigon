package heimdall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/tenderly/erigon/erigon-lib/metrics"

	"github.com/ledgerwatch/log/v3"
	"github.com/tenderly/erigon/consensus/bor/clerk"
	"github.com/tenderly/erigon/consensus/bor/heimdall/checkpoint"
	"github.com/tenderly/erigon/consensus/bor/heimdall/milestone"
	"github.com/tenderly/erigon/consensus/bor/heimdall/span"
)

var (
	// ErrShutdownDetected is returned if a shutdown was detected
	ErrShutdownDetected      = errors.New("shutdown detected")
	ErrNoResponse            = errors.New("got a nil response")
	ErrNotSuccessfulResponse = errors.New("error while fetching data from Heimdall")
	ErrNotInRejectedList     = errors.New("milestoneID doesn't exist in rejected list")
	ErrNotInMilestoneList    = errors.New("milestoneID doesn't exist in Heimdall")
	ErrServiceUnavailable    = errors.New("service unavailable")
)

const (
	stateFetchLimit    = 50
	apiHeimdallTimeout = 10 * time.Second
	retryCall          = 5 * time.Second
)

type StateSyncEventsResponse struct {
	Height string                       `json:"height"`
	Result []*clerk.EventRecordWithTime `json:"result"`
}

type SpanResponse struct {
	Height string            `json:"height"`
	Result span.HeimdallSpan `json:"result"`
}

type HeimdallClient struct {
	urlString string
	client    http.Client
	closeCh   chan struct{}
	logger    log.Logger
}

type Request struct {
	client http.Client
	url    *url.URL
	start  time.Time
}

func NewHeimdallClient(urlString string, logger log.Logger) *HeimdallClient {
	return &HeimdallClient{
		urlString: urlString,
		logger:    logger,
		client: http.Client{
			Timeout: apiHeimdallTimeout,
		},
		closeCh: make(chan struct{}),
	}
}

const (
	fetchStateSyncEventsFormat = "from-id=%d&to-time=%d&limit=%d"
	fetchStateSyncEventsPath   = "clerk/event-record/list"

	fetchCheckpoint      = "/checkpoints/%s"
	fetchCheckpointCount = "/checkpoints/count"

	fetchMilestone      = "/milestone/latest"
	fetchMilestoneCount = "/milestone/count"

	fetchLastNoAckMilestone = "/milestone/lastNoAck"
	fetchNoAckMilestone     = "/milestone/noAck/%s"
	fetchMilestoneID        = "/milestone/ID/%s"

	fetchSpanFormat = "bor/span/%d"
)

func (h *HeimdallClient) StateSyncEvents(ctx context.Context, fromID uint64, to int64) ([]*clerk.EventRecordWithTime, error) {
	eventRecords := make([]*clerk.EventRecordWithTime, 0)

	for {
		url, err := stateSyncURL(h.urlString, fromID, to)
		if err != nil {
			return nil, err
		}

		h.logger.Debug("[bor.heimdall] Fetching state sync events", "queryParams", url.RawQuery)

		ctx = withRequestType(ctx, stateSyncRequest)

		response, err := FetchWithRetry[StateSyncEventsResponse](ctx, h.client, url, h.closeCh, h.logger)
		if err != nil {
			return nil, err
		}

		if response == nil || response.Result == nil {
			// status 204
			break
		}

		eventRecords = append(eventRecords, response.Result...)

		if len(response.Result) < stateFetchLimit {
			break
		}

		fromID += uint64(stateFetchLimit)
	}

	sort.SliceStable(eventRecords, func(i, j int) bool {
		return eventRecords[i].ID < eventRecords[j].ID
	})

	return eventRecords, nil
}

func (h *HeimdallClient) Span(ctx context.Context, spanID uint64) (*span.HeimdallSpan, error) {
	url, err := spanURL(h.urlString, spanID)
	if err != nil {
		return nil, err
	}

	ctx = withRequestType(ctx, spanRequest)

	response, err := FetchWithRetry[SpanResponse](ctx, h.client, url, h.closeCh, h.logger)
	if err != nil {
		return nil, err
	}

	return &response.Result, nil
}

// FetchCheckpoint fetches the checkpoint from heimdall
func (h *HeimdallClient) FetchCheckpoint(ctx context.Context, number int64) (*checkpoint.Checkpoint, error) {
	url, err := checkpointURL(h.urlString, number)
	if err != nil {
		return nil, err
	}

	ctx = withRequestType(ctx, checkpointRequest)

	response, err := FetchWithRetry[checkpoint.CheckpointResponse](ctx, h.client, url, h.closeCh, h.logger)
	if err != nil {
		return nil, err
	}

	return &response.Result, nil
}

// FetchMilestone fetches the checkpoint from heimdall
func (h *HeimdallClient) FetchMilestone(ctx context.Context) (*milestone.Milestone, error) {
	url, err := milestoneURL(h.urlString)
	if err != nil {
		return nil, err
	}

	ctx = withRequestType(ctx, milestoneRequest)

	response, err := FetchWithRetry[milestone.MilestoneResponse](ctx, h.client, url, h.closeCh, h.logger)
	if err != nil {
		return nil, err
	}

	return &response.Result, nil
}

// FetchCheckpointCount fetches the checkpoint count from heimdall
func (h *HeimdallClient) FetchCheckpointCount(ctx context.Context) (int64, error) {
	url, err := checkpointCountURL(h.urlString)
	if err != nil {
		return 0, err
	}

	ctx = withRequestType(ctx, checkpointCountRequest)

	response, err := FetchWithRetry[checkpoint.CheckpointCountResponse](ctx, h.client, url, h.closeCh, h.logger)
	if err != nil {
		return 0, err
	}

	return response.Result.Result, nil
}

// FetchMilestoneCount fetches the milestone count from heimdall
func (h *HeimdallClient) FetchMilestoneCount(ctx context.Context) (int64, error) {
	url, err := milestoneCountURL(h.urlString)
	if err != nil {
		return 0, err
	}

	ctx = withRequestType(ctx, milestoneCountRequest)

	response, err := FetchWithRetry[milestone.MilestoneCountResponse](ctx, h.client, url, h.closeCh, h.logger)
	if err != nil {
		return 0, err
	}

	return response.Result.Count, nil
}

// FetchLastNoAckMilestone fetches the last no-ack-milestone from heimdall
func (h *HeimdallClient) FetchLastNoAckMilestone(ctx context.Context) (string, error) {
	url, err := lastNoAckMilestoneURL(h.urlString)
	if err != nil {
		return "", err
	}

	ctx = withRequestType(ctx, milestoneLastNoAckRequest)

	response, err := FetchWithRetry[milestone.MilestoneLastNoAckResponse](ctx, h.client, url, h.closeCh, h.logger)
	if err != nil {
		return "", err
	}

	return response.Result.Result, nil
}

// FetchNoAckMilestone fetches the last no-ack-milestone from heimdall
func (h *HeimdallClient) FetchNoAckMilestone(ctx context.Context, milestoneID string) error {
	url, err := noAckMilestoneURL(h.urlString, milestoneID)
	if err != nil {
		return err
	}

	ctx = withRequestType(ctx, milestoneNoAckRequest)

	response, err := FetchWithRetry[milestone.MilestoneNoAckResponse](ctx, h.client, url, h.closeCh, h.logger)
	if err != nil {
		return err
	}

	if !response.Result.Result {
		return fmt.Errorf("%w: milestoneID %q", ErrNotInRejectedList, milestoneID)
	}

	return nil
}

// FetchMilestoneID fetches the bool result from Heimdal whether the ID corresponding
// to the given milestone is in process in Heimdall
func (h *HeimdallClient) FetchMilestoneID(ctx context.Context, milestoneID string) error {
	url, err := milestoneIDURL(h.urlString, milestoneID)
	if err != nil {
		return err
	}

	ctx = withRequestType(ctx, milestoneIDRequest)

	response, err := FetchWithRetry[milestone.MilestoneIDResponse](ctx, h.client, url, h.closeCh, h.logger)

	if err != nil {
		return err
	}

	if !response.Result.Result {
		return fmt.Errorf("%w: milestoneID %q", ErrNotInMilestoneList, milestoneID)
	}

	return nil
}

// FetchWithRetry returns data from heimdall with retry
func FetchWithRetry[T any](ctx context.Context, client http.Client, url *url.URL, closeCh chan struct{}, logger log.Logger) (*T, error) {
	// request data once
	request := &Request{client: client, url: url, start: time.Now()}
	result, err := Fetch[T](ctx, request)
	if err == nil {
		return result, nil
	}

	// 503 (Service Unavailable) is thrown when an endpoint isn't activated
	// yet in heimdall. E.g. when the hardfork hasn't hit yet but heimdall
	// is upgraded.
	if errors.Is(err, ErrServiceUnavailable) {
		logger.Debug("[bor.heimdall] service unavailable at the moment", "path", url.Path, "error", err)
		return nil, err
	}

	// attempt counter
	attempt := 1

	logger.Warn("[bor.heimdall] an error while fetching", "path", url.Path, "attempt", attempt, "error", err)

	// create a new ticker for retrying the request
	ticker := time.NewTicker(retryCall)
	defer ticker.Stop()

	const logEach = 5

retryLoop:
	for {
		attempt++

		select {
		case <-ctx.Done():
			logger.Debug("[bor.heimdall] request canceled", "reason", ctx.Err(), "path", url.Path, "attempt", attempt)
			return nil, ctx.Err()
		case <-closeCh:
			logger.Debug("[bor.heimdall] shutdown detected, terminating request", "path", url.Path)

			return nil, ErrShutdownDetected
		case <-ticker.C:
			request = &Request{client: client, url: url, start: time.Now()}
			result, err = Fetch[T](ctx, request)

			if errors.Is(err, ErrServiceUnavailable) {
				logger.Debug("[bor.heimdall] service unavailable at the moment", "path", url.Path, "attempt", attempt, "error", err)
				return nil, err
			}

			if err != nil {
				if attempt%logEach == 0 {
					logger.Warn("[bor.heimdall] an error while trying fetching", "path", url.Path, "attempt", attempt, "error", err)
				}

				continue retryLoop
			}

			return result, nil
		}
	}
}

// Fetch fetches response from heimdall
func Fetch[T any](ctx context.Context, request *Request) (*T, error) {
	isSuccessful := false

	defer func() {
		if metrics.EnabledExpensive {
			sendMetrics(ctx, request.start, isSuccessful)
		}
	}()

	result := new(T)

	body, err := internalFetchWithTimeout(ctx, request.client, request.url)
	if err != nil {
		return nil, err
	}

	if body == nil {
		return nil, ErrNoResponse
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err
	}

	isSuccessful = true

	return result, nil
}

func spanURL(urlString string, spanID uint64) (*url.URL, error) {
	return makeURL(urlString, fmt.Sprintf(fetchSpanFormat, spanID), "")
}

func stateSyncURL(urlString string, fromID uint64, to int64) (*url.URL, error) {
	queryParams := fmt.Sprintf(fetchStateSyncEventsFormat, fromID, to, stateFetchLimit)

	return makeURL(urlString, fetchStateSyncEventsPath, queryParams)
}

func checkpointURL(urlString string, number int64) (*url.URL, error) {
	url := ""
	if number == -1 {
		url = fmt.Sprintf(fetchCheckpoint, "latest")
	} else {
		url = fmt.Sprintf(fetchCheckpoint, fmt.Sprint(number))
	}

	return makeURL(urlString, url, "")
}

func milestoneURL(urlString string) (*url.URL, error) {
	url := fetchMilestone
	return makeURL(urlString, url, "")
}

func checkpointCountURL(urlString string) (*url.URL, error) {
	return makeURL(urlString, fetchCheckpointCount, "")
}

func milestoneCountURL(urlString string) (*url.URL, error) {
	return makeURL(urlString, fetchMilestoneCount, "")
}
func lastNoAckMilestoneURL(urlString string) (*url.URL, error) {
	return makeURL(urlString, fetchLastNoAckMilestone, "")
}

func noAckMilestoneURL(urlString string, id string) (*url.URL, error) {
	return makeURL(urlString, fmt.Sprintf(fetchNoAckMilestone, id), "")
}

func milestoneIDURL(urlString string, id string) (*url.URL, error) {
	return makeURL(urlString, fmt.Sprintf(fetchMilestoneID, id), "")
}

func makeURL(urlString, rawPath, rawQuery string) (*url.URL, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	u.Path = rawPath
	u.RawQuery = rawQuery

	return u, err
}

// internal fetch method
func internalFetch(ctx context.Context, client http.Client, u *url.URL) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusServiceUnavailable {
		return nil, fmt.Errorf("%w: response code %d", ErrServiceUnavailable, res.StatusCode)
	}

	// check status code
	if res.StatusCode != 200 && res.StatusCode != 204 {
		return nil, fmt.Errorf("%w: %s:response code %d", ErrNotSuccessfulResponse, u.String(), res.StatusCode)
	}

	// unmarshall data from buffer
	if res.StatusCode == 204 {
		return nil, nil
	}

	// get response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func internalFetchWithTimeout(ctx context.Context, client http.Client, url *url.URL) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, apiHeimdallTimeout)
	defer cancel()

	// request data once
	return internalFetch(ctx, client, url)
}

// Close sends a signal to stop the running process
func (h *HeimdallClient) Close() {
	close(h.closeCh)
	h.client.CloseIdleConnections()
}
