package agent

import "testing"

func TestScrubCredentials(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "token with equals",
			input: "TOKEN=abcdef123456",
			want:  "TOKEN=abcd****",
		},
		{
			name:  "api_key with colon",
			input: "api_key: sk-proj-abc123",
			want:  "api_key: sk-p****",
		},
		{
			name:  "password with equals",
			input: "password=hunter2",
			want:  "password=hunt****",
		},
		{
			name:  "secret with colon space",
			input: "SECRET: mysecretvalue",
			want:  "SECRET: myse****",
		},
		{
			name:  "authorization header",
			input: "Authorization=Bearer-eyJhbGciOi",
			want:  "Authorization=Bear****",
		},
		{
			name:  "plain text unchanged",
			input: "this is just normal output",
			want:  "this is just normal output",
		},
		{
			name:  "short value",
			input: "TOKEN=abc",
			want:  "TOKEN=****",
		},
		{
			name:  "multiple matches",
			input: "TOKEN=abcdef SECRET=xyz123",
			want:  "TOKEN=abcd**** SECRET=xyz1****",
		},
		{
			name:  "api-key with dash",
			input: "api-key=longvalue123",
			want:  "api-key=long****",
		},
		{
			name:  "apikey no separator",
			input: "APIKEY=myval",
			want:  "APIKEY=myva****",
		},
		{
			name:  "case insensitive",
			input: "Api_Key = test1234",
			want:  "Api_Key = test****",
		},
		{
			name:  "bearer token",
			input: "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9",
			want:  "Authorization: Bear**** eyJh****",
		},
		{
			name:  "bearer lowercase",
			input: "authorization=bearer eyJhbGciOiJ",
			want:  "authorization=bear**** eyJh****",
		},
		{
			name:  "standalone bearer token",
			input: "Bearer eyJhbGciOiJIUzI1NiJ9",
			want:  "Bearer eyJh****",
		},
		{
			name:  "short bearer token",
			input: "Bearer abcd",
			want:  "Bearer ****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScrubCredentials(tt.input)
			if got != tt.want {
				t.Errorf("ScrubCredentials(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
