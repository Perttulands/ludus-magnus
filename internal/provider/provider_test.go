package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// --- Factory tests ---

func TestFactoryDefaultsToAnthropic(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "env-anthropic")
	provider, err := NewFactory(Config{Model: "claude-sonnet-4-5"})
	if err != nil {
		t.Fatalf("NewFactory returned error: %v", err)
	}
	if provider.GetMetadata().Provider != "anthropic" {
		t.Fatalf("expected anthropic provider, got %q", provider.GetMetadata().Provider)
	}
}

func TestFactoryValidatesCredentials(t *testing.T) {
	for _, k := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "OPENAI_COMPATIBLE_API_KEY", "API_KEY"} {
		_ = os.Unsetenv(k)
	}

	_, err := NewFactory(Config{Provider: "anthropic"})
	if err == nil {
		t.Fatalf("expected error for missing anthropic credentials")
	}
	if !strings.Contains(err.Error(), "missing anthropic credentials") {
		t.Errorf("unexpected error: %v", err)
	}

	_, err = NewFactory(Config{Provider: "openai-compatible"})
	if err == nil {
		t.Fatalf("expected error for missing openai-compatible credentials")
	}
	if !strings.Contains(err.Error(), "missing openai-compatible credentials") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFactoryPiCLINoCredentialsNeeded(t *testing.T) {
	p, err := NewFactory(Config{Provider: "pi-cli", Model: "qwen3.5:9b"})
	if err != nil {
		t.Fatalf("expected no error for pi-cli, got: %v", err)
	}
	if p.GetMetadata().Provider != "pi-cli" {
		t.Errorf("expected pi-cli provider, got %q", p.GetMetadata().Provider)
	}
}

func TestFactoryOllamaAlias(t *testing.T) {
	p, err := NewFactory(Config{Provider: "ollama", Model: "qwen3.5:9b"})
	if err != nil {
		t.Fatalf("expected no error for ollama alias, got: %v", err)
	}
	if p.GetMetadata().Provider != "pi-cli" {
		t.Errorf("expected pi-cli provider, got %q", p.GetMetadata().Provider)
	}
}

