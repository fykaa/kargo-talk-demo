package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

type MockKargoMessage struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string            `json:"name"`
		Namespace string            `json:"namespace"`
		Labels    map[string]string `json:"labels,omitempty"`
	} `json:"metadata"`
	Spec struct {
		SlackChannel string `json:"slackChannel"`
		Message      string `json:"message"`
		Team         string `json:"team,omitempty"`
		ChannelType  string `json:"channelType,omitempty"`
		Subscriptions []struct {
			Stage string `json:"stage"`
			Events []string `json:"events"`
		} `json:"subscriptions,omitempty"`
	} `json:"spec"`
	Status struct {
		CreatedAt time.Time `json:"createdAt,omitempty"`
		State     string    `json:"state,omitempty"`
	} `json:"status,omitempty"`
}

type WebhookRequest struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Request    struct {
		UID             string `json:"uid"`
		DryRun          bool   `json:"dryRun"`
		UserInfo        any    `json:"userInfo"`
		Object          any    `json:"object"`
		OldObject       any    `json:"oldObject,omitempty"`
		Resource        struct {
			Group     string `json:"group"`
			Version   string `json:"version"`
			Resource  string `json:"resource"`
		} `json:"resource"`
		SubResource string `json:"subResource"`
	} `json:"request"`
}

type WebhookResponse struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Response   struct {
		UID           string `json:"uid"`
		Allowed       bool   `json:"allowed"`
		Patch         []byte `json:"patch,omitempty"`
		PatchType     string `json:"patchType,omitempty"`
		Result        any    `json:"result,omitempty"`
	} `json:"response"`
}

type MockSlackClient struct {
	mu             sync.RWMutex
	channels       map[string]bool
	conversations  map[string]string
	lastChannelReq string
}

func NewMockSlackClient() *MockSlackClient {
	return &MockSlackClient{
		channels:      make(map[string]bool),
		conversations: make(map[string]string),
	}
}

func (m *MockSlackClient) CreateConversation(ctx context.Context, name, isPrivate bool) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	channelID := fmt.Sprintf("C%08x", len(m.channels))
	m.channels[channelID] = isPrivate
	m.lastChannelReq = name
	
	klog.Infof("MockSlack: Created channel %s (ID: %s, private: %v)", name, channelID, isPrivate)
	return channelID, nil
}

func (m *MockSlackClient) ChannelExists(channelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.channels[channelID]
	return exists
}

type Validator struct {
	slackClient *MockSlackClient
	timeout     time.Duration
}

func NewValidator(slackClient *MockSlackClient) *Validator {
	return &Validator{
		slackClient: slackClient,
		timeout:     30 * time.Second,
	}
}

func (v *Validator) ValidateMessage(ctx context.Context, msg *MockKargoMessage) (*WebhookResponse, error) {
	resp := &WebhookResponse{
		APIVersion: "admission.k8s.io/v1",
		Kind:       "AdmissionReview",
	}

	resp.Response.Allowed = true
	resp.Response.UID = fmt.Sprintf("test-uid-%d", time.Now().UnixNano())

	asyncCtx, cancel := context.WithTimeout(ctx, v.timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		err := v.validateSlackChannelCreation(asyncCtx, msg)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			klog.Errorf("Slack channel validation failed: %v", err)
			resp.Response.Allowed = false
			resp.Response.Result = map[string]string{
				"status": "Failure",
				"reason": fmt.Sprintf("Slack channel validation failed: %v", err),
			}
			return resp, err
		case <-asyncCtx.Done():
			klog.Errorf("Slack validation timeout after %v", v.timeout)
			resp.Response.Allowed = false
			resp.Response.Result = map[string]string{
				"status": "Failure",
				"reason": "Slack channel validation timeout",
			}
			return resp, asyncCtx.Err()
	}

	klog.Infof("Successfully validated Kargo message %s/%s for Slack channel %s", 
		msg.Metadata.Namespace, msg.Metadata.Name, msg.Spec.SlackChannel)
	return resp, nil
}

