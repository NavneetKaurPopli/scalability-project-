package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
)

var wg sync.WaitGroup

// Command struct is a representation of an isolated command executed by a user
type Command struct {
	Command  string `json:"Command"`
	Username string `json:"Username"`
	Amount   string `json:"Amount"`
	Stock    string `json:"Stock"`
	Filename string `json:"Filename"`
}

type Response struct {
	Data  []byte `json:"data"`
	Error string `json:"error"`
}

type Transaction struct {
	ID              int64   `bson:"id"`
	Timestamp       int64   `bson:"timestamp"`
	TransactionType string  `bson:"transactionType"`
	Amount          float64 `bson:"amount"`
	Stock           string  `bson:"stock"`
}

type UserAccount struct {
	Username     string             `bson:"username"`
	Balance      float64            `bson:"balance"`
	Created      int64              `bson:"created"`
	Updated      int64              `bson:"updated"`
	BuyAmounts   map[string]float64 `bson:"buy"`
	SellAmounts  map[string]float64 `bson:"sell"`
	BuyTriggers  map[string]float64 `bson:"buyTriggers"`
	SellTriggers map[string]float64 `bson:"sellTriggers"`
	Stocks       map[string]float64 `bson:"stocks"`
	Transactions []*Transaction     `bson:"transactions"`
	RecentBuy    *CommandHistory    `bson:"recentBuy"`
	RecentSell   *CommandHistory    `bson:"recentSell"`
}
type Trigger struct {
	Stock string  `bson:"stock"`
	Price float64 `bson:"price"`
}

type CommandHistory struct {
	Timestamp int64   `bson:"timestamp"`
	Amount    float64 `bson:"amount"`
	Stock     string  `bson:"stock"`
}

// FromStringToCommandStruct takes a line from the user command file as an input and returns a defined golang structure
func FromStringToCommandStruct(line string) (*Command, error) {
	/*
		The line variable value should have a similar format to this: '[1] ADD,oY01WVirLr,63511.53'
	*/
	line = strings.Split(line, " ")[1]
	commandVars := strings.Split(line, ",")
	cmd := commandVars[0]

	if cmd == "ADD" {
		return &Command{Command: cmd, Username: commandVars[1], Amount: commandVars[2]}, nil
	}

	if cmd == "COMMIT_BUY" || cmd == "CANCEL_BUY" || cmd == "COMMIT_SELL" || cmd == "CANCEL_SELL" || cmd == "DISPLAY_SUMMARY" {
		return &Command{Command: cmd, Username: commandVars[1]}, nil
	}

	if cmd == "BUY" || cmd == "SELL" || cmd == "SET_BUY_AMOUNT" || cmd == "SET_BUY_TRIGGER" || cmd == "SET_SELL_AMOUNT" || cmd == "SET_SELL_TRIGGER" {
		return &Command{Command: cmd, Username: commandVars[1], Stock: commandVars[2], Amount: commandVars[3]}, nil
	}

	if cmd == "QUOTE" || cmd == "CANCEL_SET_BUY" || cmd == "CANCEL_SET_SELL" {
		return &Command{Command: cmd, Username: commandVars[1], Stock: commandVars[2]}, nil
	}

	if cmd == "DUMPLOG" {
		if len(commandVars) == 3 {
			// case: DUMPLOG,userid,filename
			return &Command{Command: cmd, Username: commandVars[1], Filename: commandVars[2]}, nil
		} else {
			// case: DUMPLOG,filename
			return &Command{Command: cmd, Filename: commandVars[1]}, nil
		}
	}

	if cmd == "DISPLAY_SUMMARY" {
		return &Command{Command: cmd, Username: commandVars[1]}, nil
	}

	return nil, fmt.Errorf("unable to conver given line: %s into golang struct", line)
}

func checkError(e error, additionalMessage string) {
	if e != nil {
		log.Printf(additionalMessage+": %s\n", e)
		panic(e)
	}
}

