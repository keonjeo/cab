package models

import (
	"github.com/jinzhu/gorm"
	"strings"
	u "citicab/utils"
	"golang.org/x/crypto/bcrypt"
	"fmt"
)

type Driver struct {

	gorm.Model
	Fullname string `json:"fullname"`
	Email  string `json:"email"`
	Phone string `json:"phone"`
	Password string `json:"password"`
	Status string `json:"status"`
	Verified bool `json:"verified"`
	CarName string `json:"car_name"`
	PlateNumber string `json:"plate_number"`
	Token string `gorm:"-" sql:"-" json:"token"`

}


func VerifyDriversPhone(phone string) (map[string] interface{}) {

	var count int
	err := Db.Table("drivers").Where("phone = ?", strings.TrimSpace(phone)).Count(&count).Error
	if err != nil {
		return u.Message(false, "Failed to perform operation. Please, retry")
	}
	if count > 0 {
		resp := u.Message(true, "success")
		resp["exists"] = true
		return resp
	}

	driver := &Driver{Phone: phone}
	err = Db.Create(driver).Error
	if err != nil {
		return u.Message(false, "Failed to create new account. Please, retry")
	}

	auth := CreateAuth(driver.ID)
	if auth != nil {
		text := fmt.Sprintf("Your CitiCab authentication code: %d", auth.Code)
		smsRequest := &SmsRequest{
			Text: text,
			Phone: strings.TrimSpace(phone),
		}

		SmsQueue <- smsRequest
	}

	token := GenJWT(driver.ID)
	resp := u.Message(true, "success")
	resp["exists"] = false
	resp["token"] = token
	return resp
}


func (driver *Driver) Update() (map[string] interface{}) {

	if len(driver.Password) > 0 {
		return u.Message(false, "Cannot update password")
	}

	err := Db.Table("drivers").Where("id = ?", driver.ID).UpdateColumn(driver).Error
	if err != nil {
		return u.Message(false, "Failed to update account. Please, retry")
	}

	mailRequest := &MailRequest{
		Subject: "Welcome to CitiCab",
		Body: "Hi, Welcome to our new shining platform, citicab",
		To: driver.Email,
	}

	MailQueue <- mailRequest

	updated := GetDriver(driver.ID)
	r := u.Message(true, "account updated")

	updated.Password = ""
	r["user"] = updated
	return r
}

func DriverLogin(driver *Driver) (map[string]interface{}) {

	temp := &Driver{}
	err := Db.Table("users").Where("phone = ?", driver.Phone).First(temp).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return u.Message(false, "User with phone " + driver.Phone + " not found")
		}

		return u.Message(false, "Failed to complete login request. Please, retry")
	}

	if temp.ID <= 0 {
		return u.Message(false, "User with phone " + driver.Phone + " not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(temp.Password), []byte(driver.Password))
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
		return u.Message(false, "Invalid login credentials")
	}

	driver.Password = ""
	driver.Token = GenJWT(temp.ID)
	r := u.Message(true, "success")
	r["driver"] = driver
	return r
}

func UpdateDriversPassword(driver *Driver) (map[string]interface{}) {

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(driver.Password), bcrypt.DefaultCost)
	driver.Password = string(hashedPassword)
	err := Db.Table("drivers").Where("id = ?", driver.ID).UpdateColumn(driver).Error
	if err != nil {
		return u.Message(false, "Failed to update password. Please, retry")
	}

	return u.Message(true, "Password updated")
}


func GetDriver(id uint) *Driver {

	driver := &Driver{}
	err := Db.Table("drivers").Where("id = ?", id).First(driver).Error
	if err != nil {
		return nil
	}

	return driver
}