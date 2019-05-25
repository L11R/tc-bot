package tool

import (
	"bytes"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
	"mime/multipart"
	"net/http"
	"strings"
	"tc-bot/model"
	"time"
)

const EmptyCaptcha = "iVBORw0KGgoAAAANSUhEUgAAAMgAAAA8CAYAAAAjW/WRAAAAtklEQVR4nO3TAQ3AMAzAsH78OfcEtiCwIUTKt7s7wNWRBd4MAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSAQDALBIBAMAsEgEAwCwSDwMjM/4U4EdJ8W68gAAAAASUVORK5CYII="

func GetForm() (*model.Form, error) {
	resp, err := http.Get("http://81.23.146.8/default.aspx")
	if err != nil {
		return nil, errors.Wrap(err, "cannot do get request")
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create new document")
	}

	viewState, ok := doc.Find("#__VIEWSTATE").Attr("value")
	if !ok {
		return nil, errors.New("cannot find __VIEWSTATE value")
	}

	eventValidation, ok := doc.Find("#__EVENTVALIDATION").Attr("value")
	if !ok {
		return nil, errors.New("cannot find __EVENTVALIDATION value")
	}

	src, ok := doc.Find("img").Attr("src")
	if !ok {
		return nil, errors.New("cannot find img src value")
	}
	src = "http://81.23.146.8/" + src

	return &model.Form{
		ViewState:       viewState,
		EventValidation: eventValidation,
		CaptchaLink:     src,
	}, nil
}

var IncorrectCode = errors.New("form: incorrect code")

func PostForm(viewState, eventValidation string, cardNumber, code int) (string, error) {
	body := bytes.NewBuffer(nil)

	w := multipart.NewWriter(body)

	if err := w.WriteField("__VIEWSTATE", viewState); err != nil {
		return "", errors.Wrap(err, "cannot write field")
	}
	if err := w.WriteField("__EVENTVALIDATION", eventValidation); err != nil {
		return "", errors.Wrap(err, "cannot write field")
	}
	if err := w.WriteField("cardnum", fmt.Sprintf("%010d", cardNumber)); err != nil {
		return "", errors.Wrap(err, "cannot write field")
	}
	if err := w.WriteField("checkcode", fmt.Sprintf("%04d", code)); err != nil {
		return "", errors.Wrap(err, "cannot write field")
	}

	if err := w.Close(); err != nil {
		return "", errors.Wrap(err, "cannot close multipart writer")
	}

	req, err := http.NewRequest("POST", "http://81.23.146.8/default.aspx", body)
	if err != nil {
		return "", errors.Wrap(err, "cannot create request")
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "cannot do request")
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "cannot create new document")
	}

	if strings.Contains(doc.Text(), "Код проверки введен с ошибкой") {
		return "", IncorrectCode
	}

	text := fmt.Sprintf("<b>%s</b>\n\n", doc.Find(".PageHeader").Text())
	doc.Find("table").Last().Find("tr").Each(func(i int, selection *goquery.Selection) {
		text += fmt.Sprintf(
			"<b>%s</b>: %s\n",
			strings.ReplaceAll(selection.Find(".FieldHeader").Text(), ":", ""),
			selection.Find(".FieldValue").Text(),
		)
	})

	return text, nil
}

type HumanReadableError interface {
	error
	Human() string
	Cause() error
}

// Human-readable Error
type HRError struct {
	human string
	error error
}

func NewHRError(human string, err error) HumanReadableError {
	return &HRError{human: human, error: err}
}

// Just to complain error interface, it should be named String() I guess
func (e *HRError) Error() string {
	return e.error.Error()
}

func (e *HRError) Human() string {
	return e.human
}

func (e *HRError) Cause() error {
	return e.error
}
