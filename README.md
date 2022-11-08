# otp-golang
This is an OTP Auth package for GOLang. 

<img width="539" alt="Screen Shot 2022-11-08 at 22 44 13" src="https://user-images.githubusercontent.com/46659611/200660217-ec4f1afa-a859-4baf-9a45-7967798af70a.png">

⭐️ This package is still in development. I welcome suggestions for changes that will bring it closer to compliance without overly complicating the code, or useful test cases to add to the test suite.

## ⚡️ Quick Start
```
go get -u https://github.com/yahya077/otp-golang
```
## 📖 Prebuild Handlers
Note: There is an available documentation with *[Postman API Documentation](https://documenter.getpostman.com/view/10956074/2s8YehSvcG)* 
<table class="table">
  <thead>
    <tr>
      <th>Title</th>
      <th>Route</th>
      <th>Method</th>
      <th>Handler</th>
      <th>Middleware</th>
      <th>Customizable</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>OTP</td>
      <td>/auth/otp</td>
      <td>POST</td>
      <td>OtpHandler</td>
      <td>none</td>
      <td>✓</td>
      <td>Sends OTP to user</td>
    </tr>
    <tr>
      <td>Login</td>
      <td>/auth/login</td>
      <td>POST</td>
      <td>LoginHandler</td>
      <td>none</td>
      <td>✓</td>
      <td>Returns Bearer token</td>
    </tr>
    <tr>
      <td>Register</td>
      <td>/auth/register</td>
      <td>POST</td>
      <td>RegisterHandler</td>
      <td>AuthMiddleware</td>
      <td>✓</td>
      <td>Registers user by User Repository</td>
    </tr>
  </tbody>
</table>

## 💡 Usage
Note: Example usage on *[otp-golang-example](https://github.com/yahya077/otp-golang-example)*

```
// create instance off Auth
authApp := auth.New(app, database.Postgresql.DB, auth.Config{})

// set sms provider for sending otp code to phone number
authApp.SetSmsProvider(smsProvider.MockedSmsSenderProviderClient{})

// set your custom user model as a subject for verify, insert and register
authApp.SetUserRepository(repository.UserRepository{})

// finally initialize the auth app
authApp.Initialize()

// You can use middleware as shown below
app.Get("/test-auth", authApp.Config.AuthMiddleware, func(ctx *fiber.Ctx) error {
		return ctx.JSON("authorized")
})
  
```

## 🧬  Auth Middleware

This middleware will be ready after initialization of Auth. You can override Middleware if you want to

<table class="table">
    <thead>
    <tr>
        <th rowspan="3">Middleware</th>
        <th rowspan="3">Description</th>
    </tr>
    </thead>
    <tbody>
    <tr>
        <td>
        authApp.Config.AuthMiddleware
        </td>
        <td>
        Looks for <b>Bearer</b> token
        </td>
    </tr>
    </tbody>
</table>

## 📫&nbsp; Have a question? Want to chat? Ran into a problem?

#### *Website [yahyahindioglu.com](https://yahyahindioglu.com)*

#### *[LinkedIn](https://www.linkedin.com/in/yahyahindioglu/) Account*
