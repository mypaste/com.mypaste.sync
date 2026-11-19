package s3

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestSignMatchesAWSVector is the published SigV4 GET-object example from the
// S3 documentation. Matching it byte for byte proves the canonical request,
// string to sign and key derivation, not just self-consistency.
func TestSignMatchesAWSVector(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet,
		"https://examplebucket.s3.amazonaws.com/test.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-9")
	access := "AKIAIOSFODNN7EXAMPLE"
	secret := "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	config := Config{Region: "us-east-1", AccessKeyID: access, SecretAccessKey: secret}
	signer := &Signer{Now: func() time.Time {
		return time.Date(2013, time.May, 24, 0, 0, 0, 0, time.UTC)
	}}
	signer.Sign(req, config, EmptyPayloadHash)
	scope := "20130524/us-east-1/s3/aws4_request"
	want := "AWS4-HMAC-SHA256 Credential=" + access + "/" + scope +
		", SignedHeaders=host;range;x-amz-content-sha256;x-amz-date" +
		", Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("authorization:\n got %s\nwant %s", got, want)
	}
}

// The signature is defined on UTC: Format on a local time prints the local
// wall clock and appends a literal Z, which every endpoint rejects as a skewed
// clock. The test's zone is +08:00, so a regression shows up as 16:44:43Z.
func TestSignFormatsTheDateInUTC(t *testing.T) {
	zone := time.FixedZone("CST", 8*60*60)
	req, err := http.NewRequest(http.MethodGet, "https://example.com/bucket", nil)
	if err != nil {
		t.Fatal(err)
	}
	config := Config{Region: "auto", AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "secret"}
	signer := &Signer{Now: func() time.Time {
		return time.Date(2026, time.November, 19, 16, 44, 43, 0, zone)
	}}
	signer.Sign(req, config, EmptyPayloadHash)
	if got := req.Header.Get("X-Amz-Date"); got != "20261119T084443Z" {
		t.Fatalf("x-amz-date = %q, want the UTC instant", got)
	}
	if got := req.Header.Get("Authorization"); !strings.Contains(got, "/20261119/auto/s3/aws4_request") {
		t.Fatalf("credential scope = %q", got)
	}
}

// ErrorCode is what turns a bare 403 into an actionable message.
func TestErrorCodeReadsTheXMLCode(t *testing.T) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
		`<Error><Code>RequestTimeTooSkewed</Code><Message>skew</Message></Error>`)
	if got := ErrorCode(body); got != "RequestTimeTooSkewed" {
		t.Fatalf("code = %q", got)
	}
	for _, bad := range []string{
		"",
		"not xml at all",
		`<Error><Code>bad code</Code></Error>`,
		`<Error><Code>` + strings.Repeat("a", 65) + `</Code></Error>`,
		`<Error><Code>&lt;script&gt;</Code></Error>`,
	} {
		if got := ErrorCode([]byte(bad)); got != "" {
			t.Fatalf("%q parsed as %q", bad, got)
		}
	}
}

func TestObjectURLKeepsBucketPathAndPrefix(t *testing.T) {
	config := Config{Bucket: "my-bucket", Prefix: "/snap shots/"}
	object, err := config.ObjectURL("20260101-000000-backup.zip")
	if err != nil {
		t.Fatal(err)
	}
	if object.Scheme != "https" || object.Host != "s3.amazonaws.com" {
		t.Fatalf("endpoint: %s", object)
	}
	if object.Path != "/my-bucket/snap shots/20260101-000000-backup.zip" {
		t.Fatalf("path: %q", object.Path)
	}
	if object.EscapedPath() != "/my-bucket/snap%20shots/20260101-000000-backup.zip" {
		t.Fatalf("escaped path: %q", object.EscapedPath())
	}
}

func TestObjectURLCustomEndpointBasePath(t *testing.T) {
	config := Config{Endpoint: "https://storage.example.com/s3", Bucket: "b", Prefix: "p"}
	object, err := config.ObjectURL("a.zip")
	if err != nil {
		t.Fatal(err)
	}
	if object.String() != "https://storage.example.com/s3/b/p/a.zip" {
		t.Fatalf("url: %s", object.String())
	}
}

