import { useState, useRef, useEffect } from 'react'

function App() {
  const [messages, setMessages] = useState([
    { 
      id: 1, 
      role: 'bot', 
      content: '你好！我是 OmniDesk 企业内部智能服务台。\n你可以问我关于 HR 政策、IT 故障、报修等问题，或者试着问我：“连不上WiFi怎么办？”' 
    }
  ])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [sessionId] = useState(() => 'sess_' + Math.random().toString(36).substring(2, 9))
  const messagesEndRef = useRef(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages, isLoading])

  const handleSend = async (e) => {
    e?.preventDefault()
    if (!input.trim() || isLoading) return

    const userMsg = { id: Date.now(), role: 'user', content: input.trim() }
    setMessages(prev => [...prev, userMsg])
    setInput('')
    setIsLoading(true)

    try {
      const response = await fetch('http://localhost:8081/api/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: sessionId,
          message: userMsg.content
        })
      })

      const data = await response.json()
      
      if (!response.ok) {
        throw new Error(data.error || 'Server Error')
      }

      const botMsg = { id: Date.now(), role: 'bot', content: data.reply }
      setMessages(prev => [...prev, botMsg])
    } catch (error) {
      const errorMsg = { 
        id: Date.now(), 
        role: 'bot', 
        content: '⚠️ 系统发生错误: ' + error.message 
      }
      setMessages(prev => [...prev, errorMsg])
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="chat-container">
      {/* 头部信息区 */}
      <div className="chat-header">
        <div className="avatar">OD</div>
        <div className="header-info">
          <h1>OmniDesk Support</h1>
          <p>
            <span className="status-dot"></span>
            Agent Online
          </p>
        </div>
      </div>

      {/* 聊天记录区 */}
      <div className="messages-area">
        {messages.map((msg) => (
          <div key={msg.id} className={`message-wrapper ${msg.role}`}>
            <div className="message-content">
              {msg.content}
            </div>
          </div>
        ))}
        
        {isLoading && (
          <div className="typing-indicator">
            <span className="dot"></span>
            <span className="dot"></span>
            <span className="dot"></span>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* 输入框区 */}
      <div className="input-area">
        <form onSubmit={handleSend} className="input-wrapper">
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="输入您遇到的问题 (如：年假怎么算？)"
            disabled={isLoading}
          />
          <button type="submit" className="send-btn" disabled={!input.trim() || isLoading}>
            <svg viewBox="0 0 24 24">
              <path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z" />
            </svg>
          </button>
        </form>
      </div>
    </div>
  )
}

export default App
