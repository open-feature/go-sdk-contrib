package service


type AWSServiceOptions struct{
	Region	string
}

type AWS struct{}


func New(opts ...AWSServiceOptions) *AWS{
	return &AWS{}
}
