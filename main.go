package main

import (
	"fmt"
	. "github.com/0x19/goesl"
	"github.com/jmcvetta/restclient"
	. "github.com/xuyu/goredis"
	"runtime"
	"strings"
)

var (
	goeslMessage = "Hello from GoESL. Open source FreeSWITCH event socket wrapper written in Go!"
)

type ServerInfo struct {
	Company     int
	Tenant      int
	Class       string
	Type        string
	Category    string
	CallbackUrl string
	ServerID    int
}

type RequestData struct {
	Company         int
	Tenant          int
	Class           string
	Type            string
	Category        string
	SessionId       string
	Attributes      []string
	Priority        string
	RequestServerId string
	OtherInfo       string
}

type Request1 struct {
	Request1 RequestData
}

/*

public class InputData
    {
        public int Company { get; set; }
        public int Tenant { get; set; }
        public string Class { get; set; }
        public string Type { get; set; }
        public string Category { get; set; }
        public string SessionId { get; set; }
        public List<string> Attributes { get; set; }
        public string RequestServerId { get; set; }
        public string Priority { get; set; }
        public string OtherInfo { get; set; }
    }

*/

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

	registered := true

	f := ServerInfo{
		Company:     1,
		Tenant:      3,
		Class:       "CALLSERVER",
		Type:        "ARDS",
		Category:    "CALL",
		CallbackUrl: "http://192.168.0.79:8086/route",
		ServerID:    1,
	}

	r := restclient.RequestResponse{
		Url:    "http://192.168.0.25:2225/requestserver/add",
		Method: "POST",
		Data:   &f,
		Result: &registered,
	}
	status, err := restclient.Do(&r)
	if err != nil {
		//panic(err)
	}
	if status == 200 {
		println(registered)
	}

	///////////////////////////////////////////////////////////////////////////////////////////////////////

	if s, err := NewOutboundServer(":8084"); err != nil {
		Error("Got error while starting FreeSWITCH outbound server: %s", err)
	} else {
		go handle(s)
		s.Start()
	}

}

func RemoveRequest(company, tenant, sessionid string) {

	r := restclient.RequestResponse{
		Url:    fmt.Sprintf("http://192.168.0.25:2225/request/remove/%s/%s/%s", company, tenant, sessionid),
		Method: "DELETE",
	}
	status, err := restclient.Do(&r)
	if err != nil {
		//panic(err)
		fmt.Printf("MSG error -------------------------------------> %s", err)
	}
	if status == 200 {
		//fmt.Printf(registered)
	}

}

func RejectRequest(company, tenant, sessionid, reason string) {

	r := restclient.RequestResponse{
		Url:    fmt.Sprintf("http://192.168.0.25:2225/request/reject/%s/%s/%s/%s", company, tenant, sessionid, reason),
		Method: "DELETE",
	}
	status, err := restclient.Do(&r)
	if err != nil {
		//panic(err)
		fmt.Printf("MSG error -------------------------------------> %s", err)
	}
	if status == 200 {
		//fmt.Printf(registered)
	}

}

