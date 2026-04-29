package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"gophkeeper/client/internal/adapters/encryption"
	"gophkeeper/client/internal/adapters/rest"
	"gophkeeper/client/internal/adapters/storage"
	"gophkeeper/client/internal/domain"
	"os"
	"strings"
)

type AppContext struct {
	token      string
	userID     string
	email      string
	salt       []byte
	storage    *storage.FileStorage
	keyManager *encryption.KeyManager
	restClient *rest.Client
}

var app *AppContext

// RootCmd
var RootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "GophKeeper - secure password and secret manager",
	Long:  "GophKeeper is a secure, end-to-end encrypted password manager.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("GophKeeper CLI")
		fmt.Println("Use --help to see available commands")
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}

func init() {

	// Инициализация приложения
	fileStorage, err := storage.NewFileStorage()
	if err != nil {
		fmt.Printf("Warning: storage init failed: %v\n", err)
		fileStorage = nil
	}

	app = &AppContext{
		storage:    fileStorage,
		keyManager: encryption.NewKeyManager(),
		restClient: rest.NewClient("http://localhost:8080"),
	}

	// Загружаем сохранённую сессию
	if fileStorage != nil {
		saved, err := fileStorage.Load()
		if err != nil {
			fmt.Printf("Warning: loading session: %v\n", err)
		}

		if saved != nil && saved.Token != "" {
			app.token = saved.Token
			app.userID = saved.UserID
			app.email = saved.Email
			app.restClient.SetToken(saved.Token)

			if saved.Salt != "" {
				salt, err := encryption.StringToSalt(saved.Salt)
				if err == nil {
					app.salt = salt
				}
			}

			fmt.Printf("Welcome back, %s!\n", app.email)
		}
	}

	// Login flags
	loginCmd.Flags().StringP("email", "e", "", "Email address")
	loginCmd.Flags().StringP("password", "p", "", "Password")
	loginCmd.MarkFlagRequired("email")
	loginCmd.MarkFlagRequired("password")

	// Register flags
	registerCmd.Flags().StringP("email", "e", "", "Email address")
	registerCmd.Flags().StringP("password", "p", "", "Password")
	registerCmd.MarkFlagRequired("email")
	registerCmd.MarkFlagRequired("password")

	// Secret add flags
	secretAddCmd.Flags().StringP("title", "t", "", "Title of the secret")
	secretAddCmd.Flags().StringP("type", "y", "password", "Type of secret")
	secretGetCmd.Flags().StringP("id", "i", "", "Secret ID")
	secretGetCmd.MarkFlagRequired("id")
	secretAddCmd.MarkFlagRequired("title")

	// Commands
	RootCmd.AddCommand(loginCmd)
	RootCmd.AddCommand(registerCmd)
	RootCmd.AddCommand(logoutCmd)

	// Secret group
	secretCmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage secrets",
	}
	secretCmd.AddCommand(secretAddCmd, secretListCmd, secretGetCmd)
	RootCmd.AddCommand(secretCmd)
}

// loginCmd
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to GophKeeper",
	Run: func(cmd *cobra.Command, args []string) {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" || password == "" {
			fmt.Println("Error: --email and --password are required")
			return
		}

		resp, err := app.restClient.Login(map[string]string{
			"email": email, "password": password,
		})
		if err != nil {
			fmt.Printf("Login failed: %v\n", err)
			return
		}

		if len(app.salt) == 0 {
			salt, err := app.keyManager.GenerateSalt()
			if err != nil {
				fmt.Printf("Error generating salt: %v\n", err)
				return
			}
			app.salt = salt
		}

		// Выводим ключ
		key, err := app.keyManager.DeriveKeyFromMasterPassword(password, app.salt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Key length: %d bytes\n", len(key))

		// Сохраняем токен и метаданные
		err = app.storage.Save(&storage.StoredCredentials{
			Token:  resp.Token,
			UserID: resp.UserID,
			Email:  email,
			Salt:   encryption.SaltToString(app.salt),
		})
		if err != nil {
			fmt.Printf("Warning: couldn't save session: %v\n", err)
		}

		app.token = resp.Token
		app.userID = resp.UserID
		app.email = email
		app.restClient.SetToken(resp.Token)

		fmt.Println("Login successful! Session saved.")
	},
}

// registerCmd
var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register new user",
	Run: func(cmd *cobra.Command, args []string) {
		email, _ := cmd.Flags().GetString("email")
		password, _ := cmd.Flags().GetString("password")

		if email == "" || password == "" {
			fmt.Println("Error: --email and --password are required")
			cmd.Usage()
			return
		}

		fmt.Printf("Registering user with email: %s\n", email)

		err := app.restClient.Register(email, password)
		if err != nil {
			fmt.Printf("Registration failed: %v\n", err)
			return
		}

		fmt.Println("Registration successful. You can now login!")
	},
}

