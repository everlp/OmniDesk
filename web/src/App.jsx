import { useState, useRef, useEffect } from 'react'
import ReactMarkdown from 'react-markdown'
import './index.css'

function App() {
  const [messages, setMessages] = useState([
    { 
      id: 1, 
      message_id: 0,
      role: 'bot', 
      content: '你好！我是 OmniDesk 企业内部智能服务台。\n你可以问我关于 HR 政策、IT 故障、报修等问题，或者试着问我：“连不上WiFi怎么办？”' 
    }
  ])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [sessionId] = useState(() => 'sess_' + Math.random().toString(36).substring(2, 9))
  const messagesEndRef = useRef(null)

  // Track feedback state for each bot message
  const [feedbacks, setFeedbacks] = useState({})
  const [feedbackText, setFeedbackText] = useState('')
  const [activeFeedbackMsgId, setActiveFeedbackMsgId] = useState(null)
  const [isResolved, setIsResolved] = useState(false)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages, isLoading, activeFeedbackMsgId])

  const handleResolve = async () => {
    try {
      await fetch('http://localhost:8082/api/resolve', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId })
      })
      setIsResolved(true)
      setMessages(prev => [...prev, {
        id: Date.now(),
        role: 'bot',
        content: '🎉 本次问题已标记为解决，对话已完整归档。您可以继续咨询其他问题。'
      }])
    } catch (err) {
      console.error("Resolve failed", err)
    }
  }

  const handleSend = async (e) => {
    e?.preventDefault()
    if (!input.trim() || isLoading) return

    // 重新开启状态
    if (isResolved) setIsResolved(false)

    const userMsg = { id: Date.now(), role: 'user', content: input.trim() }
    setMessages(prev => [...prev, userMsg])
    setInput('')
    setIsLoading(true)

    try {
      const response = await fetch('http://localhost:8082/api/chat', {
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

      const botMsg = { 
        id: Date.now(), 
        message_id: data.message_id, 
        role: 'bot', 
        content: data.reply 
      }
      setMessages(prev => [...prev, botMsg])
    } catch (error) {
      const errorMsg = { 
        id: Date.now(), 
        message_id: 0,
        role: 'bot', 
        content: '⚠️ 系统拦截或发生错误: ' + error.message 
      }
      setMessages(prev => [...prev, errorMsg])
    } finally {
      setIsLoading(false)
    }
  }

  const handleFeedback = async (msgId, dbMessageId, rating) => {
    setFeedbacks(prev => ({ ...prev, [msgId]: { rating, submitted: false } }))
    
    // 如果是点赞，直接提交
    if (rating === 'up') {
      submitFeedback(dbMessageId, 'up', '')
      setFeedbacks(prev => ({ ...prev, [msgId]: { rating: 'up', submitted: true } }))
    } else {
      // 展开反馈文本框
      setActiveFeedbackMsgId(msgId)
    }
  }

  const submitFeedbackText = async (msgId, dbMessageId) => {
    await submitFeedback(dbMessageId, 'down', feedbackText)
    setFeedbacks(prev => ({ ...prev, [msgId]: { rating: 'down', submitted: true } }))
    setActiveFeedbackMsgId(null)
    setFeedbackText('')
  }

  const submitFeedback = async (dbMessageId, rating, comments) => {
    if (!dbMessageId) return;
    try {
      await fetch('http://localhost:8082/api/feedback', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: sessionId,
          message_id: dbMessageId,
          rating: rating,
          comments: comments
        })
      })
    } catch (err) {
      console.error("Failed to submit feedback", err)
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
            Agent Online (Session: {sessionId})
          </p>
        </div>
      </div>

      {/* 聊天记录区 */}
      <div className="messages-area">
        {messages.map((msg) => {
          const feedback = feedbacks[msg.id]
          const showFeedbackBox = activeFeedbackMsgId === msg.id
          
          // 只要是最后一条 bot 回复，就显示解决按钮（即使用户刚刚发了新消息还在 loading）
          const latestBotMsg = [...messages].reverse().find(m => m.role === 'bot');
          const isLatestBotMsg = msg.role === 'bot' && latestBotMsg && msg.id === latestBotMsg.id;

          return (
            <div key={msg.id} className={`message-wrapper ${msg.role}`}>
              <div className="message-content">
                <ReactMarkdown>{msg.content}</ReactMarkdown>
                
                {/* 如果是 Bot 回复且来自数据库(message_id>0)，则显示 Feedback */}
                {msg.role === 'bot' && msg.message_id > 0 && (
                  <div className="feedback-container">
                    <button 
                      className={`feedback-btn ${feedback?.rating === 'up' ? 'active' : ''}`}
                      onClick={() => handleFeedback(msg.id, msg.message_id, 'up')}
                      disabled={feedback?.submitted}
                    >
                      👍
                    </button>
                    <button 
                      className={`feedback-btn ${feedback?.rating === 'down' ? 'active' : ''}`}
                      onClick={() => handleFeedback(msg.id, msg.message_id, 'down')}
                      disabled={feedback?.submitted && feedback?.rating !== 'down'}
                    >
                      👎
                    </button>
                    {feedback?.submitted && <span className="feedback-thanks">感谢反馈！</span>}
                    
                    {/* 把问题已解决按钮跟点赞放一起，且只有最新一条消息显示 */}
                    {!isResolved && isLatestBotMsg && (
                      <button className="resolve-btn" onClick={handleResolve}>
                        ✅ 问题已解决
                      </button>
                    )}
                  </div>
                )}
                
                {/* 针对差评的意见输入框 */}
                {showFeedbackBox && (
                  <div className="feedback-form">
                    <textarea 
                      placeholder="很抱歉没能解决您的问题，请留下您的建议..." 
                      value={feedbackText}
                      onChange={e => setFeedbackText(e.target.value)}
                    />
                    <button onClick={() => submitFeedbackText(msg.id, msg.message_id)}>提交反馈</button>
                  </div>
                )}
              </div>
            </div>
          )
        })}
        
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
            placeholder="输入您遇到的问题 (如：WiFi怎么连？)"
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
