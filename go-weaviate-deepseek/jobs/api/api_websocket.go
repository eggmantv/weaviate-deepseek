package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-weaviate-deepseek/conf"
	"go-weaviate-deepseek/ext"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-weaviate-deepseek/ext/connpool"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/tidwall/gjson"
)

const (
	wsHeaderGID = "X_GID"

	// 最高连接数限制
	wsMaxConnection = 10000
)

var wsConnPool *connpool.Pool

var (
	wsWriteWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	wsPongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	wsPingPeriod = (wsPongWait * 9) / 10

	wsNewline = []byte{'\n'}
	wsSpace   = []byte{' '}
)

type wsData struct {
	GID  string            `json:"gid"` // 目前没用，只有再发送广播消息时才需要客户端发送，目前没有使用广播消息
	Cmd  string            `json:"cmd"`
	Data map[string]string `json:"data"`
}

func wsl() *logrus.Entry {
	return ext.LF("ws")
}

func init() {
	wsConnPool = connpool.NewPool()
	wsConnPool.OnAdd = func(gid string, c *connpool.Client) {
		wsl().Printf("gid: %s connected", gid)
	}
	wsConnPool.OnRemove = func(gid string) {
		wsl().Printf("gid: %s disconnected", gid)
	}
	wsConnPool.OnBroadcast(func(_ *connpool.Pool, data []byte) {
		var d wsData
		_ = json.Unmarshal(data, &d)
		if len(d.GID) == 0 {
			return
		}

		wsSend(d.GID, data)
	})
}

// apiWS
func apiWS(r *gin.Engine) {
	// 推送数据
	// body结构为 `wsData` struct
	r.POST("/ws/push", func(ctx *gin.Context) {
		wsConnPool.Broadcast([]byte(readBody(ctx)))
		ctx.JSON(http.StatusOK, ext.M{"status": "ok"})
	})

	// 获取连接数量
	r.POST("/ws/runtime", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, ext.M{
			"status": "ok",
			"data": ext.M{
				"max_connection":      wsMaxConnection,
				"current_groups":      len(wsConnPool.ClientsGroup),
				"current_connections": len(wsConnPool.Clients),
			},
		})
	})

	// websocket连接路由
	// ws://host:port/ws?ref=gidSecret
	r.GET("/ds-ws", func(ctx *gin.Context) {
		// reach max connection limit
		if len(wsConnPool.Clients) >= wsMaxConnection {
			ctx.AbortWithStatusJSON(402, ext.M{
				"status": "error",
				"error":  "ws reach max connection limit",
			})
			return
		}

		// validate gid 如有需要可增加验证校验
		// gid 是为了区分是否是同一个连接
		// gid, err := validateRef(ctx)
		// if ok := checkErr(err, ctx); !ok {
		// 	return
		// }

		gid := ext.GenUUID()
		wshandler(ctx.Writer, ctx.Request, gid)
	})

	// 监控连接池的大小
	if !conf.IsPrd() {
		go func() {
			for {
				wsl().Printf("conn group size: %d, conn size: %d", len(wsConnPool.ClientsGroup), len(wsConnPool.Clients))
				time.Sleep(2 * time.Second)
			}
		}()
	}
}

var wsGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wshandler(w http.ResponseWriter, r *http.Request, gid string) {
	conn, err := wsGrader.Upgrade(w, r, nil)
	if err != nil {
		wsl().Warnln("failed to set websocket upgrade:", err)
		return
	}

	sid := ext.GenUUID()
	c := connpool.NewClient(gid, sid, nil)
	chatCtx, cancel := context.WithCancel(context.Background())
	c.ChatCtx = chatCtx
	c.ChatCancelFn = cancel
	c.Conn = conn
	wsConnPool.Add(gid, c)
	wsl().Printf("ws new connection, client addr: %s, gid: %s", r.RemoteAddr, gid)
	go wsRead(c)
	go wsWrite(c)
}

