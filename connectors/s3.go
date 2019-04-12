package connectors

import (
	"bytes"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Connector struct {
	session *session.Session
	bucket  string
}

// NewS3Connector returns a pointer to a new S3Connector that uses the profile in .aws/credentials
func NewS3Connector(bucketname string) (*S3Connector, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return &S3Connector{
		session: sess,
		bucket:  bucketname,
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
