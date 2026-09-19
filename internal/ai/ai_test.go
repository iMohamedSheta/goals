package ai

import "testing"

func TestParseRunJSON_Text(t *testing.T) {
	raw := []byte("{\"type\":\"step_start\",\"timestamp\":1,\"sessionID\":\"ses_abc\",\"part\":{\"type\":\"step-start\"}}\n" +
		"{\"type\":\"text\",\"timestamp\":1,\"sessionID\":\"ses_abc\",\"part\":{\"type\":\"text\",\"text\":\"HI\"}}\n" +
		"{\"type\":\"step_finish\",\"timestamp\":1,\"sessionID\":\"ses_abc\",\"part\":{\"type\":\"step-finish\"}}\n")
	reply, sid, apiErr := parseRunJSON(raw)
	if reply != "HI" {
		t.Fatalf("reply = %q, want HI", reply)
	}
	if sid != "ses_abc" {
		t.Fatalf("sessionID = %q, want ses_abc", sid)
	}
	if apiErr != "" {
		t.Fatalf("apiErr = %q, want empty", apiErr)
	}
}

func TestParseRunJSON_Error(t *testing.T) {
	raw := []byte("{\"type\":\"error\",\"timestamp\":1,\"sessionID\":\"ses_x\",\"error\":{\"name\":\"APIError\",\"data\":{\"message\":\"No payment method.\"}}}\n")
	_, sid, apiErr := parseRunJSON(raw)
	if sid != "ses_x" {
		t.Fatalf("sessionID = %q, want ses_x", sid)
	}
	if apiErr != "No payment method." {
		t.Fatalf("apiErr = %q", apiErr)
	}
}

func TestParseRunJSON_MultiText(t *testing.T) {
	raw := []byte("{\"type\":\"text\",\"sessionID\":\"s\",\"part\":{\"type\":\"text\",\"text\":\"Hel\"}}\n" +
		"{\"type\":\"text\",\"sessionID\":\"s\",\"part\":{\"type\":\"text\",\"text\":\"lo\"}}\n")
	reply, _, _ := parseRunJSON(raw)
	if reply != "Hello" {
		t.Fatalf("reply = %q, want Hello", reply)
	}
}