func HandleCommand(command *Command, conn net.Conn) error {
	var buffer bytes.Buffer
	err := json.NewEncoder(&buffer).Encode(command)
	if err != nil {
		log.Printf("Error while encoding command: %+v", command)
		return err
	}

	request := append([]byte(strconv.Itoa(buffer.Len())), buffer.Bytes()...)
	_, err = conn.Write(request)
	if err != nil {
		log.Printf("Error while writing command: %+v", command)
		return err
	}

	return nil
}

func HandleResponse(cmd *Command, res *http.Response) error {
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error while reading response body: %s\n", err)
		return err
	}

	responseStruct := &Response{}
	err = json.Unmarshal(bodyBytes, responseStruct)
	if err != nil {
		log.Printf("Error while unmarshalling response body: %s\n", err)
		return err
	}

	if responseStruct.Error != "" {
		log.Printf("Got an error in the response for command: %+v, error: %s\n", cmd, responseStruct.Error)
		return nil
	}

	if cmd.Command == "DUMPLOG" {
		file, err := os.Create("logfile.xml")
		if err != nil {
			log.Printf("error while creating file: %s\n", err)
			return err
		}

		_, err = file.Write(responseStruct.Data)
		if err != nil {
			log.Printf("Error while writing response body to file: %s\n", err)
			return err
		}

		err = file.Close()
		if err != nil {
			log.Printf("Error while closing file: %s\n", err)
		}

		fmt.Printf("Contents successfully written to %s\n", cmd.Filename)
		return nil
	}

	if cmd.Command == "DISPLAY_SUMMARY" {
		userAccount := &UserAccount{}
		err = bson.Unmarshal(responseStruct.Data, userAccount)
		if err != nil {
			log.Printf("Error while unmarshalling response body for cmd: %s, rawBytes: %s, error: %s\n", cmd.Command, responseStruct.Data, err)
		}

		fmt.Printf("-----User Account Summary-----\n")
		fmt.Printf("Username: %s\n", userAccount.Username)
		fmt.Printf("balance: %f\n", userAccount.Balance)
		for stock, amount := range userAccount.Stocks {
			fmt.Printf("stock %s: %f\n", stock, amount)
		}
		for _, t := range userAccount.Transactions {
			fmt.Printf("transaction: %3d, %9d, %s, %s, %f\n", t.ID, t.Timestamp, t.TransactionType, t.Stock, t.Amount)
		}
		for _, t := range userAccount.BuyTriggers {
			fmt.Printf("buy trigger: %v\n", t)
		}
		for _, t := range userAccount.SellTriggers {
			fmt.Printf("sell trigger: %v\n", t)
		}
		fmt.Printf("-----End------\n\n")

		return nil
	}

	// For other commands
	return nil
}

func ReadResponse(conn net.Conn) {
	response := make([]byte, 1024)

	for {
		numberOfBytes, err := conn.Read(response)
		if err != nil {
			if err == io.EOF {
				_ = conn.Close()
				log.Printf("connection closed")
				defer wg.Done()
				return
			}

			log.Printf("error while reading: %+v\n", err)
		}

		log.Printf("tried to read..\n")

		if numberOfBytes == 0 {
			log.Printf("number of bytes read is 0")
			continue
		}

		log.Println(string(response))
	}
}

func MakeSocketConnection() net.Conn {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Printf("Error while dialing: %s\n", err)
		panic(err)
	}

	return conn
}

func main() {

	conn := MakeSocketConnection()

	if len(os.Args) != 2 {
		fmt.Println("Please follow the following format: go run cmd.go <path_to_workload_file.txt>")
		panic("Unexpected number of arguments")
	}

	commandsFilePath := os.Args[1]
	data, err := os.ReadFile(filepath.Clean(commandsFilePath))
	checkError(err, "Error while reading file")

	lines := strings.Split(string(data), "\n")

	go ReadResponse(conn)
	wg.Add(1)

	for _, line := range lines {
		if line == "" {
			continue
		}

		requestData, err := FromStringToCommandStruct(line)
		checkError(err, "Couldn't convert line from file to command struct")

		if err != nil {
			log.Fatal(err)
		}

		err = HandleCommand(requestData, conn)
		if err != nil {
			log.Printf("Error while handling command %+v: %s\n", requestData, err)
		}
	}

	wg.Wait()
}
