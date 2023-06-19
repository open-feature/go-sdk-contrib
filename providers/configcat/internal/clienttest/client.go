package clienttest

import (
	"sync"

	sdk "github.com/configcat/go-sdk/v7"
)

// NewClient creates enough of the ConfigCat client to record flag interactions.
func NewClient() *Client {
	return &Client{}
}

type Client struct {
	mu       sync.Mutex
	requests []Request

	boolEvaluation   func(req Request) sdk.BoolEvaluationDetails
	stringEvaluation func(req Request) sdk.StringEvaluationDetails
	floatEvaluation  func(req Request) sdk.FloatEvaluationDetails
	intEvaluation    func(req Request) sdk.IntEvaluationDetails
}

type Request struct {
	Key          string
	DefaultValue interface{}
	User         sdk.User
}

func (r *Request) UserData() sdk.UserData {
	userData, ok := r.User.(*sdk.UserData)
	if !ok {
		panic("user is not of type sdk.UserData")
	}
	return *userData
}

func (c *Client) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = nil
	c.boolEvaluation = nil
	c.stringEvaluation = nil
	c.floatEvaluation = nil
	c.intEvaluation = nil
}

func (c *Client) GetRequests() []Request {
	c.mu.Lock()
	defer c.mu.Unlock()
	requests := make([]Request, len(c.requests))
	copy(requests, c.requests)
	return requests
}

func (c *Client) WithBoolEvaluation(eval func(req Request) sdk.BoolEvaluationDetails) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.boolEvaluation = eval
}

func (c *Client) WithStringEvaluation(eval func(req Request) sdk.StringEvaluationDetails) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stringEvaluation = eval
}

func (c *Client) WithFloatEvaluation(eval func(req Request) sdk.FloatEvaluationDetails) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.floatEvaluation = eval
}

func (c *Client) WithIntEvaluation(eval func(req Request) sdk.IntEvaluationDetails) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.intEvaluation = eval
}

func (c *Client) GetBoolValueDetails(key string, defaultValue bool, user sdk.User) sdk.BoolEvaluationDetails {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := Request{
		Key:          key,
		DefaultValue: defaultValue,
		User:         user,
	}
	c.requests = append(c.requests, req)

	if c.boolEvaluation == nil {
		return sdk.BoolEvaluationDetails{
			Data:  evalNotFound(key, user),
			Value: defaultValue,
		}
	}

	return c.boolEvaluation(req)
}

func (c *Client) GetStringValueDetails(key string, defaultValue string, user sdk.User) sdk.StringEvaluationDetails {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := Request{
		Key:          key,
		DefaultValue: defaultValue,
		User:         user,
	}
	c.requests = append(c.requests, req)

	if c.stringEvaluation == nil {
		return sdk.StringEvaluationDetails{
			Data:  evalNotFound(key, user),
			Value: defaultValue,
		}
	}

	return c.stringEvaluation(req)
}

func (c *Client) GetFloatValueDetails(key string, defaultValue float64, user sdk.User) sdk.FloatEvaluationDetails {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := Request{
		Key:          key,
		DefaultValue: defaultValue,
		User:         user,
	}
	c.requests = append(c.requests, req)

	if c.floatEvaluation == nil {
		return sdk.FloatEvaluationDetails{
			Data:  evalNotFound(key, user),
			Value: defaultValue,
		}
	}

	return c.floatEvaluation(req)
}

func (c *Client) GetIntValueDetails(key string, defaultValue int, user sdk.User) sdk.IntEvaluationDetails {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := Request{
		Key:          key,
		DefaultValue: defaultValue,
		User:         user,
	}
	c.requests = append(c.requests, req)

	if c.intEvaluation == nil {
		return sdk.IntEvaluationDetails{
			Data:  evalNotFound(key, user),
			Value: defaultValue,
		}
	}

	return c.intEvaluation(req)
}

func evalNotFound(key string, user sdk.User) sdk.EvaluationDetailsData {
	return sdk.EvaluationDetailsData{
		Key:            key,
		User:           user,
		IsDefaultValue: true,
		Error:          sdk.ErrKeyNotFound{Key: key},
	}
}
