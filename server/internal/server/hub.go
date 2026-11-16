package server

import (
	"log"
	"net/http"
	"server/pkg/packets"
)

type ClientInterfacer interface {
	Id() uint64

	//Sets up the client's ID and anything else that needs to be initialised
	Initialise(id uint64)

	ProcessMessage(senderId uint64, msg packets.Msg)

	//Pump data from the connected socket directly to the client
	ReadPump()

	//Pump data from the client directly to the connected socket
	WritePump()

	//Puts data from this client into the write pump
	SocketSend(message packets.Msg)

	//Put data from another client into the write pump
	SocketSendAs(message packets.Msg, senderId uint64)

	//Forward message to another clients for processing
	PassToPeer(message packets.Msg, peerId uint64)

	//Forward message to all other cleints for processing
	Broadcast(message packets.Msg)

	// Close the client's connection fand cleanup
	Close(reason string)
}


//The hub is a central point of communication between all connected clients
type Hub struct {
	//these are all our cleints
	Clients map[uint64]ClientInterfacer

	//boardcast channel, holds pointers to packets that will be processed by all connected clients except the sender
	BroadcastChan chan *packets.Packet

	//Clients in this channel will be registered on the hub
	RegisterChan chan ClientInterfacer

	//Clients in this channel will be unregistered from the hub
	UnregisterChan chan ClientInterfacer
}

func NewHub() *Hub {
	return &Hub {
		Clients: make(map[uint64]ClientInterfacer),
		BroadcastChan: make(chan *packets.Packet),
		RegisterChan: make(chan ClientInterfacer),
		UnregisterChan: make(chan ClientInterfacer),
	}
}

func (h *Hub) Run() {
	log.Println("Awaiting client registrations")
	for {
		select {
			case client := <- h.RegisterChan:
				client.Initialise(uint64(len(h.Clients)))
			case client := <- h.UnregisterChan:
				h.Clients[client.Id()] = nil
			case packet := <- h.BroadcastChan:
				for id, client := range h.Clients {
					if id != packet.SenderId {
						client.ProcessMessage(packet.SenderId, packet.Msg)
					}
				}
		}
	}
}

//this function takes in a callback function and http items and returns a client interface and error if any
func (h *Hub) Serve(getNewClient func(*Hub, http.ResponseWriter, *http.Request) (ClientInterfacer, error), writer http.ResponseWriter, request *http.Request ){
	log.Println("New client connected from: ", request.RemoteAddr)

	client, err := getNewClient(h, writer, request)

	if err != nil {
		log.Printf("Error obtaining client for new connection: %v", err)
		return
	}

	h.RegisterChan <- client //This pushes data into a channel

	//starts the functions on their own goroutine
	go client.WritePump()
	go client.ReadPump()

}