func (v *Validator) validateSlackChannelCreation(ctx context.Context, msg *MockKargoMessage) error {
	if msg.Spec.SlackChannel == "" {
		return fmt.Errorf("slackChannel is required")
	}

	if msg.Metadata.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	channelID, err := v.slackClient.CreateConversation(ctx, msg.Spec.SlackChannel, 
		msg.Spec.ChannelType == "private")
	if err != nil {
		return fmt.Errorf("failed to create Slack channel: %w", err)
	}

	if !v.slackClient.ChannelExists(channelID) {
		return fmt.Errorf("Slack channel %s not found after creation", channelID)
	}

	time.Sleep(100 * time.Millisecond)
	
	for _, sub := range msg.Spec.Subscriptions {
		if sub.Stage == "" {
			return fmt.Errorf("subscription stage cannot be empty")
		}
		if len(sub.Events) == 0 {
			return fmt.Errorf("subscription must have at least one event")
		}
	}

	klog.Infof("Slack channel %s validated successfully for message %s", 
		msg.Spec.SlackChannel, msg.Metadata.Name)
	return nil
}

func (v *Validator) WebhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req WebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	var msg MockKargoMessage
	if obj, ok := req.Request.Object.(map[string]interface{}); ok {
		msgBytes, _ := json.Marshal(obj)
		json.Unmarshal(msgBytes, &msg)
	} else {
		http.Error(w, "Invalid object format", http.StatusBadRequest)
		return
	}

	resp, err := v.ValidateMessage(r.Context(), &msg)
	if err != nil {
		klog.Errorf("Webhook validation failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func TestWebhookValidator_Success(t *testing.T) {
	slackClient := NewMockSlackClient()
	validator := NewValidator(slackClient)

	msg := &MockKargoMessage{
		APIVersion: "kargo.akuity.io/v1alpha1",
		Kind:       "SlackMessage",
		Metadata: struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels,omitempty"`
		}{
			Name:      "test-slack-msg",
			Namespace: "kargo",
			Labels:    map[string]string{"app": "kargo"},
		},
		Spec: struct {
			SlackChannel string `json:"slackChannel"`
			Message      string `json:"message"`
			Team         string `json:"team,omitempty"`
			ChannelType  string `json:"channelType,omitempty"`
			Subscriptions []struct {
				Stage string `json:"stage"`
				Events []string `json:"events"`
			} `json:"subscriptions,omitempty"`
		}{
			SlackChannel: "kargo-notifications",
			Message:      "Pipeline {{.Stage.Name}} completed successfully",
			ChannelType:  "public",
			Subscriptions: []struct {
				Stage string `json:"stage"`
				Events []string `json:"events"`
			}{
				{Stage: "production", Events: []string{"PromoteSucceeded"}},
				{Stage: "staging", Events: []string{"PromoteFailed"}},
			},
		},
	}

	resp, err := validator.ValidateMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.True(t, resp.Response.Allowed, "Validation should allow valid message")
	assert.NotEmpty(t, resp.Response.UID)
}

func TestWebhookValidator_MissingChannel(t *testing.T) {
	slackClient := NewMockSlackClient()
	validator := NewValidator(slackClient)

	msg := &MockKargoMessage{
		APIVersion: "kargo.akuity.io/v1alpha1",
		Kind:       "SlackMessage",
		Metadata: struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels,omitempty"`
		}{
			Name:      "invalid-msg",
			Namespace: "kargo",
		},
		Spec: struct {
			SlackChannel string `json:"slackChannel"`
			Message      string `json:"message"`
			Team         string `json:"team,omitempty"`
			ChannelType  string `json:"channelType,omitempty"`
			Subscriptions []struct {
				Stage string `json:"stage"`
				Events []string `json:"events"`
			} `json:"subscriptions,omitempty"`
		}{
			Message: "Test message",
		},
	}

	resp, err := validator.ValidateMessage(context.Background(), msg)
	require.Error(t, err)
	assert.False(t, resp.Response.Allowed)
	assert.Contains(t, resp.Response.Result.(map[string]string)["reason"], "slackChannel is required")
}