// handle - Running under goroutine here to explain how to run tts outbound server
func handle(s *OutboundServer) {

	client, err := Dial(&DialConfig{Address: "127.0.0.1:6379"})

	if err != nil {

		Error("Error occur in connecting redis", err)
		return

	}

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

				if direction == "outbound" {
					Debug("OutBound Call recived ---->")

					if channelStatus != "answered" {
						////////////////////////////////////////////////////////////
						if len(originateSession) > 0 {

							Debug("Original session found %s", originateSession)

							var isStored = true
							partykey := fmt.Sprintf("ARDS:Leg:%s", uniqueID)
							key := fmt.Sprintf("ARDS:Session:%s", originateSession)

							exsists, exsisterr := client.Exists(key)
							if exsisterr == nil && exsists == true {

								redisErr := client.SimpleSet(partykey, originateSession)
								Debug("Store Data : %s ", redisErr)
								isStored, redisErr = client.HSet(key, "AgentStatus", "AgentFound")
								Debug("Store Data : %s %s", isStored, redisErr)
								isStored, redisErr = client.HSet(key, "AgentUUID", uniqueID)
								Debug("Store Data : %s %s", isStored, redisErr)
								//msg, err = conn.Execute("wait_for_answer", "", true)
								//Debug("wait for answer ----> %s", msg)
								//msg, err = conn.ExecuteSet("CHANNEL_CONNECTION", "true", false)
								//Debug("Set variable ----> %s", msg)

							} else {

								RejectRequest("1", "3", originateSession, "ClientReject")

								cmd := fmt.Sprintf("uuid_kill %s ", uniqueID)
								Debug(cmd)
								conn.BgApi(cmd)

							}

							conn.Send("myevents json")
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

											contentType := msg.GetHeader("Content-Type")
											event := msg.GetHeader("Event-Name")
											Debug("Content types -------------------->", contentType)

											if contentType == "text/disconnect-notice" {

												//key := fmt.Sprintf("ARDS:Session:%s", uniqueID)

											} else {

												if event == "CHANNEL_ANSWER" {

													client.HSet(key, "AgentStatus", "AgentConnected")

													cmd := fmt.Sprintf("uuid_bridge %s %s", originateSession, uniqueID)
													Debug(cmd)
													conn.BgApi(cmd)
													/////////////////////Remove///////////////////////

													RemoveRequest("1", "3", originateSession)

												} else if event == "CHANNEL_HANGUP" {

													value1, getErr1 := client.HGet(key, "AgentStatus")
													agentstatus := string(value1[:])
													if getErr1 == nil {

														if agentstatus != "AgentConnected" {

															//////////////////////////////Reject//////////////////////////////////////////////
															//http://localhost:2225/request/remove/company/tenant/sessionid

															RejectRequest("1", "3", originateSession, "AgentRejected")

														}
													}

												}
											}
										}
									}

									Debug("Got message: %s", msg)
								}
								Debug("Leaving go routing after everithing completed OutBound %s %s", key, partykey)
								//client.Del(key)
								client.Del(partykey)
							}()

						}
						/////////////////////////////////////////////////////////////
					}

				} else {

					answer, err := conn.ExecuteAnswer("", false)

					if err != nil {
						Error("Got error while executing answer: %s", err)

					}

					Debug("Answer Message: %s", answer)
					Debug("Caller UUID: %s", uniqueID)

					//////////////////////////////////////////Add to queue//////////////////////////////////////
					var registered string

					attrib := []string{"123456"}
					f := RequestData{
						Company:         1,
						Tenant:          3,
						Class:           "CALLSERVER",
						Type:            "ARDS",
						Category:        "CALL",
						SessionId:       uniqueID,
						RequestServerId: "1",
						Priority:        "L",
						OtherInfo:       "",
						Attributes:      attrib,
					}

					postData := Request1{

						Request1: f,
					}

					r := restclient.RequestResponse{
						Url:    "http://192.168.0.25:2221/startArds/web/Start",
						Method: "POST",
						Data:   &postData,
						Result: &registered,
					}
					status, err := restclient.Do(&r)
					if err != nil {
						//panic(err)
						fmt.Printf("%s", err)
					}
					if status == 200 {
						fmt.Printf(registered)
					}

					///////////////////////////////////////////////////////////////////////////////////////////

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

									contentType := msg.GetHeader("Content-Type")
									event := msg.GetHeader("Event-Name")
									application := msg.GetHeader("variable_current_application")

									Debug("Content types -------------------->", contentType)
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

											}
										} else if event == "CHANNEL_HANGUP" {

											value1, getErr1 := client.HGet(key, "AgentStatus")
											agentstatus := string(value1[:])
											if getErr1 == nil {

												if agentstatus != "AgentConnected" {

													//////////////////////////////Remove//////////////////////////////////////////////

													RemoveRequest("1", "3", uniqueID)
												}
											}

										}
									}
								}
							}
							//Debug("Got message: %s", msg)
						}
						Debug("Leaving go routing after everithing completed Inbound %s %s", key, partykey)
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
