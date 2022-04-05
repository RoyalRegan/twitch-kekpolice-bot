package main

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/textproto"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const botName = "__"
const botToken = "__"
const channel = "__"

var (
	userRe     = regexp.MustCompile("\\w+")
	msgRe      = regexp.MustCompile("#\\w+ :.+$")
	tempUserRe = regexp.MustCompile("@[\\w]+")
)

var userDict = map[string]Counter{}

var kekCounts uint64

var twitchEmote = map[int]string{0: "KEKW", 1: "WutFace", 2: "Kreygasm", 3: "SeemsGood", 4: "BibleThump", 5: "OpieOP"}

type Counter struct {
	messages     uint64
	kekwMessages uint64
}

type OrderedMap struct {
	keys   []string
	values []Counter
}

func (o *OrderedMap) Len() int {
	return len(o.keys)
}
func (o *OrderedMap) Less(i, j int) bool {
	return o.values[i].kekwMessages > o.values[j].kekwMessages
}
func (o *OrderedMap) Swap(i, j int) {
	o.keys[i], o.keys[j] = o.keys[j], o.keys[i]
	o.values[i], o.values[j] = o.values[j], o.values[i]
}

func startReader(r io.Reader, readChan chan string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	reader := textproto.NewReader(bufio.NewReader(r))
	for {
		line, _ := reader.ReadLine()
		messages := strings.Split(line, "\r\n")
		for _, msg := range messages {
			readChan <- msg
		}
	}
}

func startWriter(w io.Writer, writeChan chan string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	for {
		select {
		case msg := <-writeChan:
			_, err := w.Write([]byte(msg))
			if err != nil {
				println(err)
			}
		}
	}
}

func processMsg(readChan, writeChan chan string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	for {
		select {
		case rawMessage := <-readChan:
			if strings.HasPrefix(rawMessage, "PING") {
				writeChan <- "PONG :tmi.twitch.tv\r\n"
			} else if strings.HasPrefix(rawMessage, "PONG") {
				continue
			} else {
				user := extractUser(rawMessage)
				if user != "tmi" && user != botName {
					message := extractMsg(rawMessage)

					if strings.HasPrefix(message, "!kekwcount") {
						writeChan <- buildMsg(fmt.Sprintf("Чат покекал %v раз https://coub.com/view/2f7hq4 ", kekCounts))
						continue
					}

					if strings.HasPrefix(message, "!kekpolice") {
						writeChan <- buildMsg(fmt.Sprintf("Кекполиция на месте, если хочешь узнать сколько раз ты кекнул введи !kekw, если сколько раз чат - !kekwcount"))
						continue
					}

					if strings.HasPrefix(message, "!kekwtop") && user == "vladviq1" {
						go topKekers(userDict, writeChan)
						continue
					}

					if strings.HasPrefix(message, "!kekw") {
						var appealUser string

						if appealUser = extractAppeal(message); appealUser == "" {
							appealUser = user
						}

						if counter, ok := userDict[appealUser]; ok == true {
							writeChan <- buildMsg(fmt.Sprintf("@%s покекал %v раз в %v сообщениях https://coub.com/view/2f7hq4 ", appealUser, counter.kekwMessages, counter.messages))
						} else {
							writeChan <- buildMsg(fmt.Sprintf("@%s ничего не писал ", appealUser))
						}
						continue
					}

					kekwCounter := strings.Count(message, "KEKW")

					kekCounts += uint64(kekwCounter)

					if counter, ok := userDict[user]; ok == true {
						userDict[user] = Counter{counter.messages + 1, counter.kekwMessages + uint64(kekwCounter)}
					} else {
						userDict[user] = Counter{1, uint64(kekwCounter)}
					}
				}
			}
		}
	}
}

func topKekers(m map[string]Counter, writeChan chan string) {
	var keys []string
	var values []Counter

	for k, v := range m {
		keys = append(keys, k)
		values = append(values, v)
	}

	oMap := OrderedMap{keys: keys, values: values}
	sort.Sort(&oMap)

	max := 5
	if len(keys) < max {
		max = len(keys)
	}

	msg := "Топ кекеров:\t"
	for i := 0; i < max; i++ {
		msg += fmt.Sprintf("%v. @%s кеков %d за %d сообщений\t", i+1, keys[i], values[i].kekwMessages, values[i].messages)
	}

	writeChan <- buildMsg(msg + ". https://coub.com/view/2f7hq4 ")
}

func buildMsg(msg string) string {

	return fmt.Sprintf("PRIVMSG #%s :%s\r\n", channel, msg+twitchEmote[rand.Intn(6)])
}

func extractUser(msg string) string {
	return userRe.FindString(msg)
}

func extractMsg(msg string) string {
	message := msgRe.FindString(msg)
	return message[strings.Index(message, ":")+1:]
}

func extractAppeal(msg string) string {
	str := tempUserRe.FindString(msg)
	if str == "" {
		return str
	} else {
		return strings.ToLower(tempUserRe.FindString(msg)[1:])
	}
}

func main() {
	conn, _ := net.Dial("tcp", "irc.twitch.tv:6667")

	readChan := make(chan string)
	writeChan := make(chan string)

	conn.Write([]byte(fmt.Sprintf("PASS %s\r\n", botToken)))
	conn.Write([]byte(fmt.Sprintf("NICK %s\r\n", botName)))
	conn.Write([]byte(fmt.Sprintf("JOIN #%s\r\n", channel)))

	var wg sync.WaitGroup

	wg.Add(1)
	go startReader(conn, readChan, &wg)

	wg.Add(1)
	go startWriter(conn, writeChan, &wg)

	wg.Add(1)
	go processMsg(readChan, writeChan, &wg)

	wg.Wait()
}
