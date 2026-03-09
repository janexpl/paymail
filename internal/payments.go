package internal

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net/smtp"
	"sort"
	"strings"
	"sync"
)

const overduePaymentsQuery = "SELECT [kh_Id]," +
	"[adr_Nazwa], " +
	"[nzf_NumerPelny], " +
	"[naleznosc], " +
	"[nzf_WartoscPierwotna], " +
	"[spoznienie], " +
	"[nzf_Wystawil] " +
	"FROM [Nasza_Era_Sp__z_o_o_].[dbo].[vwFinanseRozrachunkiWgKontrahentow] " +
	"left outer join [Nasza_Era_Sp__z_o_o_].dbo.fl_Wartosc on nzf_Id=flw_IdObiektu " +
	"join [Nasza_Era_Sp__z_o_o_].dbo.vwKlienci on nzf_IdObiektu=kh_Id " +
	"join [Nasza_Era_Sp__z_o_o_].dbo.pd_Uzytkownik " +
	"on nzf_IdWystawil=uz_Id " +
	"WHERE nzf_Typ=39 and naleznosc<>0 and spoznienie>0 and flw_IdFlagi=1000 and dok_Typ=2 "

var emailTemplate = template.Must(template.New("overdue-report").Parse(`<!DOCTYPE html>
<html>
<body>
<p>Witaj {{ .EmployeeName }},</p>
<p>Zestawienie zaleglych faktur dla {{ .CompanyName }}.</p>
<table border="1" cellspacing="0" cellpadding="4">
<tr>
<th>Nazwa firmy</th>
<th>Numer dokumentu</th>
<th>Dni spoznienia</th>
<th>Wartosc</th>
<th>Do zaplaty</th>
</tr>
{{ range .Payments }}
<tr>
<td><b>{{ .CompanyName }}</b></td>
<td>{{ .DocNo }}</td>
<td>{{ .DaysLate }}</td>
<td>{{ printf "%.2f" .Amount }}</td>
<td>{{ printf "%.2f" .ToPay }}</td>
</tr>
{{ end }}
<tr>
<td></td>
<td></td>
<td></td>
<td><b>RAZEM</b></td>
<td><b>{{ printf "%.2f" .Total }}</b></td>
</tr>
</table>
<p>Prosze o sprawdzenie zaleglosci i windykacje.</p>
<p>{{ .Signature }}</p>
</body>
</html>
`))

// Payment describes a single overdue invoice row.
type Payment struct {
	CompanyID    int64
	CompanyName  string
	DocNo        string
	DaysLate     int
	Amount       float64
	ToPay        float64
	EmployeeName string
}

// PaymentsController handles fetching payments and sending emails.
type PaymentsController struct {
	db         *sql.DB
	employees  *EmployeeDirectory
	config     *Config
	smtpSender func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error
}

type emailViewData struct {
	EmployeeName string
	CompanyName  string
	Signature    string
	Payments     []Payment
	Total        float64
}

// NewPaymentsController builds the payments service.
func NewPaymentsController(conn *sql.DB, employees *EmployeeDirectory, config *Config) *PaymentsController {
	return &PaymentsController{
		db:         conn,
		employees:  employees,
		config:     config,
		smtpSender: smtp.SendMail,
	}
}

// FetchPayments loads all overdue payments in one query.
func (p *PaymentsController) FetchPayments(ctx context.Context) ([]Payment, error) {
	rows, err := p.db.QueryContext(ctx, overduePaymentsQuery)
	if err != nil {
		return nil, fmt.Errorf("query overdue payments: %w", err)
	}
	defer rows.Close()

	payments := make([]Payment, 0)
	for rows.Next() {
		payment := Payment{}
		if err := rows.Scan(
			&payment.CompanyID,
			&payment.CompanyName,
			&payment.DocNo,
			&payment.ToPay,
			&payment.Amount,
			&payment.DaysLate,
			&payment.EmployeeName,
		); err != nil {
			return nil, fmt.Errorf("scan overdue payment: %w", err)
		}

		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate overdue payments: %w", err)
	}

	return payments, nil
}

