package otp_golang

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const (
	PathAuth     = "/auth"
	PathRegister = "/register"
	PathLogin    = "/login"
	PathOtp      = "/otp"
	PathGetUser  = "/user"
	LocalUser    = "user_model"
	LocalClaims  = "claims"
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
	GetUserHandler    fiber.Handler
	AuthMiddleware    fiber.Handler
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
	Register(parser func(interface{}) error) error
	Registered(phone string) bool
	FindByPhone(phone string) (interface{}, error)
}

func New(app *fiber.App, db *gorm.DB, config Config) *Auth {
	auth := &Auth{
		App:    app,
		DB:     db,
		Config: config,
	}
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
		a.Config.AuthMiddleware = AuthMiddleware
	}
	if a.Config.GetUserHandler == nil {
		a.Config.GetUserHandler = a.getUserHandler
	}
	a.Config.OtpCodeRepository.DB = a.DB
	a.SetRoutes()
}

func (a *Auth) SetRoutes() {
	authRouter := a.App.Group("/auth")
	authRouter.Post(PathOtp, a.Config.OtpHandler)
	authRouter.Post(PathLogin, a.Config.LoginHandler)
	authRouter.Post(PathRegister, a.Config.AuthMiddleware, a.Config.RegisterHandler)
	authRouter.Get(PathGetUser, a.Config.AuthMiddleware, a.Config.GetUserHandler)
}

func (a *Auth) SetSmsProvider(provider ISmsProvider) {
	a.Config.SmsProvider = provider
}

func (a *Auth) SetUserRepository(subject IUserRepository) {
	a.Config.UserRepository = subject
}

func (a *Auth) loginHandler(c *fiber.Ctx) error {
	var otpCheckerResponse OtpCheckerResponse

	phone := c.FormValue("phone")
	code := c.FormValue("code")

	otpCode, e := a.Config.OtpCodeRepository.Validate(phone, code)

	if otpCheckerResponse.Authenticated = e == nil; otpCheckerResponse.Authenticated {

		otpCheckerResponse.Registered = a.Config.UserRepository.Registered(phone)

		claims := jwt.MapClaims{
			"registered": otpCheckerResponse.Registered,
			"otp":        code,
			"phone":      phone,
			"exp":        otpCode.ExpiredAt.Unix(),
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

// getUserHandler gets user from db
func (a *Auth) getUserHandler(c *fiber.Ctx) error {
	claims := c.Locals("claims").(jwt.MapClaims)

	if userData, e := a.Config.UserRepository.FindByPhone(claims["phone"].(string)); e == nil {
		return c.JSON(userData)
	}
	return c.SendStatus(fiber.StatusNotFound)
}

// otpHandler will be creating and sending otp code
func (a *Auth) otpHandler(c *fiber.Ctx) error {
	phone := c.FormValue("phone")

	otpCode := createOtpCode()

	a.Config.OtpCodeRepository.Insert(phone, otpCode, time.Now().Add(time.Hour*72))

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

func (a *Auth) GetUserHandler(ctx *fiber.Ctx) error {
	claims := ctx.Locals(LocalClaims).(jwt.MapClaims)

	if user, err := a.Config.UserRepository.FindByPhone(claims["phone"].(string)); err == nil {
		ctx.Locals(LocalUser, user)
	}

	return ctx.SendStatus(fiber.StatusNotFound)
}

// createOtpCode is a helper for creating six digits otp codes
func createOtpCode() string {
	min := 100000
	max := 999999

	rand.Seed(time.Now().UnixNano())

	return strconv.Itoa(rand.Intn(max-min) + min)
}

type GormRepository struct {
	DB *gorm.DB
}

type OtpCodeRepository struct {
	GormRepository
	OtpCode OtpCode
}

func (repository OtpCodeRepository) Validate(phone string, otp string) (OtpCode, error) {
	var err error

	result := repository.DB.Where("phone = ? AND code = ?", phone, otp).Last(&repository.OtpCode)

	if result.Error != nil {
		err = errors.New("invalid Otp Code")
	}

	if time.Now().After(repository.OtpCode.ExpiredAt) {
		err = errors.New("otp code expired")
	}

	return repository.OtpCode, err
}

func (repository OtpCodeRepository) Insert(phone string, code string, expiredAt time.Time) {
	repository.OtpCode.Phone = phone
	repository.OtpCode.Code = code
	repository.OtpCode.ExpiredAt = expiredAt

	repository.DB.Create(&repository.OtpCode)
}