func TestFactoryUnsupportedProvider(t *testing.T) {
	_, err := NewFactory(Config{Provider: "azure-custom-thing"})
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFactoryOpenAICompatibleWithAPIKeyFlag(t *testing.T) {
	// All env vars unset, but API key passed via config
	for _, k := range []string{"OPENAI_API_KEY", "OPENAI_COMPATIBLE_API_KEY", "API_KEY"} {
		_ = os.Unsetenv(k)
	}
	p, err := NewFactory(Config{Provider: "openai-compatible", APIKey: "flag-key"})
	if err != nil {
		t.Fatalf("expected success with API key in config, got: %v", err)
	}
	if p.GetMetadata().Provider != "openai-compatible" {
		t.Errorf("expected openai-compatible, got %q", p.GetMetadata().Provider)
	}
}

func TestFactoryAnthropicWithAPIKeyFlag(t *testing.T) {
	_ = os.Unsetenv("ANTHROPIC_API_KEY")
	p, err := NewFactory(Config{Provider: "anthropic", APIKey: "flag-key"})
	if err != nil {
		t.Fatalf("expected success with API key in config, got: %v", err)
	}
	if p.GetMetadata().Provider != "anthropic" {
		t.Errorf("expected anthropic, got %q", p.GetMetadata().Provider)
	}
}

// --- normalizeProviderName tests ---

func TestNormalizeProviderName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "anthropic"},
		{"anthropic", "anthropic"},
		{"ANTHROPIC", "anthropic"},
		{"openai", "openai-compatible"},
		{"openai_compatible", "openai-compatible"},
		{"openrouter", "openai-compatible"},
		{"litellm", "openai-compatible"},
		{"  Anthropic  ", "anthropic"},
		{"pi", "pi-cli"},
		{"pi_cli", "pi-cli"},
		{"pi-cli", "pi-cli"},
		{"pi-ollama", "pi-cli"},
		{"ollama", "pi-cli"},
		{"custom-thing", "custom-thing"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeProviderName(tt.input)
			if got != tt.want {
				t.Errorf("normalizeProviderName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- firstNonEmpty tests ---

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"all empty", []string{"", "  ", ""}, ""},
		{"first wins", []string{"a", "b"}, "a"},
		{"second wins", []string{"", "b", "c"}, "b"},
		{"trims whitespace", []string{"  ", " x "}, "x"},
		{"no values", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstNonEmpty(tt.values...)
			if got != tt.want {
				t.Errorf("firstNonEmpty(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

// --- Anthropic provider tests ---

func TestAnthropicProviderGenerateAgentParsesResponse(t *testing.T) {
	t.Parallel()

	var gotAuth string
	var gotVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{{"type": "text", "text": "System prompt from anthropic"}},
			"usage":   map[string]any{"input_tokens": 100, "output_tokens": 40},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test-key", "claude-sonnet-4-5", server.URL)
	agent, meta, err := p.GenerateAgent(context.Background(), "customer support", []string{"be brief"})
	if err != nil {
		t.Fatalf("GenerateAgent returned error: %v", err)
	}

	if gotAuth != "test-key" {
		t.Fatalf("expected x-api-key header, got %q", gotAuth)
	}
	if gotVersion == "" {
		t.Fatalf("expected anthropic-version header")
	}
	if agent.SystemPrompt != "System prompt from anthropic" {
		t.Fatalf("unexpected system prompt: %q", agent.SystemPrompt)
	}
	if meta.TokensUsed != 140 {
		t.Fatalf("expected 140 tokens used, got %d", meta.TokensUsed)
	}
	if meta.CostUSD <= 0 {
		t.Fatalf("expected positive cost, got %f", meta.CostUSD)
	}
}

func TestAnthropicProviderAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "rate limit exceeded"},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test-key", "claude-sonnet-4-5", server.URL)
	_, _, err := p.GenerateAgent(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestAnthropicProviderAPIErrorNoMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test-key", "claude-sonnet-4-5", server.URL)
	_, _, err := p.GenerateAgent(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 error, got: %v", err)
	}
}

func TestAnthropicProviderEmptyContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{},
			"usage":   map[string]any{"input_tokens": 10, "output_tokens": 0},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test-key", "claude-sonnet-4-5", server.URL)
	_, _, err := p.GenerateAgent(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
	if !strings.Contains(err.Error(), "missing content") {
		t.Errorf("expected missing content error, got: %v", err)
	}
}

func TestAnthropicProviderExecuteAgent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.System == "" {
			t.Error("expected non-empty system prompt in request")
		}
		if req.MaxTokens != 2048 {
			t.Errorf("expected max_tokens=2048, got %d", req.MaxTokens)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{{"type": "text", "text": "execution response"}},
			"usage":   map[string]any{"input_tokens": 50, "output_tokens": 20},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test-key", "claude-sonnet-4-5", server.URL)
	out, meta, err := p.ExecuteAgent(context.Background(), AgentDefinition{
		SystemPrompt: "You are a helpful agent",
		Model:        "claude-sonnet-4-5",
		MaxTokens:    2048,
	}, "hello")
	if err != nil {
		t.Fatalf("ExecuteAgent returned error: %v", err)
	}
	if out != "execution response" {
		t.Errorf("unexpected output: %q", out)
	}
	if meta.TokensUsed != 70 {
		t.Errorf("expected 70 tokens, got %d", meta.TokensUsed)
	}
}

func TestAnthropicProviderExecuteAgentDefaultMaxTokens(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.MaxTokens != 1024 {
			t.Errorf("expected default max_tokens=1024, got %d", req.MaxTokens)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{{"type": "text", "text": "ok"}},
			"usage":   map[string]any{"input_tokens": 10, "output_tokens": 5},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("test-key", "claude-sonnet-4-5", server.URL)
	_, _, err := p.ExecuteAgent(context.Background(), AgentDefinition{
		SystemPrompt: "test",
		MaxTokens:    0, // should default to 1024
	}, "hello")
	if err != nil {
		t.Fatalf("ExecuteAgent returned error: %v", err)
	}
}

func TestAnthropicProviderGetMetadata(t *testing.T) {
	p := NewAnthropicProvider("key", "claude-sonnet-4-5", "https://custom.api.com")
	info := p.GetMetadata()
	if info.Provider != "anthropic" {
		t.Errorf("Provider = %q, want %q", info.Provider, "anthropic")
	}
	if info.Model != "claude-sonnet-4-5" {
		t.Errorf("Model = %q, want %q", info.Model, "claude-sonnet-4-5")
	}
	if info.BaseURL != "https://custom.api.com" {
		t.Errorf("BaseURL = %q, want %q", info.BaseURL, "https://custom.api.com")
	}
}

func TestAnthropicProviderDefaultModel(t *testing.T) {
	p := NewAnthropicProvider("key", "", "")
	if p.model != "claude-sonnet-4-5" {
		t.Errorf("default model = %q, want %q", p.model, "claude-sonnet-4-5")
	}
}

func TestAnthropicProviderDefaultBaseURL(t *testing.T) {
	p := NewAnthropicProvider("key", "model", "")
	if p.baseURL != "https://api.anthropic.com" {
		t.Errorf("default baseURL = %q, want %q", p.baseURL, "https://api.anthropic.com")
	}
}

func TestAnthropicProviderTrimsTrailingSlash(t *testing.T) {
	p := NewAnthropicProvider("key", "model", "https://api.example.com/")
	if p.baseURL != "https://api.example.com" {
		t.Errorf("baseURL = %q, expected trailing slash trimmed", p.baseURL)
	}
}

func TestAnthropicProviderUnknownModelCostZero(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{{"type": "text", "text": "ok"}},
			"usage":   map[string]any{"input_tokens": 100, "output_tokens": 50},
		})
	}))
	defer server.Close()

	p := NewAnthropicProvider("key", "unknown-model-xyz", server.URL)
	_, meta, err := p.GenerateAgent(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.CostUSD != 0 {
		t.Errorf("expected 0 cost for unknown model, got %f", meta.CostUSD)
	}
}

func TestAnthropicProviderContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server — but the context should cancel before we respond
		select {}
	}))
	defer server.Close()

	p := NewAnthropicProvider("key", "model", server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, _, err := p.GenerateAgent(ctx, "test", nil)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- OpenAI-compatible provider tests ---

func TestOpenAICompatibleProviderExecuteAgentParsesResponse(t *testing.T) {
	t.Parallel()

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "execution output"}}},
			"usage":   map[string]any{"prompt_tokens": 25, "completion_tokens": 15, "total_tokens": 40},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("open-key", "gpt-4o-mini", server.URL)
	out, meta, err := p.ExecuteAgent(context.Background(), AgentDefinition{SystemPrompt: "you are helpful"}, "hello")
	if err != nil {
		t.Fatalf("ExecuteAgent returned error: %v", err)
	}

	if gotAuth != "Bearer open-key" {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if out != "execution output" {
		t.Fatalf("unexpected output: %q", out)
	}
	if meta.TokensUsed != 40 {
		t.Fatalf("expected 40 total tokens, got %d", meta.TokensUsed)
	}
	if meta.DurationMs < 0 {
		t.Fatalf("expected non-negative duration, got %d", meta.DurationMs)
	}
}

func TestOpenAICompatibleProviderAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "invalid api key"},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("bad-key", "gpt-4o-mini", server.URL)
	_, _, err := p.GenerateAgent(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("expected auth error, got: %v", err)
	}
}

func TestOpenAICompatibleProviderEmptyChoices(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 0},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("key", "gpt-4o-mini", server.URL)
	_, _, err := p.GenerateAgent(context.Background(), "test", nil)
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
	if !strings.Contains(err.Error(), "missing choices") {
		t.Errorf("expected missing choices error, got: %v", err)
	}
}

func TestOpenAICompatibleProviderDefaultModel(t *testing.T) {
	p := NewOpenAICompatibleProvider("key", "", "")
	if p.model != "gpt-4o-mini" {
		t.Errorf("default model = %q, want %q", p.model, "gpt-4o-mini")
	}
}

func TestOpenAICompatibleProviderDefaultBaseURL(t *testing.T) {
	p := NewOpenAICompatibleProvider("key", "model", "")
	if p.baseURL != "https://api.openai.com/v1" {
		t.Errorf("default baseURL = %q, want %q", p.baseURL, "https://api.openai.com/v1")
	}
}

func TestOpenAICompatibleProviderGetMetadata(t *testing.T) {
	p := NewOpenAICompatibleProvider("key", "custom-model", "https://custom.api.com")
	info := p.GetMetadata()
	if info.Provider != "openai-compatible" {
		t.Errorf("Provider = %q, want %q", info.Provider, "openai-compatible")
	}
	if info.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", info.Model, "custom-model")
	}
}

