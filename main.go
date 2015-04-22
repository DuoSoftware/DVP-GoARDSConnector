package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	. "github.com/0x19/goesl"
	"github.com/jmcvetta/restclient"
	. "github.com/xuyu/goredis"
	"runtime"
	"strconv"
	"strings"
)

var (
	goeslMessage = "Hello from GoESL. Open source FreeSWITCH event socket wrapper written in Go!"
)

type Config struct {
	Server struct {
		Ip   string
		Port int
	}

	Dispatcher struct {
		Ip   string
		Port int
	}

	Ards struct {
		Ip   string
		Port int
	}

	Redis struct {
		Ip   string
		Port int
	}
}

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

var client *Redis
var cfg Config

func main() {

	err := gcfg.ReadFileInto(&cfg, "./config.ini")

	defer func() {
		if r := recover(); r != nil {
			Error("Recovered in: ", r)
		}
	}()

	// Boost it as much as it can go ...
	runtime.GOMAXPROCS(runtime.NumCPU())

	///////////////////////register with ards as a requester//////////////////////////////////////////

	callbakURL := fmt.Sprintf("http://%s:%d/route", cfg.Dispatcher.Ip, cfg.Dispatcher.Port)

	ardsADDServerURL := fmt.Sprintf("http://%s:%d/requestserver/add", cfg.Ards.Ip, cfg.Ards.Port)

	fmt.Print(callbakURL)

	registered := true

	f := ServerInfo{
		Company:     1,
		Tenant:      3,
		Class:       "CALLSERVER",
		Type:        "ARDS",
		Category:    "CALL",
		CallbackUrl: callbakURL,
		ServerID:    1,
	}

	r := restclient.RequestResponse{
		Url:    ardsADDServerURL,
		Method: "POST",
		Data:   &f,
		Result: &registered,
	}
	status, err := restclient.Do(&r)
	if err != nil {
		//panic(err)
	}
	if status == 200 {
		Debug("Registered %v", registered)
	}

	///////////////////////////////////////////////////////////////////////////////////////////////////////

	if s, err := NewOutboundServer(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		Error("Got error while starting FreeSWITCH outbound server: %s", err)
	} else {
		go handle(s)
		s.Start()
	}

}

func RemoveRequest(company, tenant int, sessionid string) {

	go func() {

		r := restclient.RequestResponse{
			Url:    fmt.Sprintf("http://%s:%d/request/remove/%d/%d/%s", cfg.Ards.Ip, cfg.Ards.Port, company, tenant, sessionid),
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
	}()

}

func RejectRequest(company, tenant int, sessionid, reason string) {

	go func() {

		Debug("Reject request recived --------------->")

		r := restclient.RequestResponse{
			Url:    fmt.Sprintf("http://%s:%d/request/reject/%d/%d/%s/%s", cfg.Ards.Ip, cfg.Ards.Port, company, tenant, sessionid, reason),
			Method: "DELETE",
		}

		Info(r.Url)

		status, err := restclient.Do(&r)
		if err != nil {
			//panic(err)
			fmt.Printf("MSG error -------------------------------------> %s", err)
		}
		if status == 200 {
			//fmt.Printf(registered)
		}
	}()

}

func AddRequest(company, tenant int, sessionid string, skills []string) {

	Debug("Add ARDS item ----->")

	var registered string

	//attrib := []string{"123456"}
	f := RequestData{
		Company:         company,
		Tenant:          tenant,
		Class:           "CALLSERVER",
		Type:            "ARDS",
		Category:        "CALL",
		SessionId:       sessionid,
		RequestServerId: "1",
		Priority:        "L",
		OtherInfo:       "",
		Attributes:      skills,
	}

	postData := Request1{

		Request1: f,
	}

	//Url:    fmt.Sprintf("http://%s:%d/startArds/web/Start", cfg.Ards.Ip, cfg.Ards.Port),
	r := restclient.RequestResponse{
		Url:    fmt.Sprintf("http://192.168.3.13:2221/startArds/web/Start"),
		Method: "POST",
		Data:   &postData,
		Result: &registered,
	}

	Info(r.Url)

	status, err := restclient.Do(&r)
	if err != nil {
		//panic(err)
		Error("%s", err)
	}
	if status == 200 {
		Debug(registered)
	}

}

// handle - Running under goroutine here to explain how to run tts outbound server
func handle(s *OutboundServer) {

	client, err := Dial(&DialConfig{Address: fmt.Sprintf("%s:%d", cfg.Redis.Ip, cfg.Redis.Port)})

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
				company := msg.GetHeader("Variable_company")
				tenant := msg.GetHeader("Variable_tenant")
				skill := msg.GetHeader("Variable_skill")
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
				Debug(company)
				Debug(tenant)
				Debug(skill)

				comapnyi, _ := strconv.Atoi(company)
				tenanti, _ := strconv.Atoi(tenant)

				if direction == "outbound" {
					Debug("OutBound Call recived ---->")

					//if channelStatus != "answered" {
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

							RejectRequest(1, 3, originateSession, "NoSession")

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

												exsists, exsisterr := client.Exists(key)
												if exsisterr == nil && exsists == true {

													client.HSet(key, "AgentStatus", "AgentConnected")

													cmd := fmt.Sprintf("uuid_bridge %s %s", originateSession, uniqueID)
													Debug(cmd)
													conn.BgApi(cmd)
													/////////////////////Remove///////////////////////

													RemoveRequest(comapnyi, tenanti, originateSession)

												} else {

													RejectRequest(comapnyi, tenanti, originateSession, "NoSession")

													conn.ExecuteHangup(uniqueID, "", false)

												}

											} else if event == "CHANNEL_HANGUP" {

												value1, getErr1 := client.HGet(key, "AgentStatus")
												agentstatus := string(value1[:])
												if getErr1 == nil {

													if agentstatus != "AgentConnected" {

														if agentstatus == "AgentKilling" {

														} else {

															//////////////////////////////Reject//////////////////////////////////////////////
															//http://localhost:2225/request/remove/company/tenant/sessionid

															RejectRequest(comapnyi, tenanti, originateSession, "AgentRejected")
														}

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

						//}
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
					skills := []string{skill}
					AddRequest(comapnyi, tenanti, uniqueID, skills)

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
					conn.Send("linger")
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

										Debug("Event -------------------->", event)

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

											value2, _ := client.HGet(key, "AgentUUID")
											sValue2 := string(value2[:])
											if getErr1 == nil {

												if agentstatus != "AgentConnected" {

													//////////////////////////////Remove//////////////////////////////////////////////

													if agentstatus == "AgentFound" {

														client.HSet(key, "AgentStatus", "AgentKilling")

														cmd := fmt.Sprintf("uuid_kill %s ", sValue2)
														Debug(cmd)
														conn.Api(cmd)

														RejectRequest(comapnyi, tenanti, uniqueID, "ClientRejected")

													} else {
														RemoveRequest(comapnyi, tenanti, uniqueID)

													}

												}

											}

											client.Del(key)
											client.Del(partykey)
											conn.Exit()

										} else if event == "CHANNEL_HANGUP_COMPLETED" {

											conn.Close()

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
