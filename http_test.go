package di

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPMiddleware(t *testing.T) {
	b, _ := NewBuilder()

	appClosed := false
	reqClosed := false

	b.Add([]Def{
		{
			Name: "object",
			Build: func(ctn Container) (interface{}, error) {
				return 1, nil
			},
			Close: func(obj interface{}) error {
				appClosed = true
				return nil
			},
		},
		{
			Name:  "request-object",
			Scope: Request,
			Build: func(ctn Container) (interface{}, error) {
				return 2, nil
			},
			Close: func(obj interface{}) error {
				reqClosed = true
				return nil
			},
		},
	}...)

	app := b.Build()

	h := func(w http.ResponseWriter, r *http.Request) {
		total := Get(r, "object").(int) + Get(r, "request-object").(int)
		io.WriteString(w, strconv.Itoa(total))
	}

	h = HTTPMiddleware(h, app, nil)

	ts := httptest.NewServer(http.HandlerFunc(h))
	defer ts.Close()

	res, err := http.Get(ts.URL)
	require.Nil(t, err)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		require.Nil(t, err)
	}

	require.Equal(t, "3", string(body))
	require.False(t, appClosed)
	require.True(t, reqClosed)
}

func TestHTTPMiddlewarePanicSubContainer(t *testing.T) {
	b, _ := NewBuilder(App)
	app := b.Build()

	recovered := false

	panicRecoveryMiddleware := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					recovered = true
				}
			}()
			h(w, r)
		}
	}

	h := func(w http.ResponseWriter, r *http.Request) {}

	h = panicRecoveryMiddleware(HTTPMiddleware(h, app, nil))

	ts := httptest.NewServer(http.HandlerFunc(h))
	defer ts.Close()

	_, err := http.Get(ts.URL)
	require.Nil(t, err)
	require.True(t, recovered)
}

func TestHTTPMiddlewarePanicDelete(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name:  "object",
		Scope: Request,
		Build: func(ctn Container) (interface{}, error) {
			return 1, nil
		},
		Close: func(obj interface{}) error {
			return errors.New("close error")
		},
	})

	app := b.Build()

	recovered := false

	panicRecoveryMiddleware := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if r := recover(); r != nil {
					recovered = true
				}
			}()
			h(w, r)
		}
	}

	logFuncUsed := false
	logFunc := func(msg string) { logFuncUsed = true }

	h := func(w http.ResponseWriter, r *http.Request) {
		Get(r, "object")
	}

	h = panicRecoveryMiddleware(HTTPMiddleware(h, app, logFunc))

	ts := httptest.NewServer(http.HandlerFunc(h))
	defer ts.Close()

	_, err := http.Get(ts.URL)
	require.Nil(t, err)
	require.False(t, recovered)
	require.True(t, logFuncUsed)
}

func TestC(t *testing.T) {
	b, _ := NewBuilder()
	app := b.Build()

	// real container
	ctn := C(app)
	require.Equal(t, app, ctn)

	// real http.Request with a container
	req, _ := http.NewRequest("", "", nil)
	req = req.WithContext(
		context.WithValue(req.Context(), ContainerKey("di"), app),
	)

	ctn = C(req)
	require.Equal(t, app, ctn)

	// real http.Request but without a container
	req, _ = http.NewRequest("", "", nil)

	require.Panics(t, func() {
		C(req)
	})

	// random object
	require.Panics(t, func() {
		C("")
	})
}

func TestRawGet(t *testing.T) {
	b, _ := NewBuilder()

	b.Add(Def{
		Name: "object",
		Build: func(ctn Container) (interface{}, error) {
			return 10, nil
		},
	})

	app := b.Build()

	obj := Get(app, "object")
	require.Equal(t, 10, obj.(int))

	require.Panics(t, func() {
		Get("", "object")
	})
}
