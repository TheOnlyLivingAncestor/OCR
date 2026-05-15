package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/mail"
	"net/smtp"
	"ocr/packages/queue"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

var Emails = []string{"vitez.alexandra@icloud.com"}
var SMTPPassword = "rfbi hdlb jenz dftr"
var SMTPUser = "vitez.alexandra@gmail.com"
var SMTPAddr = "smtp.gmail.com"
var SMTPPort = "587"

var RabbitAddr = "hello-world.rabbitmq-cluster.svc.cluster.local"
var RabbitPort = "5672"
var RabbitConsumerQueue = "recognition-notification"
var RabbitUser = "guest"
var RabbitPassword = "guest"

func valid_email(s string) bool {
	_, err := mail.ParseAddress(s)
	if err != nil {
		return true
	}
	return false
}

func read_rabbitmq() {
	//Read rabbitmq variables from environment variables
	if os.Getenv("RABBIT_CONSUMER_QUEUE") != "" {
		logger.Info("RABBIT_CONSUMER_QUEUE environment variable is set, using that", "value", os.Getenv("RABBIT_CONSUMER_QUEUE"))
		RabbitConsumerQueue = os.Getenv("RABBIT_CONSUMER_QUEUE")
	} else {
		logger.Info("RABBIT_CONSUMER_QUEUE environment variable is not set, using default", "value", RabbitConsumerQueue)
	}

	if os.Getenv("RABBIT_ADDR") != "" {
		logger.Info("RABBIT_ADDR environment variable is set, using that", "value", os.Getenv("RABBIT_ADDR"))
		RabbitAddr = os.Getenv("RABBIT_ADDR")
	} else {
		logger.Info("RABBIT_ADDR environment variable is not set, using default", "value", RabbitAddr)
	}

	if os.Getenv("RABBIT_USER") != "" {
		logger.Info("RABBIT_USER environment variable is set, using that", "value", os.Getenv("RABBIT_USER"))
		RabbitUser = os.Getenv("RABBIT_USER")
	} else {
		logger.Info("RABBIT_USER environment variable is not set, using default", "value", RabbitUser)
	}

	if os.Getenv("RABBIT_PASSWORD") != "" {
		logger.Info("RABBIT_PASSWORD environment variable is set, using that", "value", os.Getenv("RABBIT_PASSWORD"))
		RabbitPassword = os.Getenv("RABBIT_PASSWORD")
	} else {
		logger.Info("RABBIT_PASSWORD environment variable is not set, using default", "value", RabbitPassword)
	}
	if os.Getenv("RABBIT_PORT") != "" {
		logger.Info("RABBIT_PORT environment variable is set, using that", "value", os.Getenv("RABBIT_PORT"))
		RabbitPort = os.Getenv("RABBIT_PORT")
	} else {
		logger.Info("RABBIT_PORT environment variable is not set, using default", "value", RabbitPort)
	}
}

func read_emails() {

	if os.Getenv("SMTP_PASSWORD") != "" {
		logger.Info("SMTP_PASSWORD environment variable is set, using that", "value", os.Getenv("SMTP_PASSWORD"))
		SMTPPassword = os.Getenv("SMTP_PASSWORD")
	} else {
		logger.Info("Using default SMTP password", "value", SMTPPassword)
	}

	if os.Getenv("SMTP_USER") != "" {
		logger.Info("SMTP_USER environment variable is set, using that", "value", os.Getenv("SMTP_USER"))
		SMTPUser = os.Getenv("SMTP_USER")
	} else {
		logger.Info("Using default SMTP username", "value", SMTPUser)
	}

	if os.Getenv("SMTP_ADDR") != "" {
		logger.Info("SMTP_ADDR environment variable is set, using that", "value", os.Getenv("SMTP_ADDR"))
		SMTPAddr = os.Getenv("SMTP_ADDR")
	} else {
		logger.Info("Using default SMTP address", "value", SMTPAddr)
	}

	if os.Getenv("SMTP_PORT") != "" {
		logger.Info("SMTP_PORT environment variable is set, using that", "value", os.Getenv("SMTP_PORT"))
		SMTPPort = os.Getenv("SMTP_PORT")
	} else {
		logger.Info("Using default SMTP port", "value", SMTPPort)
	}

	//Prio1: Környezeti változóból beolvassuk az email címeket
	if os.Getenv("RECIPIENTS") != "" {
		logger.Info("RECIPIENTS environment variable is set", "value", os.Getenv("RECIPIENTS"))
		//Vesszővel elválasztott stringek, amiket fel kell darabolni
		env_emails := strings.SplitN(os.Getenv("RECIPIENTS"), ",", -1)
		//Levalidáljuk, hogy valós email címek-e
		if valid_emails := slices.DeleteFunc(env_emails, valid_email); valid_emails != nil {
			//Ha sikeresen beparszoltunk valid email-okat, akkor felülcsapjuk a defaultot
			logger.Info("Valid emails were read from environment, using those", "value", valid_emails)
			Emails = valid_emails
			return
		}
	}
	logger.Info("RECIPIENTS environment variable is empty, falling back to use default emails")
	//Prio2: Ha nincs env variable, vagy nem olvastunk be valid email-okat, akkor a default értéket használjuk
	if Emails != nil {
		logger.Info("Using default emails", "value", Emails)
		if Emails = slices.DeleteFunc(Emails, valid_email); Emails != nil {
			logger.Info("There are still default values after validation, using those", "value", Emails)
			return
		}
	}
	//Ha eddig eljutottunk, akkor nincs valid email
	logger.Error("There were no valid emails found, exiting")
	os.Exit(1)

}