// secretAddCmd
var secretAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new secret",
	Run: func(cmd *cobra.Command, args []string) {
		title, _ := cmd.Flags().GetString("title")
		secretType, _ := cmd.Flags().GetString("type")

		if title == "" {
			fmt.Println("Error: --title is required")
			return
		}

		if app.token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}

		// Принудительно устанавливаем токен
		app.restClient.SetToken(app.token)

		if len(app.salt) == 0 {
			fmt.Println("Error: Session corrupted. Please login again")
		}

		// Мастер-пароль
		fmt.Print("Enter master password: ")
		masterPass, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		masterPass = strings.TrimSpace(masterPass)

		if masterPass == "" {
			fmt.Println(domain.ErrMasterPasswordRequired)
			return
		}

		// Данные секрета
		fmt.Print("Enter secret data (login:password or note text): ")
		data, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		data = strings.TrimSpace(data)

		if data == "" {
			fmt.Println(domain.ErrInvalidInput)
			return
		}

		// Шифрование
		key, err := app.keyManager.DeriveKeyFromMasterPassword(masterPass, app.salt)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		encrypted, err := app.keyManager.Encrypt([]byte(data), key)
		if err != nil {
			fmt.Printf("Encryption failed: %v\n", err)
			return
		}

		encryptedBase64 := base64.StdEncoding.EncodeToString(encrypted)

		secretID, err := app.restClient.CreateSecret(title, secretType, string(encryptedBase64))
		if err != nil {
			fmt.Printf("Server error: %v\n", err)
			return
		}

		fmt.Printf(" Secret added successfully!\n")
		fmt.Printf("   ID: %s\n", secretID)
		fmt.Printf("   Title: %s\n", title)
	},
}

// secretListCmd
var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all your secrets",
	Long:  "List all your secrets",
	Run: func(cmd *cobra.Command, args []string) {
		if app.token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}
		fmt.Println("Fetching your secrets from server...")

		secrets, err := app.restClient.GetSecrets()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		if len(secrets) == 0 {
			fmt.Println("No secrets found.")
			return
		}

		fmt.Println("\n Your secrets: ")
		for i, s := range secrets {
			title := s["title"]
			secretType := s["type"]
			createdAt := s["created_at"]
			fmt.Printf("%d. %s (type: %s) created: %v\n", i+1, title, secretType, createdAt)
		}

	},
}

// secretGetCmd - получение и расшифровка секрета
var secretGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get and decrypt a secret by ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")

		if id == "" {
			fmt.Println("Error: --id is required")
			return
		}

		if app.token == "" {
			fmt.Println("Error: Not logged in. Run: gophkeeper login")
			return
		}

		if len(app.salt) == 0 {
			fmt.Println("Error: No salt in session. Login again.")
			return
		}

		// Мастер-пароль
		fmt.Print("Enter master password: ")
		masterPass, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		masterPass = strings.TrimSpace(masterPass)

		if masterPass == "" {
			fmt.Println(domain.ErrMasterPasswordRequired)
			return
		}

		// Получаем секрет с сервера
		secret, err := app.restClient.GetSecret(id)
		if err != nil {
			fmt.Printf("Error fetching secret: %v\n", err)
			return
		}

		// Извлекаем зашифрованные данные (приходит как строка)
		encryptedStr, ok := secret["data"].(string)
		if !ok {
			fmt.Println("Error: no data field")
			return
		}

		// Декодируем base64 в []byte
		encryptedData, err := base64.StdEncoding.DecodeString(encryptedStr)
		if err != nil {
			fmt.Printf("Base64 decode error: %v\n", err)
			return
		}

		// Генерируем ключ
		key, err := app.keyManager.DeriveKeyFromMasterPassword(masterPass, app.salt)
		if err != nil {
			fmt.Printf("Key error: %v\n", err)
			return
		}

		// Расшифровываем
		decrypted, err := app.keyManager.Decrypt(encryptedData, key)
		if err != nil {
			fmt.Printf("Decryption failed: %v\n", err)
			return
		}

		fmt.Printf("Decrypted data: %s\n", string(decrypted))

		// Выводим результат
		fmt.Println("\n Secret Detail")
		fmt.Printf("Title: %v\n", secret["title"])
		fmt.Printf("Type: %v\n", secret["type"])
		fmt.Printf("Data: %s\n", string(decrypted))
		fmt.Printf("Created: %v\n", secret["created_at"])
	},
}

// logoutCmd
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear saved session",
	Run: func(cmd *cobra.Command, args []string) {
		if err := app.storage.Clear(); err != nil {
			fmt.Printf("Error clearing session: %v\n", err)
			return
		}
		app.token = ""
		app.userID = ""
		app.email = ""
		app.salt = nil
		app.restClient.SetToken("")
		fmt.Println("Logged out. Session cleared.")
	},
}