func wsRead(client *connpool.Client) {
	conn := client.Conn
	defer func() {
		conn.Close()
		wsConnPool.Remove(client.GID, client.ConnID)
	}()
	// conn.SetReadLimit(int64(maxMessageSize))

	// conn.SetPongHandler(func(string) error {
	// 	conn.SetReadDeadline(time.Now().Add(wsPongWait))
	// 	return nil
	// })
	for {
		// conn.SetReadDeadline(time.Now().Add(15 * time.Minute))
		_, message, err := conn.ReadMessage()
		if err != nil {
			// if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			// 	wsl().Printf("error: %v", err)
			// }
			wsl().Println("ws client read failed, read exit, err:", err)
			return
		}
		message = bytes.TrimSpace(bytes.Replace(message, wsNewline, wsSpace, -1))
		wsOnMessage(string(message), client)
		// wsl().Printf("received: %s\n", string(message))
	}
}

// write pumps messages from the hub to the websocket connection.
//
// A goroutine running write is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func wsWrite(client *connpool.Client) {
	conn := client.Conn
	ticker := time.NewTicker(wsPingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
		wsConnPool.Remove(client.GID, client.ConnID)
	}()
	for {
		select {
		case <-client.ChatCtx.Done():
			wsl().Printf("ws close conn, chatCtx canceled, %s", client.ChatCtx.Err())
			return
		case <-client.CloseChan:
			wsl().Println("ws close conn, write exit")
			return
		case message, ok := <-client.SendChan:
			conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				wsl().Println("read from send failed, write exit")
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				wsl().Println("get next writer failed, write exit, err:", err)
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			// n := len(client.SendChan)
			// for i := 0; i < n; i++ {
			// 	w.Write(wsNewline)
			// 	w.Write(<-client.SendChan)
			// }

			if err := w.Close(); err != nil {
				wsl().Println("close ws writer failed, write exit, err:", err)
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				wsl().Println("ws ping failed, write exit, err:", err)
				return
			}
		}
	}
}

func wsOnMessage(msg string, client *connpool.Client) {
	var d wsData
	_ = json.Unmarshal([]byte(msg), &d)
	switch d.Cmd {
	case "test":
		wsSend(client.GID, ext.ToB(ext.M{
			"cmd":  "test-back",
			"data": d.Data,
		}))
	case "create":
		go ChatStreamWithCallback(client.ChatCtx, d.Data, func(msg ext.M) {
			wsSend(client.GID, ext.ToB(msg))
		}, func(url string, res ext.M) {
			// notify to 3rd party
			data := res["data"].(ext.M)
			if cast.ToString(data["is3rd"]) == "true" {
				webHook := cast.ToString(data["web_hook"])
				if len(webHook) > 0 {
					go notifyJobTo3rd(webHook, res)
				}
			}

			// notify to us
			rsp := notifyJob(url, res)
			if rsp != nil {
				userPoints := gjson.ParseBytes(rsp.Body()).Get("user_points").Float()
				if userPoints <= 0 {
					wsl().Println("user has no credits, disconnect ws connection manually!")
					if client.ChatCancelFn != nil {
						client.ChatCancelFn()
					}
				}
			}
		})
	case "stop":
		if client.ChatCancelFn != nil {
			client.ChatCancelFn()
		}
	}
}

func wsSend(gid string, msg []byte) {
	// wsl().Printf("ws send msg: %s, gid: %s\n", msg, gid)

	for _, sid := range wsConnPool.ClientsGroup[gid] {
		co := wsConnPool.Clients[sid]
		if co == nil {
			continue
		}
		co.SendChan <- msg
	}
}

func validateRef(ctx *gin.Context) (string, error) {
	gidSecret := ctx.GetHeader(wsHeaderGID)
	if len(gidSecret) == 0 {
		gidSecret = ctx.Query("ref")
	}
	if len(gidSecret) == 0 {
		return "", errors.New("gid is invalid#1")
	}
	gidSecret, err := url.PathUnescape(gidSecret)
	if err != nil {
		return "", err
	}
	sec, err := ext.Decrypt(gidSecret)
	if err != nil {
		return "", err
	}
	secs := strings.Split(string(sec), "-")
	if len(secs) != 2 {
		return "", errors.New("gid is invalid#2")
	}
	if secs[1] != string(ext.EncryptSecret)[0:8] {
		return "", errors.New("gid is invalid#3")
	}
	gid := secs[0]
	if len(gid) == 0 {
		return "", errors.New("gid is invalid#4")
	}
	return gid, nil
}
