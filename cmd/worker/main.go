package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// MeasurementStation represents a single measurement station that
// is placed somewhere in the cit
type MeasurementStation struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	UpdatedAt   string  `json:"updated_at"`
	Value       float64 `json:"value"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type field struct {
	Value string `json:"valor_alfanumerico"`
}

type siataPoint struct {
	Attributes struct {
		Metadata struct {
			Name field `json:"name"`
		} `json:"metadato"`

		Description struct {
			PM25Val     field `json:"ICA_PM25_Valor"`
			PM25ValDesc field `json:"ICA_PM25_Descripcion"`
			Longitude   field `json:"Longitud"`
			Latitude    field `json:"Latitud"`
			UpdatedAt   field `json:"fecha_ultima_actualizacion"`
		} `json:"descripcion"`
	} `json:"atributos"`
}

// Station returns a normalized `MeasurementStation` from the messy
// original siata response.
func (sp *siataPoint) Station() (*MeasurementStation, error) {
	value, err := strconv.ParseFloat(sp.Attributes.Description.PM25Val.Value, 64)
	if err != nil {
		return nil, err
	}

	lat, err := strconv.ParseFloat(sp.Attributes.Description.Latitude.Value, 64)
	if err != nil {
		return nil, err
	}

	lng, err := strconv.ParseFloat(sp.Attributes.Description.Longitude.Value, 64)
	if err != nil {
		return nil, err
	}

	return &MeasurementStation{
		Name:        sp.Attributes.Metadata.Name.Value,
		Description: sp.Attributes.Description.PM25ValDesc.Value,
		UpdatedAt:   sp.Attributes.Description.UpdatedAt.Value,
		Value:       value,
		Latitude:    lat,
		Longitude:   lng,
	}, nil
}

func upload(s *session.Session, key string, data []byte) error {
	log.Printf("Uploading file: %s", key)

	var (
		bucket      = "siata.picoyplaca.org"
		acl         = "public-read"
		contentType = "application/json"
	)

	uploader := s3manager.NewUploader(s)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Body:        bytes.NewReader(data),
		ACL:         aws.String(acl),
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	})

	return err
}

func run() error {
	var (
		url    = "https://siata.gov.co/siata_nuevo/index.php/capa_service/consultar_capa_carga"
		params = bytes.NewBufferString("id_capa=C_00000000000000000000602")

		// The SIATA service is very slow so we neeed to set very long timeouts
		client = http.Client{
			Timeout: 2 * time.Minute,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
					DualStack: true,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   30 * time.Second,
				ExpectContinueTimeout: 2 * time.Second,
			},
		}
	)

	log.Printf("Requesting map data")
	resp, err := client.Post(url, "application/x-www-form-urlencoded", params)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response struct {
		Items []*siataPoint `json:"feature_vector"`
	}

	log.Printf("Parsing map data")
	if err := json.Unmarshal(body, &response); err != nil {
		log.Println(string(body))
		return err
	}

	var data struct {
		GeneratedAt string                `json:"generated_at"`
		Stations    []*MeasurementStation `json:"stations"`
	}

	loc, _ := time.LoadLocation("America/Bogota")
	genTime := time.Now().In(loc)
	data.GeneratedAt = genTime.Format(time.Stamp)

	log.Printf("Got %d items", len(response.Items))
	if len(response.Items) == 0 {
		log.Println(string(body))
		return nil
	}

	for _, item := range response.Items {
		station, err := item.Station()
		if err != nil {
			return err
		}

		data.Stations = append(data.Stations, station)
	}

	s, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		return err
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := upload(s, "pm25.json", raw); err != nil {
		return err
	}

	js := []byte(fmt.Sprintf("var data = %s;", string(raw)))
	if err := upload(s, "data.js", js); err != nil {
		return err
	}

	return recordMetrics(s, data.Stations)
}

func asciiName(name string) string {
	name = strings.Replace(name, "á", "a", -1)
	name = strings.Replace(name, "é", "e", -1)
	name = strings.Replace(name, "í", "i", -1)
	name = strings.Replace(name, "ó", "o", -1)
	name = strings.Replace(name, "ú", "u", -1)
	name = strings.Replace(name, "ü", "u", -1)
	name = strings.Replace(name, "#", "No. ", -1)

	return name
}

func recordMetrics(s *session.Session, stations []*MeasurementStation) error {
	svc := cloudwatch.New(s)
	var metrics []*cloudwatch.MetricDatum

	for _, station := range stations {
		if station.Value < 0 {
			station.Value = 0
		}

		datum := &cloudwatch.MetricDatum{
			MetricName: aws.String("PM25"),
			Unit:       aws.String("Count"),
			Value:      aws.Float64(station.Value),
			Dimensions: []*cloudwatch.Dimension{
				&cloudwatch.Dimension{
					Name:  aws.String("StationName"),
					Value: aws.String(asciiName(station.Name)),
				},
			},
		}

		metrics = append(metrics, datum)
	}

	_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace:  aws.String("Station/2.5Meassurements"),
		MetricData: metrics,
	})

	return err
}

func Handler(ctx context.Context) error {
	for i := 0; i < 3; i++ {
		log.Printf("[%d/3] Fetching data", i+1)
		if err := run(); err == nil {
			return nil
		}
	}

	return errors.New("could not get data after 3 attempts")
}

func main() {
	lambda.Start(Handler)
}