func TestOpenAICompatibleProviderSystemPromptSentWhenPresent(t *testing.T) {
	t.Parallel()

	var receivedMessages int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedMessages = len(req.Messages)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("key", "gpt-4o-mini", server.URL)
	_, _, err := p.ExecuteAgent(context.Background(), AgentDefinition{SystemPrompt: "be helpful"}, "hello")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if receivedMessages != 2 { // system + user
		t.Errorf("expected 2 messages (system+user), got %d", receivedMessages)
	}
}

func TestOpenAICompatibleProviderNoSystemPrompt(t *testing.T) {
	t.Parallel()

	var receivedMessages int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		receivedMessages = len(req.Messages)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "generated"}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("key", "gpt-4o-mini", server.URL)
	_, _, err := p.GenerateAgent(context.Background(), "test need", nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if receivedMessages != 1 { // only user
		t.Errorf("expected 1 message (user only), got %d", receivedMessages)
	}
}

// --- completionsURL tests ---

func TestCompletionsURL(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"https://openrouter.ai/api", "https://openrouter.ai/api/chat/completions"},
	}
	for _, tt := range tests {
		t.Run(tt.baseURL, func(t *testing.T) {
			p := NewOpenAICompatibleProvider("key", "model", tt.baseURL)
			got := p.completionsURL()
			if got != tt.want {
				t.Errorf("completionsURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenAICompatibleProviderUnknownModelCostZero(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 100, "completion_tokens": 50, "total_tokens": 150},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("key", "unknown-model", server.URL)
	_, meta, err := p.GenerateAgent(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.CostUSD != 0 {
		t.Errorf("expected 0 cost for unknown model, got %f", meta.CostUSD)
	}
}

// --- Pi CLI provider tests ---

func TestPiCLIProviderDefaultModel(t *testing.T) {
	p := NewPiCLIProvider("", "", "")
	if p.model != "qwen3.5:9b" {
		t.Errorf("default model = %q, want %q", p.model, "qwen3.5:9b")
	}
}

func TestPiCLIProviderDefaultBinary(t *testing.T) {
	p := NewPiCLIProvider("model", "", "")
	if p.binary != "pi" {
		t.Errorf("default binary = %q, want %q", p.binary, "pi")
	}
}

func TestPiCLIProviderDefaultBaseURL(t *testing.T) {
	p := NewPiCLIProvider("model", "pi", "")
	if p.baseURL != "http://localhost:11434" {
		t.Errorf("default baseURL = %q, want %q", p.baseURL, "http://localhost:11434")
	}
}

func TestPiCLIProviderTrimsTrailingSlash(t *testing.T) {
	p := NewPiCLIProvider("model", "pi", "http://localhost:11434/")
	if p.baseURL != "http://localhost:11434" {
		t.Errorf("baseURL = %q, expected trailing slash trimmed", p.baseURL)
	}
}

func TestPiCLIProviderGetMetadata(t *testing.T) {
	p := NewPiCLIProvider("qwen3.5:35b", "pi", "http://localhost:11434")
	info := p.GetMetadata()
	if info.Provider != "pi-cli" {
		t.Errorf("Provider = %q, want %q", info.Provider, "pi-cli")
	}
	if info.Model != "qwen3.5:35b" {
		t.Errorf("Model = %q, want %q", info.Model, "qwen3.5:35b")
	}
	if info.BaseURL != "http://localhost:11434" {
		t.Errorf("BaseURL = %q, want %q", info.BaseURL, "http://localhost:11434")
	}
}

func TestPiCLIProviderResultTextPrefersResponse(t *testing.T) {
	p := NewPiCLIProvider("", "", "")
	r := &piJSONResult{Response: "from response", Result: "from result"}
	if got := p.resultText(r); got != "from response" {
		t.Errorf("resultText = %q, want %q", got, "from response")
	}
}

func TestPiCLIProviderResultTextFallsBackToResult(t *testing.T) {
	p := NewPiCLIProvider("", "", "")
	r := &piJSONResult{Response: "", Result: "from result"}
	if got := p.resultText(r); got != "from result" {
		t.Errorf("resultText = %q, want %q", got, "from result")
	}
}

func TestOpenAICompatibleProviderTotalTokensFallback(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "ok"}}},
			"usage":   map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 0},
		})
	}))
	defer server.Close()

	p := NewOpenAICompatibleProvider("key", "gpt-4o-mini", server.URL)
	_, meta, err := p.GenerateAgent(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// When total_tokens is 0, should fall back to input + output
	if meta.TokensUsed != 15 {
		t.Errorf("expected 15 tokens (10+5), got %d", meta.TokensUsed)
	}
}
