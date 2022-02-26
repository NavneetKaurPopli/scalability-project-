package main

import (
	"context"
	"day-trading/txserver/event"
	"log"
	"time"
)

func logUserCommandEvent(
	ctx *context.Context, timestamp, transactionNum int64,
	server, command, username, stock, filename string, funds float32) {
	data := &event.UserCommand{
		Timestamp:      timestamp,
		Server:         server,
		TransactionNum: transactionNum,
		Command:        command,
		Username:       username,
		StockSymbol:    stock,
		Filename:       filename,
		Funds:          funds,
	}
	event := &event.Event{EventType: event.EventUserCommand, Data: data}
	insertEventToDB(ctx, event)
}

func logQuoteServerEvent(ctx *context.Context, timestamp, transactionNum int64,
	server, stock, username, cryptokey string, quoteServerTime int64) {
	data := &event.QuoteServer{
		Timestamp:       timestamp,
		Server:          server,
		TransactionNum:  transactionNum,
		StockSymbol:     stock,
		Username:        username,
		QuoteServerTime: timestamp,
		Cryptokey:       cryptokey,
	}
	event := &event.Event{EventType: event.EventQuoteServer, Data: data}
	insertEventToDB(ctx, event)
}

func logAccountTransactionEvent(ctx *context.Context, timestamp, transactionNum int64,
	server, action, username string, funds float32) {
	data := &event.AccountTransaction{
		Timestamp:      timestamp,
		Server:         server,
		TransactionNum: transactionNum,
		Action:         action,
		Username:       username,
		Funds:          funds,
	}
	event := &event.Event{EventType: event.EventAccountTransaction, Data: data}
	insertEventToDB(ctx, event)
}

func logSystemEvent(ctx *context.Context, timestamp, transactionNum int64,
	server, command, username, stock, filename string, funds float32) {
	data := &event.SystemEvent{
		Timestamp:      timestamp,
		Server:         server,
		TransactionNum: transactionNum,
		Command:        command,
		Username:       username,
		StockSymbol:    stock,
		Filename:       filename,
		Funds:          funds,
	}
	event := &event.Event{EventType: event.EventSystem, Data: data}
	insertEventToDB(ctx, event)
}

func logErrorEvent(ctx *context.Context, timestamp, transactionNum int64,
	server, command, username, stock, filename, errorMsg string, funds float32) {
	data := &event.ErrorEvent{
		Timestamp:      timestamp,
		Server:         server,
		TransactionNum: transactionNum,
		Command:        command,
		Username:       username,
		StockSymbol:    stock,
		Filename:       filename,
		ErrorMessage:   errorMsg,
		Funds:          funds,
	}
	event := &event.Event{EventType: event.EventError, Data: data}
	insertEventToDB(ctx, event)
}

func logDebugEvent(ctx *context.Context, timestamp, transactionNum int64,
	server, command, username, stock, filename, debugMsg string, funds float32) {
	data := &event.DebugEvent{
		Timestamp:      timestamp,
		Server:         server,
		TransactionNum: transactionNum,
		Command:        command,
		Username:       username,
		StockSymbol:    stock,
		Filename:       filename,
		DebugMessage:   debugMsg,
		Funds:          funds,
	}

	event := &event.Event{EventType: event.EventDebug, Data: data}
	insertEventToDB(ctx, event)
}

func insertEventToDB(ctx *context.Context, event *event.Event) {
	eventsCollection := client.Database("test").Collection("events")

	maxAttempts := 5
	for i := 1; i <= maxAttempts; i++ {
		_, err := eventsCollection.InsertOne(*ctx, event)
		if err == nil {
			return
		}

		log.Printf("Error inserting event to DB: %v, err: %s, attempt: %d", event, err, i)
		if i == maxAttempts {
			log.Printf("Failed to insert data to DB after %d attempts", maxAttempts)
			return
		}
		time.Sleep(time.Minute)
	}
}
