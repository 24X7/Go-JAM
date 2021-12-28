package backplane

import (
	"crypto/rand"
	"math/big"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/workos-inc/workos-go/pkg/sso"

	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/encoding"
	"github.com/philippgille/gokv/leveldb"

	petname "github.com/dustinkirkland/golang-petname"
)

type SystemConfig struct {
	WORKOS_API_KEY         string
	WORKOS_CLIENT_ID       string
	ALLOWED_ROOT_APP_USERS string
	PORT                   string
	APP_PORT               string
	API_PORT               string
}

type BlobCallOptions struct {
	AppCode     string
	AuthToken   string
	ContentType string
	Key         string
	Path        string
}

type Blob struct {
	Id       string      `json:"id"`
	Title    string      `json:"title"`
	Contents interface{} `json:"contents"`
}

type ListOfItems struct {
	Id       string   `json:"id"`
	Title    string   `json:"title"`
	Contents []string `json:"contents"`
}

type AppCreds struct {
	AppCode      string   `json:"appCode"`
	APIKey       string   `json:"apiKey"`
	Title        string   `json:"title"`
	ContentTypes []string `json:"contentTypes"`
}

const lettersForId = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const lettersForToken = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz!@#$%^*()_-=+?.,"

var CONFIG SystemConfig
var isBootstrap bool = false

func init() {
	CONFIG = SystemConfig{
		WORKOS_API_KEY:         os.Getenv("WORKOS_API_KEY"),
		WORKOS_CLIENT_ID:       os.Getenv("WORKOS_CLIENT_ID"),
		ALLOWED_ROOT_APP_USERS: os.Getenv("ALLOWED_ROOT_APP_USERS"),
		PORT:                   os.Getenv("PORT"),
		APP_PORT:               os.Getenv("APP_PORT"),
		API_PORT:               os.Getenv("API_PORT"),
	}
}

func Bootstrap() {
	if isBootstrap {
		return
	}
	isBootstrap = true

	sso.Configure(CONFIG.WORKOS_API_KEY, CONFIG.WORKOS_CLIENT_ID)

	root := AppCreds{}
	exist, err := GetAppAuthStore().Get("__ROOT__", &root)
	if !exist || err != nil {
		root.Title = "__ROOT__"
		root.AppCode = "__ROOT__"
		root.ContentTypes = []string{"user"}
		root.APIKey = GenerateAuthToken()
		GetAppAuthStore().Set("__ROOT__", root)
	}
}

func GenerateRandomString(n int, abc string) (string, error) {
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(abc))))
		if err != nil {
			return "", err
		}
		ret[i] = abc[num.Int64()]
	}

	return string(ret), nil
}

func GenerateId(prefix string, n int) string {
	token, _ := GenerateRandomString(n, lettersForId)
	return prefix + "_" + token
}

func GenerateAuthToken() string {
	token, _ := GenerateRandomString(2048, lettersForToken)
	return token
}

func GetCallOptionsFromCtx(c *fiber.Ctx, pathPrefix string) BlobCallOptions {
	opt := BlobCallOptions{
		AppCode:     c.Params("appcode"),
		AuthToken:   c.Get("authorization"),
		ContentType: strings.ToUpper(c.Params("contentType")),
		Key:         c.Params("key"),
	}
	opt.Path = pathPrefix + "-" + opt.AppCode + "-" + opt.ContentType
	return opt
}

var DataStores map[string]gokv.Store

func GetStore(opt BlobCallOptions) gokv.Store {

	if DataStores[opt.AppCode] == nil {
		options := leveldb.Options{
			Path:  opt.Path,
			Codec: encoding.Gob,
		}

		client, err := leveldb.NewStore(options)
		DataStores[opt.AppCode] = client

		if err != nil {
			panic("Failed to connect to storage system.")
		}
	}

	return DataStores[opt.AppCode]
}

var AppAuthStore gokv.Store

func GetAppAuthStore() gokv.Store {
	options := leveldb.Options{
		Path:  "./.data/app-registry",
		Codec: encoding.Gob,
	}
	if AppAuthStore == nil {
		AppAuthStore, _ = leveldb.NewStore(options)
	}

	return AppAuthStore
}

