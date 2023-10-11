package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Constants and global variables

var (
	MODE                                = os.Getenv("MODE")
	GOOGLE_CREDENTIALS_SECRET_NAME      = os.Getenv("GOOGLE_CREDENTIALS_SECRET_NAME")
	GOOGLE_SMTP_CREDENTIALS_SECRET_NAME = os.Getenv("GOOGLE_SMTP_CREDENTIALS_SECRET_NAME")
	GOOGLE_SHEETS_ID_PLAN               = os.Getenv("GOOGLE_SHEETS_ID_PLAN")
	GOOGLE_SHEETS_RANGE_PLAN            = os.Getenv("GOOGLE_SHEETS_RANGE_PLAN")
	GOOGLE_SHEETS_RANGE_MAPPING         = os.Getenv("GOOGLE_SHEETS_RANGE_MAPPING")
	GOOGLE_SHEETS_ID_ADDRESSES          = os.Getenv("GOOGLE_SHEETS_ID_ADDRESSES")
	GOOGLE_SHEETS_RANGE_ADDRESSES       = os.Getenv("GOOGLE_SHEETS_RANGE_ADDRESSES")
	GOOGLE_SHEETS_ID_OPTIONS            = os.Getenv("GOOGLE_SHEETS_ID_OPTIONS")
	GOOGLE_SHEETS_RANGE_OPTIONS         = os.Getenv("GOOGLE_SHEETS_RANGE_OPTIONS")
	secretCache, _                      = secretcache.New()
)

var smtpA SMTPAccount

// Structs

type MyEvent struct{}

type ServiceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

type SMTPAccount struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Sender struct {
	auth smtp.Auth
}

type Message struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string
	ContentType string
	Attachments map[string][]byte
}

// Templates

// Whatsapp-Url (see: https://www.callmebot.com/blog/free-api-whatsapp-messages/)
const whatsappApiUrlTemplate = "https://api.callmebot.com/whatsapp.php?phone=%s&text=%s&apikey=%s"

// Email-Template Evening
const emailSubjectTemplateEvening = "Erinnerung Ampeldienst morgen früh!"
const emailBodyTemplateEvening = `
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ampeldienst-Reminder</title>
    <style>
        /* Add any desired CSS styling here. */
        body {
            font-family: Arial, sans-serif;
            margin: 40px;
        }
        header {
            background-color: #f5f5f5;
            padding: 10px 20px;
            border-bottom: 2px solid #ddd;
        }
        .content {
            padding: 20px;
        }
    </style>
</head>

<body>
    <header>
        <h2 text-align: center>🚦 🚦 🚦 Ampeldienst Bernried 🚦 🚦 🚦</h1>
    </header>

    <div class="content">
		<p>Hallo {{.Name}},<br></p>
		<p>kleine Erinnerung, dass du morgen früh Ampeldienst hast 😉.<br></p>
		<p>Vielen Dank, dass du daran denkst! 👊 👨‍👩‍👧‍👦 🚴‍♂️ <br></p>
		<p>Viele Grüße,<br></p>
		<p>Dein Ampeldienst-Team</p>
		<p>🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦</p>
	</div>
</body>
				
</html>
`

// Whatsapp-Template Evening
const whatsAppTemplateEvening = "🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦\r\n \r\n \r\nHallo %s,\r\n \r\nkleine Erinnerung, dass du morgen früh Ampeldienst hast. 😉\r\n \r\nVielen Dank, dass du daran denkst! 👊 👨‍👩‍👧‍👦 🚴‍♂️ \r\n \r\nDein Ampeldienst-Team \r\n \r\n \r\n🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦"

// Morning

// Email-Template Morning
const emailSubjectTemplateMorning = "Erinnerung an deinen Ampeldienst in 15 Minuten!"
const emailBodyTemplateMorning = `
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ampeldienst-Reminder</title>
    <style>
        /* Add any desired CSS styling here. */
        body {
            font-family: Arial, sans-serif;
            margin: 40px;
        }
        header {
            background-color: #f5f5f5;
            padding: 10px 20px;
            border-bottom: 2px solid #ddd;
        }
        .content {
            padding: 20px;
        }
    </style>
</head>

<body>
    <header>
        <h2 text-align: center>🚦 🚦 🚦 Ampeldienst Bernried 🚦 🚦 🚦</h1>
    </header>

    <div class="content">
		<p>Hallo {{.Name}},<br></p>
		<p>kleine Erinnerung, dass du gleich Ampeldienst hast ⏰ ⏰ ⏰.<br></p>
		<p>Vielen Dank, dass du daran denkst! 👊 👨‍👩‍👧‍👦 🚴‍♂️ <br></p>
		<p>Viele Grüße,<br></p>
		<p>Dein Ampeldienst-Team</p>
		<p>🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦</p>
	</div>
</body>
								
</html>
`

