package internal

import (
	"strings"
	"testing"
)

func TestGroupPaymentsByEmployee(t *testing.T) {
	payments := []Payment{
		{EmployeeName: "anna", DocNo: "A/1"},
		{EmployeeName: "bob", DocNo: "B/1"},
		{EmployeeName: "anna", DocNo: "A/2"},
	}

	grouped := groupPaymentsByEmployee(payments)
	if len(grouped["anna"]) != 2 {
		t.Fatalf("expected 2 payments for anna, got %d", len(grouped["anna"]))
	}
	if len(grouped["bob"]) != 1 {
		t.Fatalf("expected 1 payment for bob, got %d", len(grouped["bob"]))
	}
}

func TestBuildEmailMessageEscapesHTMLAndCalculatesTotal(t *testing.T) {
	controller := NewPaymentsController(nil, nil, &Config{
		Email: EmailConfig{
			Username: "robot@example.com",
		},
		Report: ReportConfig{
			Subject:     "Raport zaleglych faktur",
			CompanyName: "Test Company",
			Signature:   "Janusz",
		},
	})

	message, err := controller.buildEmailMessage("Anna", "anna@example.com", []Payment{
		{
			CompanyName: `<script>alert("xss")</script>`,
			DocNo:       "FV/1",
			DaysLate:    12,
			Amount:      120.55,
			ToPay:       100.10,
		},
		{
			CompanyName: "Beta",
			DocNo:       "FV/2",
			DaysLate:    3,
			Amount:      50,
			ToPay:       40.25,
		},
	})
	if err != nil {
		t.Fatalf("buildEmailMessage() returned error: %v", err)
	}

	body := string(message)
	if !strings.Contains(body, "140.35") {
		t.Fatalf("expected total in message body, got: %s", body)
	}
	if strings.Contains(body, `<script>alert("xss")</script>`) {
		t.Fatalf("expected HTML to be escaped, got: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;") {
		t.Fatalf("expected escaped HTML in message body, got: %s", body)
	}
}