func GetVal(opt BlobCallOptions, v interface{}) error {
	GetStore(opt).Get(opt.Key, &v)
	return nil
}

func SetVal(opt BlobCallOptions, v *interface{}) error {
	return GetStore(opt).Set(opt.Key, v)
}

func Run() {
	config := fiber.Config{
		Prefork:      true,
		ServerHeader: "Honu.Works/services/backplane", // add custom server header
	}

	pathPrefix := "./.data/storage/"

	service := fiber.New(config)
	service.Use(recover.New())
	service.Use(requestid.New())
	service.Use(logger.New())
	service.Use(etag.New())
	service.Use(compress.New())

	service.Use(basicauth.New(basicauth.Config{
		Next: func(c *fiber.Ctx) bool {
			Bootstrap()
			if !strings.HasPrefix(c.Path(), "/api") {
				return true
			}
			return false
		},
		Authorizer: func(user string, pass string) bool {
			appCreds := new(AppCreds)
			if len(user) > 0 && len(pass) > 0 {
				GetAppAuthStore().Get(user, &appCreds)
				if pass == appCreds.APIKey {
					return true
				}
			}

			return false
		},
	}))

	api := fiber.New(config)
	api.Use(recover.New())

	api.Use(basicauth.New(basicauth.Config{
		Next: func(c *fiber.Ctx) bool {
			Bootstrap()
			if strings.HasPrefix(c.Path(), "/api/app/new") {
				return true
			}
			return false
		},
		Authorizer: func(user string, pass string) bool {
			appCreds := new(AppCreds)
			if len(user) > 0 && len(pass) > 0 {
				GetAppAuthStore().Get(user, &appCreds)
				if pass == appCreds.APIKey {
					return true
				}
			}

			return false
		},
	}))

	api.Get("/api/app/new", func(c *fiber.Ctx) error {
		appcode := GenerateId("app", 32)
		token := GenerateAuthToken()

		appData := AppCreds{
			AppCode:      appcode,
			APIKey:       token,
			Title:        petname.Generate(3, " "),
			ContentTypes: []string{"user"},
		}

		GetAppAuthStore().Set(appcode, appData)
		return c.JSON(appData)
	})

	api.Get("/api/app/code/gen/token/:appcode", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	api.Get("/api/storage/:appcode/:contentType/:key", func(c *fiber.Ctx) error {
		opt := GetCallOptionsFromCtx(c, pathPrefix)
		v := new(interface{})
		GetVal(opt, &v)
		return c.JSON(v)
	})

	api.Post("/api/storage/:appcode/:contentType/:key", func(c *fiber.Ctx) error {
		opt := GetCallOptionsFromCtx(c, pathPrefix)
		v := new(interface{})
		if c.BodyParser(&v) == nil {
			SetVal(opt, v)
			return c.SendStatus(200)
		}

		return c.SendStatus(404)
	})

	/*--------------------------------------------------------------------------------

		Context:   The workos SDK for Go seems to force one set of creds globally.
		TODO:      Make this a separate program / process call

	----------------------------------------------------------------------------------*/

	api.Get("/api/auth/sign-up/:username/:email", func(c *fiber.Ctx) error {
		return nil
	})

	api.Get("/api/auth/sign-in/start/:username/:email", func(c *fiber.Ctx) error {
		return nil
	})

	api.Post("/api/auth/sign-in/complete/:code", func(c *fiber.Ctx) error {
		return nil
	})

	// API Proxy
	service.All("/api/**/*", proxy.Balancer(proxy.Config{
		Servers: []string{
			"http://localhost:" + CONFIG.API_PORT,
		},
	}))

	// APP Proxy
	service.All("/*", proxy.Balancer(proxy.Config{
		Servers: []string{
			"http://localhost:" + CONFIG.APP_PORT,
		},
	}))

	go api.Listen(":" + CONFIG.API_PORT)
	service.Listen(":" + CONFIG.PORT)
}
