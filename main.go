// Package main is the entry point of the chat application.
//
// The chat application allows peers to discover each other and exchange messages
// in a peer-to-peer manner.
//
// Usage:
//   - inbox: show all incoming messages
//   - send $peer_name: send a private message to a peer with $peer_name nickname
//   - exit: terminate the application.
//
// Example:
//
//	$ chat
//	Enter your name: YOUR_NAME
//	> Private listener started
//	Peer discovered: IP=192.168.1.11, Name=ANOTHER_NAME
//	> send ANOTHER_NAME
//	Enter message: Hello, peers!
//	> exit
//	Exiting...
//
// Networking:
//
//	The chat application uses port 8888 for UDP communication (broadcast) and 1234 for TCP (private messages)
//
// Package main provides the main function and related functions for the chat application.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Program settings - port for private messages and for broadcast
const (
	privatePort   = ":1234"
	broadcastPort = ":8888"
)

// Peer stores a dicovered peer information
type Peer struct {
	IP   string
	Name string
}

// General information about current session
var (
	// name represents user nickname and is initialized on startup
	name string
	// myIP - deprecated
	myIP string
	// peers is a map of discovered peers. Todo - remove peer if it is inactive for more than N minutes
	peers = make(map[string]string)
	// messages stores all inbox messages
	messages []string
	// mutex - synchronization primitive
	mutex sync.Mutex
	// lastActiveTimes - the last time when a user was online
	lastActiveTimes = make(map[string]time.Time) // Initialize lastActiveTimes map
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter your name: ")
	name, _ = reader.ReadString('\n')
	name = strings.TrimSpace(name)

	// Start private listener, broadcast listener, and broadcaster in separate goroutines
	go listenForPrivateMessages()
	go listenForBroadcast()
	go broadcaster()

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "peers":
			showPeers()
		case "inbox":
			showInbox()
		case "exit":
			fmt.Println("Exiting...")
			return
		default:
			if strings.HasPrefix(input, "send ") {
				peerName := strings.TrimSpace(strings.TrimPrefix(input, "send "))
				sendMessage(peerName)
			} else {
				fmt.Println("Invalid command.")
			}
		}
	}
}

// listenForPrivateMessages waits for all private messages
func listenForPrivateMessages() {
	l, err := net.Listen("tcp", privatePort)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	fmt.Println("Private listener started.")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		// Extract the sender's IP from the connection object
		senderIP, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			log.Fatal(err)
		}

		go handlePrivateMessage(conn, senderIP)
	}
}

func handlePrivateMessage(conn net.Conn, senderIP string) {
	defer conn.Close()

	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	message := string(buffer[:n])
	mutex.Lock()
	messages = append(messages, fmt.Sprintf("%s\nFrom: IP=%s, Name=%s", message, senderIP, peers[senderIP]))
	mutex.Unlock()
}

// listenForPeers listens for UDP broadcast messages to discover peers
func listenForBroadcast() {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0"+broadcastPort)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}
		message := string(buffer[:n])
		ip, peerName := parseBroadcastMessage(message)
		if ip != "" && peerName != "" {
			mutex.Lock()
			// Skip printing if the message is about our own IP
			if ip != myIP {
				// Skip printing if the peer has already been discovered before
				if _, exists := peers[ip]; !exists {
					fmt.Printf("Peer discovered: IP=%s, Name=%s\n", ip, peerName)
				}
			}
			// Update the peer's name in the peers map
			peers[ip] = peerName
			// Update the last active time of the peer
			lastActiveTimes[ip] = time.Now()
			mutex.Unlock()
		}
	}
}

func parseBroadcastMessage(message string) (string, string) {
	re := regexp.MustCompile(`IP: (\d+\.\d+\.\d+\.\d+), Name: (.+)`)
	match := re.FindStringSubmatch(message)
	if len(match) >= 3 {
		ip := match[1]
		peerName := match[2]
		return ip, peerName
	}
	return "", ""
}

// broadcaster notifies other peers every 10 seconds
func broadcaster() {
	conn, err := net.Dial("udp", "255.255.255.255"+broadcastPort)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ip, err := getOutboundIP()
	if err != nil {
		log.Fatal(err)
	}
	myIP = ip.String()

	for {
		broadcastMessage(fmt.Sprintf("IP: %s, Name: %s", myIP, name), conn)
		time.Sleep(10 * time.Second)
	}
}

// broadcastMessage is used to broadcast messages
func broadcastMessage(message string, conn net.Conn) {
	_, err := conn.Write([]byte(message))
	if err != nil {
		log.Fatal(err)
	}
}

func getOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// showPeers prints all known peers
func showPeers() {
	mutex.Lock()
	defer mutex.Unlock()

	fmt.Println("Peers:")
	for ip, name := range peers {
		fmt.Printf("Name: %s, IP: %s\n", name, ip)
	}
}

// sendMessage sends a message from command input to the peerName
func sendMessage(peerName string) {
	mutex.Lock()
	defer mutex.Unlock()

	targetIP := ""
	for ip, name := range peers {
		if name == peerName {
			targetIP = ip
			break
		}
	}

	if targetIP == "" {
		fmt.Println("Peer not found.")
		return
	}

	lastActiveTime, found := lastActiveTimes[targetIP]
	if found && time.Since(lastActiveTime) > time.Minute {
		fmt.Println("Warning: The peer has not been active for more than a minute.")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter message: ")
	message, _ := reader.ReadString('\n')
	message = strings.TrimSpace(message)

	conn, err := net.Dial("tcp", targetIP+privatePort)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		log.Fatal(err)
	}
}

// showInbox prints all inbox messages
func showInbox() {
	mutex.Lock()
	defer mutex.Unlock()

	fmt.Println("Inbox:")
	for _, message := range messages {
		fmt.Println(message)
		fmt.Println("|")
	}
}