func TestListURLQuery(t *testing.T) {
	config := Config{Bucket: "b", Prefix: "p"}
	list, err := config.ListURL(50, "token+/=")
	if err != nil {
		t.Fatal(err)
	}
	want := "continuation-token=" + url.QueryEscape("token+/=") +
		"&list-type=2&max-keys=50&prefix=p%2F"
	if list.RawQuery != want {
		t.Fatalf("query:\n got %s\nwant %s", list.RawQuery, want)
	}
	if list.Path != "/b" {
		t.Fatalf("path: %q", list.Path)
	}
}

func TestObjectNameOrdering(t *testing.T) {
	now := time.Date(2026, time.November, 10, 9, 8, 7, 0, time.UTC)
	if got := ObjectName("/tmp/mypaste-backup.zip", now); got !=
		"20261110-090807-mypaste-backup.zip.mypaste-sync" {
		t.Fatalf("name: %q", got)
	}
}

func TestParseList(t *testing.T) {
	body := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <IsTruncated>true</IsTruncated>
  <NextContinuationToken>next+/=</NextContinuationToken>
  <Contents>
    <Key>p/20260101-000000-a.mypaste-sync</Key>
    <LastModified>2026-01-01T00:00:00.000Z</LastModified>
    <Size>42</Size>
  </Contents>
  <Contents>
    <Key>p/20260102-000000-b.mypaste-sync</Key>
    <LastModified>2026-01-02T00:00:00.000Z</LastModified>
    <Size>7</Size>
  </Contents>
</ListBucketResult>`
	entries, token, err := ParseList([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || token != "next+/=" {
		t.Fatalf("entries=%d token=%q", len(entries), token)
	}
	latest, ok := Latest(entries)
	if !ok || latest.Key != "p/20260102-000000-b.mypaste-sync" || latest.Size != 7 {
		t.Fatalf("latest: %+v ok=%v", latest, ok)
	}
	if _, _, err := ParseList([]byte("<not-xml")); err == nil {
		t.Fatal("malformed list accepted")
	}
}

func TestLatestFallsBackToKeyTimestamp(t *testing.T) {
	entries := []ListEntry{
		{Key: "p/20260101-000000-a.mypaste-sync", Size: 1},
		{Key: "p/20260102-000000-b.mypaste-sync", Size: 1},
	}
	latest, ok := Latest(entries)
	if !ok || latest.Key != "p/20260102-000000-b.mypaste-sync" {
		t.Fatalf("latest: %+v", latest)
	}
	if _, ok := Latest(nil); ok {
		t.Fatal("empty list claimed an object")
	}
}

func TestValidateRejectsBadConnections(t *testing.T) {
	valid := Config{Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cases := []Config{
		{Bucket: "b/c", AccessKeyID: "k", SecretAccessKey: "s"},
		{Bucket: "", AccessKeyID: "k", SecretAccessKey: "s"},
		{Bucket: "b", AccessKeyID: "", SecretAccessKey: "s"},
		{Bucket: "b", AccessKeyID: "k", SecretAccessKey: ""},
		{Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s", Prefix: "../escape"},
		{Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s", Endpoint: "s3.example.com"},
	}
	for index, config := range cases {
		if err := config.Validate(); err == nil {
			t.Fatalf("case %d accepted", index)
		}
	}
}

func TestStatusError(t *testing.T) {
	message := StatusError("上传", 403).Error()
	if !strings.Contains(message, "403") || !strings.Contains(message, "Access Key") {
		t.Fatalf("message: %q", message)
	}
	if !strings.Contains(StatusError("下载", 404).Error(), "不存在") {
		t.Fatalf("404 message: %q", StatusError("下载", 404))
	}
}

func TestHeadersDropsHost(t *testing.T) {
	req := &http.Request{Header: http.Header{
		"Host": {"example.com"}, "X-Amz-Date": {"20260101T000000Z"},
	}}
	headers := Headers(req)
	if _, ok := headers["Host"]; ok {
		t.Fatal("host header leaked into a signed request")
	}
	if headers["X-Amz-Date"] != "20260101T000000Z" {
		t.Fatalf("headers: %v", headers)
	}
}
