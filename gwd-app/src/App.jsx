import { useState, useRef } from 'react'
import { NotificationError } from "./Notification";
import './App.css'

// 添加消息气泡的样式
const messageStyles = {
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'flex-start',
  marginBottom: '10px',
};

const bubbleStyles = {
  backgroundColor: '#ededed',
  borderRadius: '15px',
  padding: '10px',
  maxWidth: '80%',
  wordWrap: 'break-word',
  textAlign: 'left',
};

let ws = null;

function App() {
  const [inputValue, setInputValue] = useState('');
  const chatHistoryRef = useRef(null);
  
  const [status, setStatus] = useState("idle"); // idle, busy
  const responseContentRef = useRef('');
  const messagesRef = useRef('');

  const [dialogMessages, setDialogMessages] = useState([]);

  const sendMessage = (message) => {
    // console.log("send message:", message);
    setDialogMessages(prevMessages => [...prevMessages, message]);
    setInputValue('');
    setTimeout(() => {
      if (chatHistoryRef.current) {
        chatHistoryRef.current.scrollTop = chatHistoryRef.current.scrollHeight;
      }
    }, 200);
  };

  const handleInputChange = (event) => {
    setInputValue(event.target.value);
  };

  const handleSendButtonClick = () => {
    if (inputValue.trim()) {
      sendMessage("问：" + inputValue);

      // 建立ws连接
      if (ws) {
        ws.send(
          JSON.stringify({
            gid: "",
            cmd: "create",
            data: {
              "prompt": inputValue,
              "user_uuid": "user_uuid",
              "notify_url": "",
              "from": "achat",
              "chat_uuid": "",
              "parent_chat_uuid": "",
              "pid": "GoWeaviateDeepseek"
            },
          })
        );
      } else {
        fetchWs(inputValue);
      }
    }
  };

  const handleKeyPress = (event) => {
    if (event.key === 'Enter') {
      handleSendButtonClick();
    }
  };

  const fetchWs = (inputValue) => {
    ws = new WebSocket("ws://localhost:5012/ds-ws");
    ws.onerror = function (e) {
      console.log("received error:", {
        event: e,
        isTrusted: e.isTrusted,
        timestamp: new Date().toISOString(),
        wsState: ws ? ws.readyState : "disconnected",
        wsUrl: ws ? ws.url : "unknown"
      });
      setStatus("idle");
      if (ws) {
        ws.close();
        ws = null;
      }
      NotificationError(i18n.t("net_error"));
    };

    ws.onopen = function () {
      setStatus("busy");
      ws.send(
        JSON.stringify({
          gid: "",
          cmd: "create",
          data: {
            "prompt": inputValue,
            "user_uuid": "user_uuid",
            "notify_url": "",
            "from": "achat",
            "chat_uuid": "",
            "parent_chat_uuid": "",
            "pid": "GoWeaviateDeepseek"
          },
        })
      );
    };

    ws.onmessage = function (event) {
      const res = JSON.parse(event.data);

      if (res.cmd === "create") {
        const data = res.data;
        console.log("data.c:", data.c);
        if (data.done) {
          sendMessage(responseContentRef.current);
          responseContentRef.current = '';
        } else {
          if (responseContentRef.current === '') {
            responseContentRef.current = "答：";
          } else {
            responseContentRef.current = responseContentRef.current + data.c;
          }
        }
      } else if (res.cmd === "error") {
        ws.close();
        NotificationError(i18n.t("net_error"));
      }
    };

    ws.onclose = function (e) {
      console.log("close", e);
      if (ws) {
        ws.close();
        ws = null;
      }
    };
  };

  return (
    <>
      <h1 className="text-center mt-5">Chatbot</h1>
      <h5>
        <a href="#" className="mt-3">详细教程点此访问蛋人网查看</a>
      </h5>
      <div className="container mt-3 position-relative">
        <div className="chat-window border p-3">
          <div className="chat-history" style={{ width: '400px', height: '500px', overflowY: 'scroll' }} ref={chatHistoryRef}>
            {dialogMessages.map((message, index) => (
              <div key={index} style={messageStyles}>
                <div style={bubbleStyles}>
                  {message}
                </div>
              </div>
            ))}
          </div>
          <div className="input-group mt-3">
            <input
              type="text"
              className="form-control"
              value={inputValue}
              onChange={handleInputChange}
              onKeyPress={handleKeyPress}
              placeholder="Type a message..."
            />
            <button className="btn btn-primary" onClick={handleSendButtonClick}>
              Send
            </button>
          </div>
        </div>
      </div>
    </>
  )
}

export default App