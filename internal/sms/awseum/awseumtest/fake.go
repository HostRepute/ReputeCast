// Package awseumtest is an in-memory AWS End User Messaging SMS (the
// pinpoint-sms-voice-v2 JSON protocol) and SNS (its query protocol), with
// the operations the adapter calls, failure modes, a signing certificate
// served where SNS serves its own, and helpers that publish SMS events and
// two-way texts as signed SNS notifications. Serve it in process with
// Client: every host the adapter calls (the APIs and the certificate) is
// answered here.
package awseumtest

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/sms/awseum"
	"github.com/hostrepute/reputecast/internal/sms/smsfake"
)

// The sample keys.
const (
	AccessKeyID     = "AKIAACMEINCSMS7Q2D"
	SecretAccessKey = "recorded-aws-secret-access-key"
	TopicARN        = "arn:aws:sns:us-east-1:123456789012:reputecast-sms"
	CertURL         = "https://sns.us-east-1.amazonaws.com/SimpleNotificationService-reputecast-fake.pem"
)

var (
	signingKey *rsa.PrivateKey
	certPEM    []byte
)

func init() {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "sns.amazonaws.com"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(365 * 24 * time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if err != nil {
		panic(err)
	}
	signingKey = k
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// Fake is the in-memory account.
type Fake struct {
	mu        sync.Mutex
	rec       smsfake.Recorder
	Mode      smsfake.Mode
	Rejected  bool
	OptedOut  map[string]time.Time
	Confirmed bool
	TwoWay    map[string]string
	seq       int
}

// New returns the sample account.
func New() *Fake {
	return &Fake{OptedOut: map[string]time.Time{"+13125550111": time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)}, TwoWay: map[string]string{}}
}

// Handler serves the fake.
func (f *Fake) Handler() http.Handler { return http.HandlerFunc(f.serve) }

// Client serves the fake in process.
func (f *Fake) Client() *http.Client { return smsfake.Client(f.Handler()) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func awsErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	writeJSON(w, status, map[string]any{"__type": code, "message": msg})
}

func (f *Fake) serve(w http.ResponseWriter, r *http.Request) {
	body := f.rec.Record(r)
	if strings.HasSuffix(r.URL.Path, ".pem") {
		w.Header().Set("Content-Type", "application/x-pem-file")
		_, _ = w.Write(certPEM)
		return
	}
	if strings.Contains(r.URL.RawQuery, "Action=ConfirmSubscription") || strings.HasSuffix(r.URL.Path, "/confirm") {
		f.mu.Lock()
		f.Confirmed = true
		f.mu.Unlock()
		_, _ = w.Write([]byte("<ConfirmSubscriptionResponse/>"))
		return
	}
	switch f.Mode.Of() {
	case smsfake.ModeOutage:
		awsErr(w, 503, "ServiceUnavailableException", "Service unavailable")
		return
	case smsfake.ModeAuth:
		awsErr(w, 400, "UnrecognizedClientException", "The security token included in the request is invalid.")
		return
	}
	f.mu.Lock()
	rejected := f.Rejected
	f.mu.Unlock()
	if !strings.Contains(r.Header.Get("Authorization"), "Credential="+AccessKeyID+"/") || rejected {
		awsErr(w, 400, "UnrecognizedClientException", "The security token included in the request is invalid.")
		return
	}
	if strings.HasPrefix(r.URL.Path, "/sns") {
		f.serveSNS(w, body)
		return
	}
	var in map[string]any
	_ = json.Unmarshal(body, &in)
	op := strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "PinpointSMSVoiceV2.")
	now := float64(time.Date(2026, 10, 6, 14, 2, 0, 0, time.UTC).Unix())
	switch op {
	case "DescribeAccountAttributes":
		writeJSON(w, 200, map[string]any{"AccountAttributes": []any{map[string]any{"Name": "ACCOUNT_TIER", "Value": "PRODUCTION"}}})
	case "SendTextMessage":
		if fmt.Sprint(in["DestinationPhoneNumber"]) == smsfake.InvalidNumber {
			awsErr(w, 400, "ValidationException", "Invalid destination phone number")
			return
		}
		f.mu.Lock()
		f.seq++
		id := fmt.Sprintf("%08x-aaaa-4bbb-8ccc-%012d", f.seq, f.seq)
		_, out := f.OptedOut[fmt.Sprint(in["DestinationPhoneNumber"])]
		f.mu.Unlock()
		if out {
			awsErr(w, 400, "ConflictException", "DESTINATION_PHONE_NUMBER_OPTED_OUT")
			return
		}
		writeJSON(w, 200, map[string]any{"MessageId": id})
	case "DescribePhoneNumbers":
		writeJSON(w, 200, map[string]any{"PhoneNumbers": []any{
			map[string]any{"PhoneNumber": "+18885550160", "PhoneNumberId": "phone-0160", "PhoneNumberArn": "arn:aws:sms-voice:us-east-1:123456789012:phone-number/phone-0160",
				"IsoCountryCode": "US", "NumberType": "TOLL_FREE", "Status": "ACTIVE", "MessageType": "TRANSACTIONAL", "NumberCapabilities": []string{"SMS"},
				"MonthlyLeasingPrice": "2.00", "OptOutListName": "Default", "CreatedTimestamp": now, "RegistrationId": "registration-tf0160"},
		}})
	case "DescribeSenderIds":
		writeJSON(w, 200, map[string]any{"SenderIds": []any{map[string]any{"SenderId": "ACME", "SenderIdArn": "arn:aws:sms-voice:us-east-1:123456789012:sender-id/ACME/GB",
			"IsoCountryCode": "GB", "MessageTypes": []string{"TRANSACTIONAL"}, "MonthlyLeasingPrice": "0.00", "Registered": false}}})
	case "DescribeRegistrations":
		writeJSON(w, 200, map[string]any{"Registrations": []any{map[string]any{"RegistrationId": "registration-tf0160", "RegistrationArn": "arn:aws:sms-voice:us-east-1:123456789012:registration/registration-tf0160",
			"RegistrationType": "US_TOLL_FREE_REGISTRATION", "RegistrationStatus": "COMPLETE", "CurrentVersionNumber": 1, "CreatedTimestamp": now}}})
	case "DescribeOptedOutNumbers":
		f.mu.Lock()
		list := []any{}
		for n, at := range f.OptedOut {
			list = append(list, map[string]any{"OptedOutNumber": n, "OptedOutTimestamp": float64(at.Unix()), "EndUserOptedOut": true})
		}
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"OptOutListName": "Default", "OptOutListArn": "arn:aws:sms-voice:us-east-1:123456789012:opt-out-list/Default", "OptedOutNumbers": list})
	case "PutOptedOutNumber":
		n := fmt.Sprint(in["OptedOutNumber"])
		f.mu.Lock()
		f.OptedOut[n] = time.Now()
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"OptOutListName": "Default", "OptedOutNumber": n, "OptedOutTimestamp": now, "EndUserOptedOut": false})
	case "DeleteOptedOutNumber":
		n := fmt.Sprint(in["OptedOutNumber"])
		f.mu.Lock()
		delete(f.OptedOut, n)
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"OptOutListName": "Default", "OptedOutNumber": n, "OptedOutTimestamp": now, "EndUserOptedOut": false})
	case "CreateConfigurationSet":
		writeJSON(w, 200, map[string]any{"ConfigurationSetName": in["ConfigurationSetName"], "ConfigurationSetArn": "arn:aws:sms-voice:us-east-1:123456789012:configuration-set/reputecast-sms"})
	case "CreateEventDestination":
		writeJSON(w, 200, map[string]any{"ConfigurationSetName": in["ConfigurationSetName"], "EventDestination": map[string]any{"EventDestinationName": "reputecast", "Enabled": true,
			"MatchingEventTypes": []string{"ALL"}, "SnsDestination": in["SnsDestination"]}})
	case "UpdatePhoneNumber":
		f.mu.Lock()
		f.TwoWay[fmt.Sprint(in["PhoneNumberId"])] = fmt.Sprint(in["TwoWayChannelArn"])
		f.mu.Unlock()
		writeJSON(w, 200, map[string]any{"PhoneNumberId": in["PhoneNumberId"], "TwoWayEnabled": true})
	default:
		awsErr(w, 400, "UnknownOperationException", "Unknown operation "+op)
	}
}

