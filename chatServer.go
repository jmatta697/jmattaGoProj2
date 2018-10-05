
// Chat is a server that lets clients chat with each other.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

//!+main
func main() {
	// ask user for idle time disconnect in seconds/pass to handleConnection
	idleDisconnectTime := getConnectionTimeLimit()
	fmt.Println(idleDisconnectTime)

	// listens for incoming connections
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	// -------------
	go broadcaster()
	// -------------
	// infinite loop
	for {
		// blocks until an incoming connection request is made
		// when a connection is established, a connection object is created
		conn, err := listener.Accept()	// **blocking function**
		if err != nil {
			log.Print(err)
			continue
		}
		// a handleConn() goroutine is created for each established connection
		go handleConnection(conn, idleDisconnectTime)
	}
}

//!-main
// --------------------------------------------------------------------------------
//!+broadcaster
// an outgoing message channel
// can only RECEIVE strings
// the only information carried by 'client' is the outgoing message
type client struct {
	msgChannel chan<- string
	userName string
}


// global vars (only variables shared by all concurrent goroutines) *also network connections*
var (
	entering = make(chan client)
	leaving  = make(chan client)
	globalMessages = make(chan string) // all incoming client messages

)

func broadcaster() {
	// all connected clients
	// this map is confined to the broadcaster goroutine
	listOfActiveClients := make(map[client]bool)

	// clientTimeLimit :=
	// infinite loop
	for {
		// will wait until one of the case are executed
		select {
		// if there is anything in the 'message channel' it is broadcast to all active clients
		case msg := <-globalMessages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for ActiveClient := range listOfActiveClients {
				ActiveClient.msgChannel <- msg
			}

		// ActiveClient variable receives from 'entering channel'
		case ActiveClient := <-entering:
			// adds to active client list
			listOfActiveClients[ActiveClient] = true
			activeClientListString := makeClientListString(listOfActiveClients)
			ActiveClient.msgChannel <- "Also here: " + activeClientListString

		case ActiveClient := <-leaving:
			delete(listOfActiveClients, ActiveClient)
			close(ActiveClient.msgChannel)
		}
	}
}

//!-broadcaster

//!+handleConn
func handleConnection(conn net.Conn, disconnectTime int) {
	// outgoing client messages for 'this' client (all messages, not just chat text)
	outMessageChannel := make(chan string)
	timer := time.NewTimer(time.Duration(disconnectTime) * time.Second)
	// start a clientWriter goroutine for each connected client
	// this goroutine takes all outgoing messages from the client and writes them
	// to the client's network connection
	go clientWriter(conn, outMessageChannel)

	who := getUserName(conn)
	outMessageChannel <- "You are " + who
	// sends entering announcement to global messages channel
	globalMessages <- who + " has arrived"
	// sends message to 'entering channel'
	newClient := client{outMessageChannel, who}
	entering <- newClient

	// monitors idle timer and disconnects if timer done
	go monitorTimerDisconnect(conn, timer, newClient)

	input := bufio.NewScanner(conn)
	for input.Scan() {	// **blocking function**
		globalMessages <- who + ": " + input.Text()
		timer.Reset(time.Duration(disconnectTime) * time.Second)
	}
	// NOTE: ignoring potential errors from input.Err()

	// this code will never be reached unless the client is forcibly interrupted
	if !connectionIsClosed(conn) {
		leaving <- newClient
		globalMessages <- who + " has left"
		conn.Close()
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		// write to the clients network connection
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

//!-handleConn

// gets user name from stdin
func getUserName(conn net.Conn) string {
	//reading a string
	reader := bufio.NewReader(conn)
	fmt.Fprint(conn,"Enter a user name: ")
	// 0x0a is the byte value for <\n>
	userName, _ := reader.ReadBytes(0x0a)
	// strip newline char from end of username
	cleanUserName := userName[:(len(userName)-1)]
	// cast bytes as string
	return string(cleanUserName)
}

func makeClientListString(clientMap map[client]bool) string {
	clientListString := ""
	for activeClient := range clientMap {
		clientListString += activeClient.userName + " "
	}
	return clientListString
}

func getConnectionTimeLimit() int {
	fmt.Println("Enter connection time limit (seconds): ")
	var timeLimit int
	_, err := fmt.Scanf("%d", &timeLimit)
		if err != nil {
			log.Print(err)
		}
	return timeLimit
}

func monitorTimerDisconnect(conn net.Conn, timer *time.Timer, client client) {
	<-timer.C
	leaving <- client
	globalMessages <- client.userName + " has left"
	conn.Close()
}

func connectionIsClosed(conn net.Conn) bool {
	var buffer []byte
	_, err := conn.Read(buffer)
	if err != nil {
		return true
	}
	return false
}

