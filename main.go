package main

import (
	. "github.com/0x19/goesl"
	. "github.com/xuyu/goredis"
	"runtime"
	"strings"
)

var (
	goeslMessage = "Hello from GoESL. Open source FreeSWITCH event socket wrapper written in Go!"
)

var client *Redis

func main() {

	defer func() {
		if r := recover(); r != nil {
			Error("Recovered in: ", r)
		}
	}()

	// Boost it as much as it can go ...
	runtime.GOMAXPROCS(runtime.NumCPU())

	client, err := Dial(&DialConfig{Address: "127.0.0.1:6379"})
	if err != nil {

		Error("Error occur in connecting redis", err)
		return

	}

	///////////////////////register with ards as a requester//////////////////////////////////////////
	client.SimpleSet("ARDSCOnnector", "Started")

	if s, err := NewOutboundServer(":8084"); err != nil {
		Error("Got error while starting FreeSWITCH outbound server: %s", err)
	} else {
		go handle(s)
		s.Start()
	}

}

// handle - Running under goroutine here to explain how to run tts outbound server
func handle(s *OutboundServer) {

	for {

		select {

		case conn := <-s.Conns:
			Notice("New incomming connection: %v", conn)

			if err := conn.Connect(); err != nil {
				Error("Got error while accepting connection: %s", err)

				break
			}

			msg, err := conn.ReadMessage()

			if err == nil && msg != nil {

				//conn.Send("myevents")
				Debug("Connect message %s", msg.Headers)

				uniqueID := msg.GetCallUUID()
				from := msg.GetHeader("Caller-Caller-Id-Number")
				to := msg.GetHeader("Caller-Destination-Number")
				direction := msg.GetHeader("Call-Direction")
				channelStatus := msg.GetHeader("Answer-State")
				originateSession := msg.GetHeader("variable_Originate_session_uuid")
				fsUUID := msg.GetHeader("Core-Uuid")
				fsHost := msg.GetHeader("Freeswitch-Hostname")
				fsName := msg.GetHeader("Freeswitch-Switchname")
				fsIP := msg.GetHeader("Freeswitch-Ipv4")
				callerContext := msg.GetHeader("Caller-Context")

				//conn.Send(fmt.Sprintf("myevent json %s", uniqueID))

				if len(originateSession) == 0 {

					Debug("New Session created --->  %s", uniqueID)

				} else {

				}

				Debug(from)
				Debug(to)
				Debug(direction)
				Debug(channelStatus)
				Debug(fsUUID)
				Debug(fsHost)
				Debug(fsName)
				Debug(fsIP)
				Debug(originateSession)
				Debug(callerContext)

				if direction == "outbound" {

					if channelStatus != "answered" {

						conn.Execute("wait_for_answer", "", false)

						conn.ExecuteUUID(originateSession, "break", "", false)

						go func() {
							for {
								msg, err := conn.ReadMessage()

								if err != nil {

									// If it contains EOF, we really dont care...
									if !strings.Contains(err.Error(), "EOF") {
										Error("Error while reading Freeswitch message: %s", err)
									}
									break
								}

								Debug("Got message: %s", msg)
							}
							Debug("Leaving go routing after everithing completed")
						}()
					}

				} else {

					answer, err := conn.ExecuteAnswer("", false)

					if err != nil {
						Error("Got error while executing answer: %s", err)
						break
					}

					Debug("Answer Message: %s", answer)
					Debug("Caller UUID: %s", uniqueID)

					cUUID := uniqueID

					//conn.Send("myevents")

					eventsync := make(chan string)

					go func(sysncchannel chan string) {

						conn.Send("myevents json")
						for {
							msg, err := conn.ReadMessage()

							if err != nil {

								// If it contains EOF, we really dont care...
								if !strings.Contains(err.Error(), "EOF") {
									Error("Error while reading Freeswitch message: %s", err)
								}

								break
							} else {

								//Debug("Message recive with event name -> %s", msg.GetHeader("Event-Name"))
								sysncchannel <- msg.GetHeader("Event-Name")

							}

							//Debug("Got message: %s", msg)
						}
						Debug("Leaving go routing after everithing completed")
					}(eventsync)

					//////////////////////////////////////////Add to queue//////////////////////////////////////
					if sm, err := conn.Execute("playback", "local_stream://moh", true); err != nil {
						Error("Got error while executing speak: %s", err)
						break
					} else {

						Debug("Playback reply %s", sm)

						/*
							for {

								select {

								case msgx := <-eventsync:
									{
										Debug(msgx)
									}
								}

							}*/

						//}

						/////////////////////////////////////////////////check agent connectivity////////////////////

						/////////////////////////////////////////////////Bridge call/////////////////////////////////

						/////////////////////////////////////////////////////////////////////////////////////////////
					}

					/////////////////////////////////////////////////////////////////////////////////////////////////

					if hm, err := conn.ExecuteHangup(cUUID, "", false); err != nil {
						Error("Got error while executing hangup: %s", err)
						break
					} else {
						Debug("Hangup Message: %s", hm)
					}

				}

			} else {

				Error("Got Error %s", err)
				conn.Exit()
			}

		default:
		}
	}

}