func (f *Fake) serveSNS(w http.ResponseWriter, body []byte) {
	v, _ := url.ParseQuery(string(body))
	w.Header().Set("Content-Type", "text/xml")
	switch v.Get("Action") {
	case "CreateTopic":
		fmt.Fprintf(w, `<CreateTopicResponse xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><CreateTopicResult><TopicArn>%s</TopicArn></CreateTopicResult><ResponseMetadata><RequestId>1</RequestId></ResponseMetadata></CreateTopicResponse>`, TopicARN)
	case "Subscribe":
		fmt.Fprint(w, `<SubscribeResponse xmlns="http://sns.amazonaws.com/doc/2010-03-31/"><SubscribeResult><SubscriptionArn>pending confirmation</SubscriptionArn></SubscribeResult><ResponseMetadata><RequestId>2</RequestId></ResponseMetadata></SubscribeResponse>`)
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `<ErrorResponse><Error><Code>InvalidAction</Code><Message>unknown</Message></Error></ErrorResponse>`)
	}
}

// sign signs an envelope as SNS does (SignatureVersion 2).
func sign(e awseum.Envelope) awseum.Envelope {
	e.SignatureVersion, e.SigningCertURL = "2", CertURL
	sum := sha256.Sum256([]byte(e.StringToSign()))
	sig, err := rsa.SignPKCS1v15(rand.Reader, signingKey, crypto.SHA256, sum[:])
	if err != nil {
		panic(err)
	}
	e.Signature = base64.StdEncoding.EncodeToString(sig)
	return e
}