// Whatsapp-Template Morning
const whatsAppTemplateMorning = "🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦\r\n \r\n \r\nHallo %s,\r\n \r\nkleine Erinnerung, dass du in 15 Minuten Ampeldienst hast. ⏰ ⏰ ⏰ \r\n \r\nVielen Dank, dass du daran denkst! 👊 👨‍👩‍👧‍👦 🚴‍♂️\r\n \r\nDein Ampeldienst-Team \r\n \r\n \r\n🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦 🚦"

// ICS-Template
const icsTemplate = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:🚦Ampeldienst🚦
DTSTART:%sT050000Z
DTEND:%sT054500Z
DTSTAMP:%sT061919Z
UID:%d-Ampeldienst
DESCRIPTION:
LOCATION:Ampel
ORGANIZER:ampeldienst.bernried@gmail.com
STATUS:CONFIRMED
PRIORITY:0
END:VEVENT
END:VCALENDAR`

// Main Lambda code

func HandleRequest(ctx context.Context, event MyEvent) error {

	var err error
	var planResponse *sheets.ValueRange
	var mappingResponse *sheets.ValueRange
	var addressResponse *sheets.ValueRange
	var optionsResponse *sheets.ValueRange
	var sa ServiceAccount

	// Define date to use for the service

	location, err := time.LoadLocation("CET")
	if err != nil {
		log.Fatalf("Unable to load location: %v", err)
	}

	// Convert to CET timezone.
	nowCET := time.Now().In(location)
	log.Printf("Date:%v", nowCET)

	// Based on evening or morning mode get tomorrow's or today's date
	var actDate time.Time
	switch MODE {
	case "EVENING":
		actDate = nowCET.AddDate(0, 0, 1) // get tommorrows date
		log.Printf("Eveningmode: Date:%v", actDate)
	case "MORNING":
		actDate = nowCET // get today's date
		log.Printf("Morningmode: Date:%v", actDate)
	default:
		log.Print("Invalid MODE:", MODE)
		return nil // exit the function early if MODE is not recognized
	}

	// Create Googlesheet-Client and retrieve data from Ampelplan shield

	log.Println("Getting SMTP Credentials from secret")
	result, err := secretCache.GetSecretString(GOOGLE_SMTP_CREDENTIALS_SECRET_NAME)
	if err != nil {
		log.Fatalf("Failed to get SMTP secret: %v", err)
	}
	err = json.Unmarshal([]byte(result), &smtpA)
	if err != nil {
		fmt.Println("Error:", err)
	}

	log.Println("Getting Service Account Credentials from secret")
	// Retrieve the secrets from Secretsmanager
	result, err = secretCache.GetSecretString(GOOGLE_CREDENTIALS_SECRET_NAME)
	if err != nil {
		log.Fatalf("Failed to get GoogleCredential-secret: %v", err)
	}
	err = json.Unmarshal([]byte(result), &sa)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Create googlesheet request config from service account credentials
	conf := &jwt.Config{
		Email:        sa.ClientEmail,
		PrivateKey:   []byte(sa.PrivateKey),
		PrivateKeyID: sa.PrivateKeyID,
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"https://www.googleapis.com/auth/spreadsheets",
		},
	}
	log.Println("Credentials loaded from SecretsManager secrets")

	// Create a Google client
	client := conf.Client(context.Background())

	// Create a service object for Google sheets
	srv, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets client: %v", err)
	}

	// Retrieve the data from the Ampelplan-sheet
	planResponse, err = srv.Spreadsheets.Values.Get(GOOGLE_SHEETS_ID_PLAN, GOOGLE_SHEETS_RANGE_PLAN).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from Ampelplan-sheet: %v", err)
	}

	log.Println("Retrieved plan data from sheet")

	foundDate := false
	lotsendID := ""
	var parsedDate time.Time
	for _, row := range planResponse.Values {
		if !foundDate {
			for idx, col := range row {
				if idx == 1 {
					dateStr := fmt.Sprintf("%v", col)
					parsedDate, err = time.Parse("02.01.2006", dateStr)
					if err != nil {
						fmt.Println("Error parsing date:", err)
					}
					// Compare the year, month, and day values
					if parsedDate.Year() == actDate.Year() && parsedDate.Month() == actDate.Month() && parsedDate.Day() == actDate.Day() {
						foundDate = true
					} else {
						foundDate = false
					}
				}

				if foundDate && idx == 10 {
					lotsendID = fmt.Sprintf("%v", col)
				}
			}
		}
	}
	log.Println("Found Date:", parsedDate)
	log.Println("LotsendID:", lotsendID)

	// Get contact details and reminder options
	first_name1 := ""
	last_name1 := ""
	email_adress1 := ""
	phone_number1 := ""
	email_consent1 := ""
	phone_consent1 := ""
	whatsapp_apikey1 := ""
	first_name2 := ""
	last_name2 := ""
	email_adress2 := ""
	phone_number2 := ""
	email_consent2 := ""
	phone_consent2 := ""
	whatsapp_apikey2 := ""

	if lotsendID != "" {

		// Retrieve names for the lotsenID from the mapping sheet
		mappingResponse, err = srv.Spreadsheets.Values.Get(GOOGLE_SHEETS_ID_PLAN, GOOGLE_SHEETS_RANGE_MAPPING).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from mapping-sheet: %v", err)
		}
		foundLotse := false
		log.Println("Retrieve names for the lotsenID from the mapping sheet")

		for _, row := range mappingResponse.Values {
			if !foundLotse {
				for idx, col := range row {
					if idx == 0 {
						actLotsenID := fmt.Sprintf("%v", col)

						if actLotsenID == lotsendID {
							foundLotse = true
						} else {
							foundLotse = false
						}
					}

					if foundLotse && idx == 1 {
						last_name1 = fmt.Sprintf("%v", col)
					}

					if foundLotse && idx == 2 {
						first_name1 = fmt.Sprintf("%v", col)
					}

					if foundLotse && idx == 3 {
						last_name2 = fmt.Sprintf("%v", col)
					}

					if foundLotse && idx == 4 {
						first_name2 = fmt.Sprintf("%v", col)
					}

				}
			}
		}
		log.Printf("Found Lotsen: %v, %v, %v, %v, %v", lotsendID, first_name1, last_name1, last_name2, first_name2)

		// Get Address-data from the Address-sheet
		addressResponse, err = srv.Spreadsheets.Values.Get(GOOGLE_SHEETS_ID_ADDRESSES, GOOGLE_SHEETS_RANGE_ADDRESSES).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from Address-sheet: %v", err)
		}

		// Get Reminder-options from the Reminder-Options-sheet
		optionsResponse, err = srv.Spreadsheets.Values.Get(GOOGLE_SHEETS_ID_OPTIONS, GOOGLE_SHEETS_RANGE_OPTIONS).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from Reminder-Options-sheet: %v", err)
		}

		if first_name1 != "" && last_name1 != "" {
			// Pull the data from the sheet
			log.Printf("Getting contact details for Lotse 1")
			email_adress1, phone_number1, err = getActContactDetailsForLotse(addressResponse, first_name1, last_name1)
			if err != nil {
				log.Fatalf("Unable to retrieve contact details from sheet: %v", err)
			}

			log.Printf("Getting reminder options for Lotse 1")
			email_consent1, phone_consent1, whatsapp_apikey1, err = getActReminderOptionsForLotse(optionsResponse, first_name1, last_name1)
			if err != nil {
				log.Printf("Unable to retrieve Reminder options  from sheet: %v", err)
			}
		}

		if first_name2 != "" && last_name2 != "" {
			// Pull the data from the sheet
			log.Printf("Getting contact details for Lotse 2")
			email_adress2, phone_number2, err = getActContactDetailsForLotse(addressResponse, first_name2, last_name2)
			if err != nil {
				log.Printf("Unable to retrieve contact details from sheet: %v", err)
			}

			log.Printf("Getting reminder options for Lotse 2")
			email_consent2, phone_consent2, whatsapp_apikey2, err = getActReminderOptionsForLotse(optionsResponse, first_name2, last_name2)
			if err != nil {
				log.Printf("Unable to retrieve Reminder options  from sheet: %v", err)
			}
		}

	}

	log.Printf("Lotsen-Configuration: %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v", lotsendID, first_name1, last_name1, email_adress1, phone_number1, email_consent1, phone_consent1, whatsapp_apikey1, first_name2, last_name2, email_adress2, phone_number2, email_consent2, phone_consent2, whatsapp_apikey2)

	icsFile, err := createICSFile(actDate.Format("20060102"))
	if err != nil {
		log.Fatalf("Unable to create ics file : %v", err)
	}

	var whatsappTemplate string
	var emailTemplate string
	var emailSubject string

	switch MODE {
	case "EVENING":
		whatsappTemplate = whatsAppTemplateEvening
		emailSubject = emailSubjectTemplateEvening
		emailTemplate = emailBodyTemplateEvening
	case "MORNING":
		whatsappTemplate = whatsAppTemplateMorning
		emailSubject = emailSubjectTemplateMorning
		emailTemplate = emailBodyTemplateMorning
	default:
		log.Print("Invalid MODE:", MODE)
		return nil // exit the function early if MODE is not recognized
	}

	if first_name1 != "" && phone_number1 != "" && phone_consent1 == "Ja" && whatsapp_apikey1 != "" {

		log.Println("Sending Whatsapp message to contact 1")
		err = sendWhatsappMessage(phone_number1, whatsapp_apikey1, whatsappTemplate, first_name1)
		if err != nil {
			log.Printf("Unable to send whatsapp message: %v", err)
		}

	}

	if first_name1 != "" && email_adress1 != "" && email_consent1 == "Ja" {

		log.Println("Sending Email message to contact 1")
		err = sendMailUsingGmail(email_adress1, emailSubject, emailTemplate, first_name1, icsFile)
		if err != nil {
			log.Printf("Unable to send Email message: %v", err)
		}
	}

	if first_name2 != "" && phone_number2 != "" && phone_consent2 == "Ja" && whatsapp_apikey2 != "" {

		log.Println("Sending Whatsapp message to contact 2")
		err = sendWhatsappMessage(phone_number2, whatsapp_apikey2, whatsappTemplate, first_name2)
		if err != nil {
			log.Printf("Unable to send whatsapp message: %v", err)
		}

	}

	if first_name2 != "" && email_adress2 != "" && email_consent2 == "Ja" {

		log.Println("Sending Email message to contact 2")
		err = sendMailUsingGmail(email_adress2, emailSubject, emailTemplate, first_name2, icsFile)
		if err != nil {
			log.Printf("Unable to send Email message: %v", err)
		}
	}

	return nil

}

func getActContactDetailsForLotse(addressResponse *sheets.ValueRange, first_name string, last_name string) (string, string, error) {

	lotsenName := last_name + ", " + first_name

	email_address := ""
	phone_number := ""
	for _, row := range addressResponse.Values {
		// Check if the row has enough columns to avoid index out of range errors
		if len(row) >= 3 {
			// Perform type assertion for strings
			if lastName, ok := row[0].(string); ok && lastName == last_name {
				if firstName, ok := row[1].(string); ok && firstName == first_name {
					// Perform type assertion for 3rd and 4th columns
					if email, ok := row[2].(string); ok {
						email_address = email
					}
					if len(row) >= 4 {
						if phone, ok := row[3].(string); ok {
							phone_number = phone
						}
					}
					if email_address == "" && phone_number == "" {
						return "", "", fmt.Errorf("no contact details found for %s", lotsenName)
					}
					return email_address, phone_number, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("no contact details found for %s", lotsenName)

}

func getActReminderOptionsForLotse(optionsResponse *sheets.ValueRange, first_name string, last_name string) (string, string, string, error) {

	lotsenName := last_name + ", " + first_name

	email_consent := ""
	phone_consent := ""
	whatsapp_apikey := ""

	for _, row := range optionsResponse.Values {
		// Check if the row has enough columns to avoid index out of range errors
		if len(row) >= 4 {
			// Perform type assertion for strings
			if actLotsenName, ok := row[1].(string); ok && actLotsenName == lotsenName {
				// Perform type assertion for 3rd and 4th columns
				if email, ok := row[2].(string); ok {
					email_consent = email
				}
				if phone, ok := row[3].(string); ok {
					phone_consent = phone
				}
				if len(row) >= 5 {
					if apikey, ok := row[4].(string); ok {
						whatsapp_apikey = apikey
					}
				}
			}
		}
	}

	if email_consent == "" && phone_consent == "" {
		return "Ja", "", "", fmt.Errorf("no reminder options found for %s", lotsenName)
	}
	return email_consent, phone_consent, whatsapp_apikey, nil

}

func sendMailUsingGmail(emailAddress string, emailSubject string, emailTemplate string, firstName string, icsFile string) error {
	//log.Printf("Parse Template")
	body, err := parseEmailTemplate(emailTemplate, firstName)
	if err != nil {
		return fmt.Errorf("unable to parse email template: %v", err)
	}
	//log.Printf("Sending Mail: email_address: %v, firstname: %v, emailSubject: %v", emailAddress, firstName, emailSubject)
	sender := New()
	m := NewMessage(emailSubject, body, "text/html")
	m.To = []string{emailAddress}

	if MODE == "EVENING" {
		m.AttachFile(icsFile)
	}
	//log.Printf("Sending Mail")
	err = sender.Send(m)
	//log.Printf("Mail send!")
	if err != nil {
		return fmt.Errorf("unable to send email: %v", err)
	}
	return nil
}

func New() *Sender {
	//log.Printf("New Sender: %v, %v, %v", EMAIL_USERNAME, EMAIL_PASSWORD, EMAIL_HOST)
	auth := smtp.PlainAuth("", smtpA.Username, smtpA.Password, smtpA.Host)
	//log.Printf("New Sender: %v, %v, %v",  EMAIL_USERNAME, EMAIL_PASSWORD, EMAIL_HOST)
	return &Sender{auth}
}

func (s *Sender) Send(m *Message) error {
	return smtp.SendMail(fmt.Sprintf("%s:%s", smtpA.Host, smtpA.Port), s.auth, smtpA.Username, m.To, m.ToBytes())
}

func NewMessage(s, b, contentType string) *Message {
	return &Message{Subject: s, Body: b, ContentType: contentType, Attachments: make(map[string][]byte)}
}

func (m *Message) AttachFile(src string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	_, fileName := filepath.Split(src)
	m.Attachments[fileName] = b
	return nil
}

func (m *Message) ToBytes() []byte {
	buf := bytes.NewBuffer(nil)
	withAttachments := len(m.Attachments) > 0
	buf.WriteString(fmt.Sprintf("Subject: %s\n", m.Subject))
	buf.WriteString(fmt.Sprintf("To: %s\n", strings.Join(m.To, ",")))
	if len(m.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\n", strings.Join(m.CC, ",")))
	}

	if len(m.BCC) > 0 {
		buf.WriteString(fmt.Sprintf("Bcc: %s\n", strings.Join(m.BCC, ",")))
	}

	buf.WriteString("MIME-Version: 1.0\n")
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()
	if withAttachments {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\n", boundary))
		buf.WriteString(fmt.Sprintf("--%s\n", boundary))
		buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\n", m.ContentType))
	} else {
		buf.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\n", m.ContentType))
	}

	buf.WriteString(m.Body)

	if withAttachments {
		for k, v := range m.Attachments {
			buf.WriteString(fmt.Sprintf("\n\n--%s\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: %s\n", http.DetectContentType(v)))
			buf.WriteString("Content-Transfer-Encoding: base64\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=%s\n", k))

			b := make([]byte, base64.StdEncoding.EncodedLen(len(v)))
			base64.StdEncoding.Encode(b, v)
			buf.Write(b)
			buf.WriteString(fmt.Sprintf("\n--%s", boundary))
		}

		buf.WriteString("--")
	}

	return buf.Bytes()
}

func createICSFile(date string) (string, error) {

	// Create content of the ICS file
	content := icsTemplate

	content = fmt.Sprintf(content, date, date, date, time.Now().Unix())

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "ampeldienst-*.ics")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write content to the file
	_, err = tmpFile.WriteString(content)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func parseEmailTemplate(emailTemplateInput string, name string) (string, error) {
	t, err := template.New("email").Parse(emailTemplateInput)
	if err != nil {
		return "", err
	}

	data := struct {
		Name string
	}{
		Name: name,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func sendWhatsappMessage(phone string, apikey string, template string, first_name string) error {
	// Encode message to safely place in URL

	message := fmt.Sprintf(template, first_name)
	encodedMessage := url.QueryEscape(message)
	toString := strings.Replace(strings.Replace(phone, "0049", "49", 10), " ", "", 10)
	apiUrl := fmt.Sprintf(whatsappApiUrlTemplate, toString, encodedMessage, apikey)
	// print(apiUrl)

	resp, err := http.Get(apiUrl)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bodyString := string(bodyBytes)
	if strings.Contains(bodyString, "<b>APIKey is invalid.</b>") {
		return fmt.Errorf("failed to send message, Invalid APIKEY provided")
	}

	return nil
}

func main() {
	lambda.Start(HandleRequest)
}