func TestWebhookValidator_HTTPHandler(t *testing.T) {
	slackClient := NewMockSlackClient()
	validator := NewValidator(slackClient)

	server := httptest.NewServer(http.HandlerFunc(validator.WebhookHandler))
	defer server.Close()

	validMsg := MockKargoMessage{
		APIVersion: "kargo.akuity.io/v1alpha1",
		Kind:       "SlackMessage",
		Metadata: struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels,omitempty"`
		}{
			Name:      "http-test",
			Namespace: "default",
		},
		Spec: struct {
			SlackChannel string `json:"slackChannel"`
			Message      string `json:"message"`
			Team         string `json:"team,omitempty"`
			ChannelType  string `json:"channelType,omitempty"`
			Subscriptions []struct {
				Stage string `json:"stage"`
				Events []string `json:"events"`
			} `json:"subscriptions,omitempty"`
		}{
			SlackChannel: "devops-notifications",
			Message:      "Deployment {{.Pipeline.Name}} succeeded",
		},
	}

	reqBody := WebhookRequest{
		APIVersion: "admission.k8s.io/v1",
		Kind:       "AdmissionReview",
		Request: struct {
			UID             string `json:"uid"`
			DryRun          bool   `json:"dryRun"`
			UserInfo        any    `json:"userInfo"`
			Object          any    `json:"object"`
			OldObject       any    `json:"oldObject,omitempty"`
			Resource        struct {
				Group     string `json:"group"`
				Version   string `json:"version"`
				Resource  string `json:"resource"`
			} `json:"resource"`
			SubResource string `json:"subResource"`
		}{
			UID:     "test-http-uid",
			DryRun:  false,
			Object:  validMsg,
			Resource: struct {
				Group    string `json:"group"`
				Version  string `json:"version"`
				Resource string `json:"resource"`
			}{
				Group:    "kargo.akuity.io",
				Version:  "v1alpha1",
				Resource: "slackmessages",
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(bodyBytes))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var webhookResp WebhookResponse
	json.NewDecoder(resp.Body).Decode(&webhookResp)
	assert.True(t, webhookResp.Response.Allowed)
	assert.NotEmpty(t, slackClient.lastChannelReq)
	assert.Equal(t, "devops-notifications", slackClient.lastChannelReq)
}

func TestConcurrentValidations(t *testing.T) {
	slackClient := NewMockSlackClient()
	validator := NewValidator(slackClient)

	const numGoroutines = 50
	var wg sync.WaitGroup
	results := make(chan *WebhookResponse, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := &MockKargoMessage{
				APIVersion: "kargo.akuity.io/v1alpha1",
				Kind:       "SlackMessage",
				Metadata: struct {
					Name      string            `json:"name"`
					Namespace string            `json:"namespace"`
					Labels    map[string]string `json:"labels,omitempty"`
				}{
					Name:      fmt.Sprintf("concurrent-%d", id),
					Namespace: "concurrent-test",
				},
				Spec: struct {
					SlackChannel string `json:"slackChannel"`
					Message      string `json:"message"`
					Team         string `json:"team,omitempty"`
					ChannelType  string `json:"channelType,omitempty"`
					Subscriptions []struct {
						Stage string `json:"stage"`
						Events []string `json:"events"`
					} `json:"subscriptions,omitempty"`
				}{
					SlackChannel: fmt.Sprintf("team-channel-%d", id),
					Message:      fmt.Sprintf("Concurrent test %d", id),
				},
			}

			resp, _ := validator.ValidateMessage(context.Background(), msg)
			results <- resp
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	for resp := range results {
		if resp.Response.Allowed {
			successCount++
		}
	}

	assert.Greater(t, successCount, numGoroutines/2, "At least half of concurrent validations should succeed")
	assert.GreaterOrEqual(t, len(slackClient.channels), numGoroutines/2, "Should create multiple Slack channels")
}

func TestTimeoutValidation(t *testing.T) {
	slackClient := NewMockSlackClient()
	validator := &Validator{
		slackClient: slackClient,
		timeout:     10 * time.Millisecond,
	}

	originalCreate := slackClient.CreateConversation
	slowCtx := context.Background()
	
	msg := &MockKargoMessage{
		APIVersion: "kargo.akuity.io/v1alpha1",
		Kind:       "SlackMessage",
		Metadata: struct {
			Name      string            `json:"name"`
			Namespace string            `json:"namespace"`
			Labels    map[string]string `json:"labels,omitempty"`
		}{
			Name:      "timeout-test",
			Namespace: "kargo",
		},
		Spec: struct {
			SlackChannel string `json:"slackChannel"`
			Message      string `json:"message"`
			Team         string `json:"team,omitempty"`
			ChannelType  string `json:"channelType,omitempty"`
			Subscriptions []struct {
				Stage string `json:"stage"`
				Events []string `json:"events"`
			} `json:"subscriptions,omitempty"`
		}{
			SlackChannel: "timeout-channel",
			Message:      "This should timeout",
		},
	}

	resp, err := validator.ValidateMessage(slowCtx, msg)
	assert.Error(t, err)
	assert.False(t, resp.Response.Allowed)
	assert.Contains(t, resp.Response.Result.(map[string]string)["reason"], "timeout")
}

func main() {
	fmt.Println("Run tests with: go test -v ./...")
}