func post(hookURL string, e awseum.Envelope) (*http.Request, []byte) {
	body, _ := json.Marshal(e)
	r, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, hookURL, strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "text/plain; charset=UTF-8")
	r.Header.Set("X-Amz-Sns-Message-Type", e.Type)
	r.Header.Set("X-Amz-Sns-Topic-Arn", e.TopicArn)
	return r, body
}

// Notification publishes message on the topic, signed.
func Notification(hookURL string, message any, at time.Time) (*http.Request, []byte) {
	m, _ := json.Marshal(message)
	return post(hookURL, sign(awseum.Envelope{Type: "Notification", MessageID: fmt.Sprintf("sns-%d", at.UnixNano()), TopicArn: TopicARN, Message: string(m),
		Timestamp: at.UTC().Format("2006-01-02T15:04:05.000Z")}))
}

// SubscriptionConfirmation is SNS asking to confirm ReputeCast's
// subscription; its link is answered by this fake.
func SubscriptionConfirmation(hookURL string, at time.Time) (*http.Request, []byte) {
	return post(hookURL, sign(awseum.Envelope{Type: "SubscriptionConfirmation", MessageID: "sub-1", Token: "tok", TopicArn: TopicARN,
		Message: "You have chosen to subscribe to the topic.", SubscribeURL: "https://sns.us-east-1.amazonaws.com/confirm?Action=ConfirmSubscription&Token=tok",
		Timestamp: at.UTC().Format("2006-01-02T15:04:05.000Z")}))
}

// Event is an SMS event for a text.
func Event(hookURL, eventType, messageID, to, from, reputecastID string, at time.Time) (*http.Request, []byte) {
	return Notification(hookURL, map[string]any{"eventType": eventType, "eventVersion": "1.0", "eventTimestamp": at.UnixMilli(), "isFinal": true,
		"originationPhoneNumber": from, "destinationPhoneNumber": to, "isoCountryCode": "US", "messageId": messageID, "messageEncoding": "GSM",
		"messageType": "PROMOTIONAL", "messageStatus": strings.TrimPrefix(eventType, "TEXT_"), "totalMessageParts": 1, "totalMessagePrice": 0.00581,
		"context": map[string]string{"reputecast_message": reputecastID}}, at)
}

// Inbound is a two-way text.
func Inbound(hookURL, id, from, to, body string, at time.Time) (*http.Request, []byte) {
	return Notification(hookURL, map[string]any{"originationNumber": from, "destinationNumber": to, "messageKeyword": "KEYWORD_123456789012", "messageBody": body,
		"inboundMessageId": id}, at)
}
