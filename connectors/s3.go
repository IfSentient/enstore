package connectors

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Connector struct {
	session *session.Session
	bucket  string
}

type S3Config struct {
	Region      string
	Endpoint    string
	Bucket      string
	Credentials *S3Credentials
}

type S3Credentials struct {
	ID     string
	Secret string
	Token  string
}

func LoadS3Config(path string) (*S3Config, error) {
	var cfg S3Config
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// NewS3Connector returns a pointer to a new S3Connector
func NewS3Connector(cfg S3Config) (*S3Connector, error) {
	var sess *session.Session
	var err error
	if cfg.Endpoint == "" && cfg.Region == "" && cfg.Credentials == nil {
		sess, err = session.NewSession()
	} else {
		config := &aws.Config{}
		if cfg.Region != "" {
			config.Region = aws.String(cfg.Region)
		}
		if cfg.Endpoint != "" {
			config.Endpoint = aws.String(cfg.Endpoint)
		}
		if cfg.Credentials != nil {
			config.Credentials = credentials.NewStaticCredentials(cfg.Credentials.ID, cfg.Credentials.Secret, cfg.Credentials.Token)
		}
		sess, err = session.NewSession(config)
	}
	if err != nil {
		return nil, err
	}
	return &S3Connector{
		session: sess,
		bucket:  cfg.Bucket,
	}, nil
}

func (s *S3Connector) Read(blockname string) ([]byte, error) {
	downloader := s3manager.NewDownloader(s.session) // TODO: put these in struct?
	buffer := aws.NewWriteAtBuffer([]byte{})

	_, err := downloader.Download(buffer, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(blockname),
	})
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (s *S3Connector) Write(blockname string, data []byte) error {
	uploader := s3manager.NewUploader(s.session) // TODO: put these in struct?
	reader := bytes.NewReader(data)

	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(blockname),
		Body:   reader,
	})
	return err
}

func (s *S3Connector) Exists(filename string) bool {
	client := s3.New(s.session)
	_, err := client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filename),
	})
	return err == nil
}
