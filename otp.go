package otp_golang

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	PathAuth     = "/auth"
	PathRegister = "/register"
	PathLogin    = "/login"
	PathOtp      = "/otp"
)

type Auth struct {
	App    *fiber.App
	DB     *gorm.DB
	Config Config
}

type Config struct {
	OtpHandler        fiber.Handler
	LoginHandler      fiber.Handler
	RegisterHandler   fiber.Handler
	AuthMiddleware    func(c *fiber.Ctx) error
	SmsProvider       ISmsProvider
	UserRepository    IUserRepository
	OtpCodeRepository OtpCodeRepository
	SendOtp           func(phone string, code string) error
}

type OtpModel struct {
	Phone string `json:"phone"`
}

type OtpBaseUserModel struct {
	Phone string `json:"phone"`
}

type OtpCode struct {
	ID        uint   `gorm:"primarykey"`
	Phone     string `json:"phone"`
	Code      string `json:"code"`
	ExpiredAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (o OtpCode) IsExpired() bool {
	return !time.Now().Before(o.ExpiredAt)
}

type OtpCheckerResponse struct {
	Authenticated bool
	Registered    bool
	Phone         string
	Expiration    time.Time
}

type Router struct {
	LoginPrefix    string
	RegisterPrefix string
}

type HeaderBearer struct {
	Authorization string `reqHeader:"Authorization"`
}

type ISmsProvider interface {
	SendOtp(phone string, code string) error
}

type IUserRepository interface {
	VerifyOtp(phone string, code string) OtpCheckerResponse
	Register(parser func(interface{}) error) error
	Registered(phone string) bool
}

func New(app *fiber.App, db *gorm.DB, config Config) *Auth {
	auth := &Auth{
		App:    app,
		DB:     db,
		Config: config,
	}
	auth.Initialize()
	return auth
}

func (a *Auth) Initialize() {
	if a.Config.OtpHandler == nil {
		a.Config.OtpHandler = a.otpHandler
	}
	if a.Config.LoginHandler == nil {
		a.Config.LoginHandler = a.loginHandler
	}
	if a.Config.RegisterHandler == nil {
		a.Config.RegisterHandler = a.registerHandler
	}
	if a.Config.AuthMiddleware == nil {
		a.Config.AuthMiddleware = a.authMiddleware
	}
	a.Config.OtpCodeRepository.DB = a.DB
	a.SetRoutes()
}

func (a *Auth) SetRoutes() {
	authRouter := a.App.Group("/auth")
	authRouter.Post("/otp", a.Config.OtpHandler)
	authRouter.Post("/login", a.Config.LoginHandler)
	authRouter.Post("/register", a.authMiddleware, a.Config.RegisterHandler)
}

func (a *Auth) SetSmsProvider(provider ISmsProvider) {
	a.Config.SmsProvider = provider
}

func (a *Auth) SetUserRepository(subject IUserRepository) {
	a.Config.UserRepository = subject
}

func (a *Auth) SetOtpSender() {
	authRouter := a.App.Group(PathAuth)
	authRouter.Post(PathOtp, a.Config.OtpHandler)
	authRouter.Post(PathLogin, a.Config.LoginHandler)
	authRouter.Post(PathRegister, a.Config.RegisterHandler)
}

func (a *Auth) loginHandler(c *fiber.Ctx) error {
	var otpCheckerResponse OtpCheckerResponse

	phone := c.FormValue("phone")
	code := c.FormValue("code")

	valid := a.Config.OtpCodeRepository.Validate(phone, code)

	if otpCheckerResponse.Authenticated = valid; otpCheckerResponse.Authenticated {

		otpCheckerResponse.Registered = a.Config.UserRepository.Registered(phone)

		claims := jwt.MapClaims{
			"registered": otpCheckerResponse.Registered,
			"otp":        code,
			"phone":      phone,
			"exp":        otpCheckerResponse.Expiration,
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET_KEY")))

		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.JSON(fiber.Map{
			"token":      t,
			"phone":      phone,
			"registered": otpCheckerResponse.Registered,
			"expiration": otpCheckerResponse.Expiration,
		})
	}

	return c.Status(422).JSON("Otp Code Is Wrong")
}

// registerHandler function only stands for setting columns to user table
func (a *Auth) registerHandler(c *fiber.Ctx) error {

	e := a.Config.UserRepository.Register(c.BodyParser)

	if e != nil {
		return c.Status(422).JSON("Something went wrong")
	}

	return c.JSON("registered user")
}

// otpHandler will be creating and sending otp code
func (a *Auth) otpHandler(c *fiber.Ctx) error {
	phone := c.FormValue("phone")

	otpCode := createOtpCode()

	a.Config.OtpCodeRepository.Insert(phone, otpCode, time.Now().Add(time.Minute*2))

	e := a.Config.SmsProvider.SendOtp(phone, otpCode)

	if e != nil {
		fmt.Println(e.Error())
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON("Code Sent")
}

func (a *Auth) GetRegisterPath() string {
	return PathAuth + PathRegister
}

func (a *Auth) GetLoginPath() string {
	return PathAuth + PathLogin
}

func (a *Auth) GetOtpPath() string {
	return PathAuth + PathOtp
}

func (a *Auth) authMiddleware(c *fiber.Ctx) error {
	var authMiddlewareHandler AuthMiddlewareHandler
	c.ReqHeaderParser(&authMiddlewareHandler.Header)

	if authMiddlewareHandler.HasBearer() {
		authMiddlewareHandler.ParseToken()

		if claims, ok := authMiddlewareHandler.GetMappedClaims(); ok {
			if c.Path() == a.GetRegisterPath() && claims["registered"].(bool) {
				return c.SendStatus(fiber.StatusForbidden)
			}

			if !authMiddlewareHandler.IsTokenExpired() {
				c.Next()
			}
		}
	}
	return c.SendStatus(fiber.StatusUnauthorized)
}

// createOtpCode is a helper for creating six digits otp codes
func createOtpCode() string {
	min := 100000
	max := 999999

	rand.Seed(time.Now().UnixNano())

	return strconv.Itoa(rand.Intn(max-min) + min)
}

type AuthMiddlewareHandler struct {
	Header HeaderBearer
	Token  *jwt.Token
	Claims jwt.MapClaims
}

func (a *AuthMiddlewareHandler) HasBearer() bool {
	return strings.Contains(a.Header.Authorization, "Bearer")
}

func (a *AuthMiddlewareHandler) GetTokenString() string {
	return strings.Replace(a.Header.Authorization, "Bearer ", "", -1)
}

func (a *AuthMiddlewareHandler) ParseToken() (*jwt.Token, error) {
	var e error
	a.Token, e = jwt.Parse(a.GetTokenString(), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})
	return a.Token, e
}

func (a *AuthMiddlewareHandler) IsTokenExpired() bool {
	return !time.Now().Before(a.Claims["exp"].(time.Time))
}

func (a *AuthMiddlewareHandler) GetMappedClaims() (jwt.MapClaims, bool) {
	var ok bool
	a.Claims, ok = a.Token.Claims.(jwt.MapClaims)
	ok = a.Token.Valid
	return a.Claims, ok
}

type GormRepository struct {
	DB *gorm.DB
}

type OtpCodeRepository struct {
	GormRepository
	OtpCode OtpCode
}

func (repository OtpCodeRepository) Validate(phone string, otp string) bool {
	result := repository.DB.Where("phone = ? AND code = ?", phone, otp).Last(&repository.OtpCode)
	return result.Error == nil && time.Now().Before(repository.OtpCode.ExpiredAt)
}

func (repository OtpCodeRepository) Insert(phone string, code string, expiredAt time.Time) {
	repository.OtpCode.Phone = phone
	repository.OtpCode.Code = code
	repository.OtpCode.ExpiredAt = expiredAt

	repository.DB.Create(&repository.OtpCode)
}
