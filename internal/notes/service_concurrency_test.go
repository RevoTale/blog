package notes

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"blog/internal/imageloader"
	"github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/require"
)

type parallelProbeClient struct {
	authorsStarted chan struct{}
	tagsStarted    chan struct{}
	listStarted    chan struct{}
	release        chan struct{}

	authorsOnce sync.Once
	tagsOnce    sync.Once
	listOnce    sync.Once
}

func newParallelProbeClient() *parallelProbeClient {
	return &parallelProbeClient{
		authorsStarted: make(chan struct{}),
		tagsStarted:    make(chan struct{}),
		listStarted:    make(chan struct{}),
		release:        make(chan struct{}),
	}
}

func (c *parallelProbeClient) MakeRequest(
	ctx context.Context,
	req *graphql.Request,
	resp *graphql.Response,
) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	switch req.OpName {
	case "AvailableAuthors":
		c.authorsOnce.Do(func() { close(c.authorsStarted) })
		if err := c.awaitRelease(ctx); err != nil {
			return err
		}
		return decodeClientPayload(resp, `{"Authors":{"docs":[]}}`)
	case "AvailableTagsByPostType":
		c.tagsOnce.Do(func() { close(c.tagsStarted) })
		if err := c.awaitRelease(ctx); err != nil {
			return err
		}
		return decodeClientPayload(resp, `{"availableTagsByMicroPostType":[]}`)
	case "ListNotes":
		c.listOnce.Do(func() { close(c.listStarted) })
		if err := c.awaitRelease(ctx); err != nil {
			return err
		}
		return decodeClientPayload(resp, `{"Micro_posts":{"totalPages":1,"docs":[]}}`)
	default:
		return fmt.Errorf("unexpected operation %q", req.OpName)
	}
}

func (c *parallelProbeClient) awaitRelease(ctx context.Context) error {
	select {
	case <-c.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func decodeClientPayload(resp *graphql.Response, payload string) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}
	return json.Unmarshal([]byte(payload), resp.Data)
}

func TestServiceListNotes_StartsIndependentFetchesInParallel(t *testing.T) {
	t.Parallel()

	client := newParallelProbeClient()
	service := NewService(client, 12, "https://example.com", imageloader.New(false))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := service.ListNotes(ctx, "en", ListFilter{}, ListOptions{})
		resultCh <- err
	}()

	waitStarted := func(ch <-chan struct{}) bool {
		select {
		case <-ch:
			return true
		case <-time.After(250 * time.Millisecond):
			return false
		}
	}

	authorsStarted := waitStarted(client.authorsStarted)
	tagsStarted := waitStarted(client.tagsStarted)
	listStarted := waitStarted(client.listStarted)

	close(client.release)

	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "ListNotes did not finish in time")
	}

	require.True(t, authorsStarted, "expected AvailableAuthors to start")
	require.True(t, tagsStarted, "expected AvailableTagsByPostType to start in parallel")
	require.True(t, listStarted, "expected ListNotes to start in parallel when no tag filter is set")
}
