package main

import (
	"fmt"
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

	///////////////////////register with ards as a requester//////////////////////////////////////////

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
				originateSession := msg.GetHeader("Variable_originate_session_uuid")
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

				client, err := Dial(&DialConfig{Address: "127.0.0.1:6379"})

				if err != nil {

					Error("Error occur in connecting redis", err)
					return

				}

				if direction == "outbound" {
					Debug("OutBound Call recived ---->")

					if channelStatus != "answered" {
						////////////////////////////////////////////////////////////
						if len(originateSession) > 0 {

							Debug("Original session found %s", originateSession)

							var isStored = true
							partykey := fmt.Sprintf("ARDS:Leg:%s", uniqueID)
							redisErr := client.SimpleSet(partykey, originateSession)
							Debug("Store Data : %s ", redisErr)
							key := fmt.Sprintf("ARDS:Session:%s", originateSession)
							isStored, redisErr = client.HSet(key, "AgentStatus", "AgentFound")
							Debug("Store Data : %s %s", isStored, redisErr)
							isStored, redisErr = client.HSet(key, "AgentUUID", uniqueID)
							Debug("Store Data : %s %s", isStored, redisErr)

							conn.Execute("wait_for_answer", "", true)

							isStored, redisErr = client.HSet(key, "AgentStatus", "AgentConnected")

							conn.ExecuteSet("CHANNEL_CONNECTION", "true", false)

							/*

								breakCommand := fmt.Sprintf("uuid_break %s all", originateSession)

								conn.BgApi(breakCommand)
							*/

							cmd := fmt.Sprintf("uuid_bridge %s %s", originateSession, uniqueID)
							Debug(cmd)
							conn.BgApi(cmd)

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
								Debug("Leaving go routing after everithing completed Inbound")
								client.Del(key)
								client.Del(partykey)
							}()

							//}

						}
						/////////////////////////////////////////////////////////////
					}

				} else {

					answer, err := conn.ExecuteAnswer("", false)

					if err != nil {
						Error("Got error while executing answer: %s", err)
						break
					}

					Debug("Answer Message: %s", answer)
					Debug("Caller UUID: %s", uniqueID)

					//cUUID := uniqueID

					//conn.Send("myevents")

					//////////////////////////////////////////Add to queue//////////////////////////////////////

					key := fmt.Sprintf("ARDS:Session:%s", uniqueID)

					partykey := fmt.Sprintf("ARDS:Leg:%s", uniqueID)
					var isStored = true
					Debug("key ---> %s ", partykey)
					redisErr := client.SimpleSet(partykey, uniqueID)
					Debug("Store Data : %s ", redisErr)

					Debug("key ---> %s ", key)
					isStored, redisErr = client.HSet(key, "CallStatus", "CallOnQueue")
					Debug("Store Data : %s ", redisErr)
					isStored, redisErr = client.HSet(key, "AgentStatus", "NotFound")

					Debug("Store Data : %s %s ", redisErr, isStored)

					conn.Send("myevents json")
					if sm, err := conn.Execute("playback", "local_stream://moh", false); err != nil {
						Error("Got error while executing speak: %s", err)
						break
					} else {

						Debug("Playback reply %s", sm)

					}

					Debug("Leaving go routing after everithing completed Inbound")

					/////////////////////////////////////////////////////////////////////////////////////////////////

					go func() {

						for {
							msg, err := conn.ReadMessage()

							if err != nil {

								// If it contains EOF, we really dont care...
								if !strings.Contains(err.Error(), "EOF") {
									Error("Error while reading Freeswitch message: %s", err)
								}

								break
							} else {
								if msg != nil {

									uuid := msg.GetHeader("Unique-ID")
									Debug(uuid)

									//key := fmt.Sprintf("ARDS:Session:%s", uniqueID)
									//partykey := fmt.Sprintf("ARDS:Leg:%s", uniqueID)

									contentType := msg.GetHeader("Content-Type")
									event := msg.GetHeader("Event-Name")
									application := msg.GetHeader("variable_current_application")
									//response := msg.GetHeader("variable_current_application_response")
									if contentType == "text/disconnect-notice" {

										//key := fmt.Sprintf("ARDS:Session:%s", uniqueID)

									} else {

										if event == "CHANNEL_EXECUTE_COMPLETE" && application == "playback" {

											value1, getErr1 := client.HGet(key, "AgentStatus")
											sValue1 := string(value1[:])

											value2, getErr2 := client.HGet(key, "AgentUUID")
											sValue2 := string(value2[:])

											Debug("Client side connection values %s %s %s %s", getErr1, getErr2, sValue1, sValue2)

											if getErr1 == nil && getErr2 == nil && sValue1 == "AgentConnected" && len(sValue2) > 0 {

												/*
													cmd := fmt.Sprintf("uuid_bridge %s %s", uuid, sValue2)

													Debug(cmd)
													conn.BgApi(cmd)

												*/

											}
										}
									}
								}
							}
							Debug("Got message: %s", msg)
						}
						Debug("Leaving go routing after everithing completed Inbound")
						client.Del(key)
						client.Del(partykey)
					}()

				}

			} else {

				Error("Got Error %s", err)
				conn.Exit()
			}

		default:
		}
	}

}