func send_email(msg []byte) {
	conn, err := smtp.Dial(SMTPAddr + ":" + SMTPPort)
	if err != nil {
		logger.Error("Dial failed", "error", err)
		panic(err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			logger.Error("Failed to close connection", "error", err)
			panic(err)
		}
	}()

	if ok, _ := conn.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: SMTPAddr}
		if err = conn.StartTLS(config); err != nil {
			logger.Error("Failed to extension", "error", err)
			panic(err)
		}
	}

	auth := smtp.PlainAuth("", SMTPUser, SMTPPassword, SMTPAddr)

	if err = conn.Auth(auth); err != nil {
		logger.Error("Failed to auth", "error", err)
		panic(err)
	}

	if err = conn.Mail(SMTPUser); err != nil {
		logger.Error("Failed to mail", "error", err)
		panic(err)
	}

	for _, addr := range Emails {
		if err = conn.Rcpt(addr); err != nil {
			logger.Error("Failed to rcpt", "error", err)
			panic(err)
		}
	}

	w, err := conn.Data()
	if err != nil {
		logger.Error("Failed to data", "error", err)
		panic(err)
	}

	_, err = w.Write(msg)
	if err != nil {
		logger.Error("Failed to write", "error", err)
		panic(err)
	}

	if err = w.Close(); err != nil {
		logger.Error("Failed to close writer", "error", err)
		panic(err)
	}
	if err = conn.Quit(); err != nil {
		logger.Error("Failed to quit connection", "error", err)
		panic(err)
	}

}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	//This handler should only accept GET requests
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	json, err := json.Marshal("OK")
	if err != nil {
		logger.Info("Failed to marshal response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		logger.Info("API request failed", "error", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json)
	if err != nil {
		logger.Info("Failed to write response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func startServer(s *http.Server, logger *slog.Logger) {
	logger.Info("Server listening on http://:8080")
	err := s.ListenAndServe()
	// http.ErrServerClosed should not be logged to the user
	if err != nil && err != http.ErrServerClosed {
		logger.Error("HTTP server error", "error", err)
	}
}

func create_email(description string, results []string) []byte {
	toHeader := strings.Join(Emails, ", ")
	string_results := strings.Join(results, ", ")
	msg := []byte(
		"To: " + toHeader + "\r\n" +
			"Subject: OCR Recognition was successful\r\n" +
			"\r\n" +
			"The OCR recognition was successful with the following information:\r\n" +
			"Description of image: " + description + "\r\n" +
			"Result of recognition: " + string_results + "\r\n",
	)
	return msg
}

func rmq_consumer_loop(ctx context.Context, rmq queue.Queue) {
consumer_loop:
	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutdown signal received in consumer loop")
			break consumer_loop
		default:
			deliveryctx, err := rmq.GetMessage()
			if err != nil {
				logger.Error("Failed to receive message from RabbitMQ", "error", err)
				continue
			}
			msg := deliveryctx.Message().GetData()
			logger.Info("Received message from RabbitMQ", "data", msg)
			var msg_json queue.RmqSuccess
			if err = json.Unmarshal(msg, &msg_json); err != nil {
				logger.Error("Failed to unmarshal message", "error", err)
				//Nem tudjuk processzálni az üzenetet -> nagy valséggel hibás, discard?
				err = deliveryctx.Discard(context.Background(), nil)
				if err != nil {
					logger.Error("Failed to discard message", "error", err)
				}
				continue
			}
			logger.Info("Unmarshaled message from RabbitMQ", "msg", msg_json)
			//Elküldjük az emailt
			message := create_email(msg_json.Description, msg_json.Results)
			send_email(message)
			//Sikeres email elküldése után pedig Ack-oljuk az üzenetet
			err = deliveryctx.Accept(context.Background())
			if err != nil {
				logger.Error("Error during accepting the message", "error", err)
			}
		}
	}
}

func main() {
	read_emails()
	read_rabbitmq()

	//Initialize RabbitMQ
	rmq, err := queue.NewRabbitMQ("amqp://" + RabbitUser + ":" + RabbitPassword + "@" + RabbitAddr + ":" + RabbitPort + "/")
	if err != nil {
		logger.Error("Failed to create RabbitMQ environment, exiting", "error", err)
		panic(err)
	}

	if err = rmq.CreateConnection(); err != nil {
		logger.Error("Failed to create RabbitMQ connection, exiting", "error", err)
		panic(err)
	}

	if err = rmq.CreateConsumer(RabbitConsumerQueue); err != nil {
		logger.Error("Failed to create consumer, exiting", "error", err)
		panic(err)
	}

	defer func() {
		if err = rmq.CloseAll(); err != nil {
			logger.Error("Failed to close RabbitMQ components", "error", err)
			panic(err)
		}
	}()

	http.Handle("/healthz", http.HandlerFunc(healthzHandler))

	rmq_ctx, rmq_cancel := context.WithCancel(context.Background())
	defer rmq_cancel()

	go rmq_consumer_loop(rmq_ctx, rmq)

	//HTTP server starts in a goroutine to handle graceful shutdown
	s := &http.Server{Addr: ":8080"}
	// Start the server in the goroutine
	go startServer(s, logger)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received")
	rmq_cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err = s.Shutdown(ctx)
	if err != nil {
		logger.Info("Graceful server shutdown failed with", "error", err)
	} else {
		logger.Info("Graceful server sutdown succeeded")
	}

}
