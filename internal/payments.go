package internal

import (
	"database/sql"
	"fmt"
	"log"
	"net/smtp"
	"sync"
)

// Payment structure
type Payment struct {
	ComanyID    int64
	CompanyName string
	DocNo       string
	DaysLate    int
	Amount      float32
	ToPay       float32
	EmployeeN   string
}

// Payments structure with array of payments
type Payments struct {
	Payments []Payment
}

// PaymentsController structure with pointers to config, database and employee list,
type PaymentsController struct {
	dBConnection  *sql.DB
	employessList *Employess
	config        *Config
}

// NewPayments constructor
func NewPayments(conn *sql.DB, emp *Employess, config *Config) *PaymentsController {

	return &PaymentsController{
		dBConnection:  conn,
		employessList: emp,
		config:        config,
	}
}

func (pCon *PaymentsController) getPayments() *Payments {
	stmt, err := pCon.dBConnection.Prepare("SELECT [kh_Id]," +
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
		"WHERE nzf_Typ=39 and naleznosc<>0 and spoznienie>0 and flw_IdFlagi=1000 and dok_Typ=2 ")
	if err != nil {
		log.Fatal("Prepare failed:", err.Error())
	}
	defer stmt.Close()
	rows, err := stmt.Query()
	if err != nil {
		log.Fatal("Query failed", err.Error())
	}
	payments := []Payment{}
	for rows.Next() {
		payment := Payment{}
		err = rows.Scan(&payment.ComanyID,
			&payment.CompanyName,
			&payment.DocNo,
			&payment.ToPay,
			&payment.Amount,
			&payment.DaysLate,
			&payment.EmployeeN)
		if err != nil {
			log.Fatal("Scan failed:", err.Error())
		}
		payments = append(payments, payment)
	}

	return &Payments{
		Payments: payments,
	}
}

// SendEmail method - sending email to employee with name
func (pCon *PaymentsController) SendEmail(name string, wg *sync.WaitGroup) {

	payments := pCon.getPayments() // Get all overdue payments
	paymentWithName := []Payment{}
	var sum float32
	for _, payment := range payments.Payments {
		if payment.EmployeeN == name {
			paymentWithName = append(paymentWithName, payment)
			sum += payment.ToPay
		}
	}
	if sum != 0 {
		// Creating table with clients and overdue payments
		var msgTable string
		for _, element := range paymentWithName {
			msg := fmt.Sprintf("<tr><td><b>%s</b></td><td>%s</td><td>%d</td><td>%.2f</td><td>%.2f</td></tr>\n",
				element.CompanyName, element.DocNo, element.DaysLate, element.Amount, element.ToPay)
			msgTable += msg
		}

		auth := smtp.PlainAuth("",
			pCon.config.Email.Username,
			pCon.config.Email.Password,
			pCon.config.Email.Hostname)

		to := []string{pCon.employessList.GetEmail(name)}
		msgString := fmt.Sprintf("To: %s\r\n"+
			"Subject: Raport zaległych faktur\r\n"+
			"Content-Type:	text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"<html><body>\r\n"+
			"Witaj %s <br />\n"+
			"Zestawienie zaległych faktur dla Nasza Era Sp.z o.o.<br /><br />\n"+
			"<table border='1'><tr>"+
			"<th>Nazwa firmy</th>"+
			"<th>Numer dokumentu</th>"+
			"<th>Dni spóźnienia</th>"+
			"<th>Wartość</th>"+
			"<th>Do zapłaty</th></tr>\n"+
			"%s"+
			"<tr><td></td><td></td><td></td><td><B>RAZEM</B></td><td><B>%.2f</B></td></tr>\r\n"+
			"</table><br /><br />\n"+
			"Proszę o sprawdzenie zaległości i windykację.<br /><br />\n"+
			"Janusz<br /></body></html>\n",
			pCon.employessList.GetEmail(name),
			name,
			msgTable,
			sum)

		addr := fmt.Sprintf("%s:%d", pCon.config.Email.Hostname, pCon.config.Email.Port)
		err := smtp.SendMail(addr, auth, pCon.config.Email.Username, to, []byte(msgString))
		if err != nil {
			log.Fatal(err)
		}

	}
	wg.Done()

}
