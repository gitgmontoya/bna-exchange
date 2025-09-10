package bnaexchange

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

type (
	ExchangeResponse struct {
		Date string
		USD  Exchange
		EUR  Exchange
		BRL  Exchange
	}

	Exchange struct {
		Buy  float32
		Sell float32
	}
)

// ExtractCotizacionTable descarga la página y retorna el HTML interno del <table class="table cotizacion">
func ExtractCotizacionTable(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("no se pudo hacer GET a %s: %w", url, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("respuesta no OK: %d", res.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", fmt.Errorf("error al parsear HTML: %w", err)
	}

	sel := doc.Find("table.table.cotizacion")
	if sel.Length() == 0 {
		return "", fmt.Errorf("no se encontró elemento table.table.cotizacion")
	}

	html, err := sel.Html()
	if err != nil {
		return "", fmt.Errorf("error al obtener HTML interno: %w", err)
	}

	return html, nil
}

func main() {
}

func GetCotizacion() (*ExchangeResponse, error) {
	url := "https://www.bna.com.ar/Personas"
	htmlContent, err := ExtractCotizacionTable(url)
	if err != nil {
		return nil, err
	}

	tabla, err := ParseCotizaciones(htmlContent)
	if err != nil {
		return nil, err
	}

	resp := &ExchangeResponse{
		Date: tabla.Fecha,
		USD:  Exchange{},
		EUR:  Exchange{},
		BRL:  Exchange{},
	}
	for _, c := range tabla.Cotizaciones {
		switch c.Moneda {
		case "Dolar U.S.A":
			resp.USD = Exchange{
				Buy:  c.Compra,
				Sell: c.Venta,
			}
		case "Euro":
			resp.EUR = Exchange{
				Buy:  c.Compra,
				Sell: c.Venta,
			}
		case "Real *":
			resp.BRL = Exchange{
				Buy:  c.Compra,
				Sell: c.Venta,
			}
		}
	}

	return resp, nil
}

type Cotizacion struct {
	Moneda string
	Compra float32
	Venta  float32
}

type TablaCotizaciones struct {
	Fecha        string
	Cotizaciones []Cotizacion
}

func docFromFragmentAsRoot(raw string) (*goquery.Document, error) {
	nodes, err := html.ParseFragment(strings.NewReader(raw), &html.Node{Type: html.ElementNode, Data: "table"})
	if err != nil {
		return nil, err
	}
	root := &html.Node{Type: html.ElementNode, Data: "root"}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	return goquery.NewDocumentFromNode(root), nil
}

func ParseCotizaciones(html string) (*TablaCotizaciones, error) {
	raw := html

	// primer intento directo
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}

	lines := strings.Split(doc.Text(), "\n")
	cleanLines := []string{}
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) < 1 {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	auxValues := []float32{}
	auxIndexes := []int{4, 5, 7, 8, 10, 11}
	for i := 0; i < len(auxIndexes); i++ {
		aux, err := strconv.ParseFloat(strings.Replace(cleanLines[auxIndexes[i]], ",", ".", 1), 32)
		if err != nil {
			return nil, err
		}
		auxValues = append(auxValues, float32(aux))
	}
	tabla := TablaCotizaciones{
		Fecha: cleanLines[0],
		Cotizaciones: []Cotizacion{Cotizacion{
			Moneda: cleanLines[3],
			Compra: auxValues[0],
			Venta:  auxValues[1],
		}, Cotizacion{
			Moneda: cleanLines[6],
			Compra: auxValues[2],
			Venta:  auxValues[3],
		}, Cotizacion{
			Moneda: cleanLines[9],
			Compra: auxValues[4],
			Venta:  auxValues[5],
		}},
	}

	return &tabla, nil
}