// SendOverduePaymentEmails groups data once and sends one email per employee.
func (p *PaymentsController) SendOverduePaymentEmails(ctx context.Context, workers int) error {
	if workers < 1 {
		workers = 1
	}

	payments, err := p.FetchPayments(ctx)
	if err != nil {
		return err
	}

	grouped := groupPaymentsByEmployee(payments)
	if len(grouped) == 0 {
		return nil
	}

	sem := make(chan struct{}, workers)
	errCh := make(chan error, len(grouped))
	var wg sync.WaitGroup

	for employeeName, employeePayments := range grouped {
		email, ok := p.employees.EmailByName(employeeName)
		if !ok {
			errCh <- fmt.Errorf("missing email for employee %q", employeeName)
			continue
		}

		wg.Add(1)
		go func(name, to string, payments []Payment) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
			defer func() {
				<-sem
			}()

			if err := p.sendEmail(ctx, name, to, payments); err != nil {
				errCh <- fmt.Errorf("send email for %q: %w", name, err)
			}
		}(employeeName, email, employeePayments)
	}

	wg.Wait()
	close(errCh)

	return collectErrors(errCh)
}

func groupPaymentsByEmployee(payments []Payment) map[string][]Payment {
	grouped := make(map[string][]Payment)
	for _, payment := range payments {
		name := strings.TrimSpace(payment.EmployeeName)
		if name == "" {
			name = "<unknown>"
		}

		grouped[name] = append(grouped[name], payment)
	}

	return grouped
}

func (p *PaymentsController) sendEmail(ctx context.Context, employeeName, to string, payments []Payment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("empty recipient address")
	}

	message, err := p.buildEmailMessage(employeeName, to, payments)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", p.config.Email.Hostname, p.config.Email.Port)
	auth := smtp.PlainAuth("", p.config.Email.Username, p.config.Email.Password, p.config.Email.Hostname)
	if err := p.smtpSender(addr, auth, p.config.Email.Username, []string{to}, message); err != nil {
		return err
	}

	return nil
}

func (p *PaymentsController) buildEmailMessage(employeeName, to string, payments []Payment) ([]byte, error) {
	body, err := p.renderEmailBody(employeeName, payments)
	if err != nil {
		return nil, err
	}

	subject := mime.QEncoding.Encode("UTF-8", p.config.Report.Subject)
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("To: %s\r\n", to))
	msg.WriteString(fmt.Sprintf("From: %s\r\n", p.config.Email.Username))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	msg.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.Bytes(), nil
}

func (p *PaymentsController) renderEmailBody(employeeName string, payments []Payment) (string, error) {
	viewData := emailViewData{
		EmployeeName: employeeName,
		CompanyName:  p.config.Report.CompanyName,
		Signature:    p.config.Report.Signature,
		Payments:     append([]Payment(nil), payments...),
	}

	sort.Slice(viewData.Payments, func(i, j int) bool {
		if viewData.Payments[i].CompanyName == viewData.Payments[j].CompanyName {
			return viewData.Payments[i].DocNo < viewData.Payments[j].DocNo
		}
		return viewData.Payments[i].CompanyName < viewData.Payments[j].CompanyName
	})

	for _, payment := range viewData.Payments {
		viewData.Total += payment.ToPay
	}

	var body bytes.Buffer
	if err := emailTemplate.Execute(&body, viewData); err != nil {
		return "", fmt.Errorf("render email body: %w", err)
	}

	return body.String(), nil
}

func collectErrors(errCh <-chan error) error {
	messages := make([]string, 0)
	for err := range errCh {
		if err == nil {
			continue
		}
		messages = append(messages, err.Error())
	}

	if len(messages) == 0 {
		return nil
	}

	sort.Strings(messages)
	return errors.New(strings.Join(messages, "; "))
}
