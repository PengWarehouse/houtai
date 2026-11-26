<script setup>
import { computed, nextTick, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { chatAI, extractAI, saveInfoItems } from '@/api'

let seq = 0
function newMessage(role, content, extra = {}) {
  return { id: seq++, role, content, ...extra }
}

const messages = ref([
  newMessage('assistant', '你好，我是 AI 助手。点上方「新闻」或「视频」可进入资讯提取模式，我会整理候选条目供你勾选；不勾选不会写入数据库。')
])
const input = ref('')
const sending = ref(false)
const activeKind = ref('') // 空 = 普通对话；news / video = 资讯提取模式
const scrollRef = ref()
const inputRef = ref()

const quickPrompts = [
  { label: '新闻', kind: 'news', icon: 'NewsPaper' },
  { label: '视频', kind: 'video', icon: 'VideoCamera' }
]

const kindLabel = computed(() => {
  if (activeKind.value === 'news') return '新闻'
  if (activeKind.value === 'video') return '视频'
  return '普通问询'
})

const inputPlaceholder = computed(() =>
  activeKind.value
    ? `描述你想找的${kindLabel.value}，例如：整理两条关于人工智能的${kindLabel.value}并给出链接`
    : '输入你的问题，Enter 发送，Shift + Enter 换行'
)

function scrollToBottom() {
  nextTick(() => {
    if (scrollRef.value) scrollRef.value.scrollTop = scrollRef.value.scrollHeight
  })
}

function toggleKind(kind) {
  activeKind.value = activeKind.value === kind ? '' : kind
  inputRef.value?.focus()
}

async function send() {
  const text = input.value.trim()
  if (!text || sending.value) return

  const kind = activeKind.value
  messages.value.push(newMessage('user', text))
  input.value = ''
  sending.value = true
  scrollToBottom()

  try {
    if (kind) {
      // 资讯模式：只提取候选条目，不写库
      const data = await extractAI(kind, text)
      const items = (data?.items || []).map((item) => ({
        title: item.title || '',
        url: item.url || '',
        source: item.source || '',
        published_at: item.published_at || ''
      }))
      messages.value.push(
        newMessage(
          'assistant',
          items.length
            ? `已提取 ${items.length} 条${kindLabel.value}候选，勾选后点击「加入数据库」；不勾选则不会保存。`
            : `没有提取到可用的${kindLabel.value}条目，可以换个描述再试。`,
          {
            kind,
            items,
            selected: [], // 默认不勾选，避免误入库
            saving: false,
            saved: false,
            saveResult: null
          }
        )
      )
    } else {
      // 普通模式：多轮对话
      const history = messages.value
        .filter((m) => m.role === 'user' || m.role === 'assistant')
        .map((m) => ({ role: m.role, content: m.content }))
      const data = await chatAI(history)
      messages.value.push(newMessage('assistant', data?.content || '（无回复）'))
    }
  } catch {
    // 失败原因由 axios 拦截器统一弹出，撤掉这条用户消息避免对话错乱
    if (messages.value.at(-1)?.role === 'user') messages.value.pop()
    input.value = text
  } finally {
    sending.value = false
    scrollToBottom()
  }
}

async function saveCandidates(msg) {
  if (!msg.selected?.length || msg.saving || msg.saved) return
  const chosen = msg.items.filter((_, index) => msg.selected.includes(index))
  if (!chosen.length) return

  msg.saving = true
  try {
    const data = await saveInfoItems({
      kind: msg.kind,
      items: chosen.map(({ title, url, source, published_at }) => ({
        title,
        url,
        source,
        published_at
      }))
    })
    msg.saveResult = data
    msg.saved = true
    if (data?.inserted > 0) {
      ElMessage.success(`已加入数据库 ${data.inserted} 条${data.skipped ? `，跳过 ${data.skipped} 条` : ''}`)
    } else {
      ElMessage.warning('没有新增条目，详见下方的跳过原因')
    }
  } catch {
    // axios 拦截器已展示失败原因
  } finally {
    msg.saving = false
    scrollToBottom()
  }
}

function reset() {
  messages.value = [
    newMessage('assistant', '你好，我是 AI 助手。有什么可以帮你？')
  ]
  input.value = ''
}
</script>

<template>
  <div class="ai-page page-fill">
    <div class="ai-header">
      <div class="ai-header-left">
        <div class="ai-header-icon"><el-icon :size="20"><component is="MagicStick" /></el-icon></div>
        <div class="ai-title-wrap">
          <span class="ai-title">AI 问询</span>
          <span class="ai-subtitle">智能问答 · 资讯整理 · 勾选入库</span>
        </div>
        <el-tag v-if="activeKind" size="small" round class="ai-mode-tag" closable @close="activeKind = ''">
          {{ kindLabel }}提取模式
        </el-tag>
      </div>
      <el-button text :icon="'Refresh'" :disabled="sending" @click="reset">新对话</el-button>
    </div>

    <div class="ai-chat" ref="scrollRef">
      <div
        v-for="msg in messages"
        :key="msg.id"
        class="ai-msg"
        :class="msg.role === 'user' ? 'is-user' : 'is-assistant'"
      >
        <div class="ai-avatar">
          <el-icon v-if="msg.role === 'user'" :size="16"><component is="User" /></el-icon>
          <el-icon v-else :size="18"><component is="MagicStick" /></el-icon>
        </div>
        <div class="ai-bubble">
          <div class="ai-bubble-text">{{ msg.content }}</div>

          <!-- 提取结果：勾选后才入库 -->
          <div v-if="msg.items?.length" class="ai-candidates">
            <div class="ai-candidates-title">
              <el-icon :size="14"><component is="List" /></el-icon>
              候选{{ msg.kind === 'video' ? '视频' : '新闻' }} · 勾选要入库的条目
            </div>
            <el-checkbox-group v-model="msg.selected" class="ai-candidate-list">
              <div v-for="(item, index) in msg.items" :key="index" class="ai-candidate">
                <el-checkbox :value="index" :disabled="msg.saved">
                  <span class="ai-candidate-title">{{ item.title }}</span>
                </el-checkbox>
                <div class="ai-candidate-meta">
                  <span v-if="item.source" class="ai-candidate-source">{{ item.source }}</span>
                  <span class="ai-candidate-url">{{ item.url }}</span>
                </div>
              </div>
            </el-checkbox-group>

            <div class="ai-candidate-footer">
              <span class="ai-candidate-hint">未勾选不会写入数据库</span>
              <el-button
                type="primary"
                size="small"
                round
                :loading="msg.saving"
                :disabled="!msg.selected.length || msg.saved"
                @click="saveCandidates(msg)"
              >
                {{ msg.saved ? '已入库' : `加入数据库（${msg.selected.length}）` }}
              </el-button>
            </div>

            <div v-if="msg.saveResult" class="ai-save-result">
              <div>成功 {{ msg.saveResult.inserted }} 条，跳过 {{ msg.saveResult.skipped }} 条</div>
              <ul v-if="msg.saveResult.failures?.length">
                <li v-for="(failure, index) in msg.saveResult.failures" :key="index">
                  {{ failure.title }}：{{ failure.reason }}
                </li>
              </ul>
            </div>
          </div>
        </div>
      </div>
      <div v-if="sending" class="ai-msg is-assistant">
        <div class="ai-avatar"><el-icon :size="18"><component is="MagicStick" /></el-icon></div>
        <div class="ai-bubble is-thinking">
          <span class="ai-dot" /><span class="ai-dot" /><span class="ai-dot" />
        </div>
      </div>
    </div>

    <div class="ai-composer">
      <div class="ai-quick">
        <span class="ai-quick-label">资讯类型</span>
        <button
          v-for="q in quickPrompts"
          :key="q.kind"
          type="button"
          class="ai-quick-chip"
          :class="{ active: activeKind === q.kind }"
          :disabled="sending"
          @click="toggleKind(q.kind)"
        >
          <el-icon :size="14"><component :is="q.icon" /></el-icon>
          {{ q.label }}
        </button>
        <span class="ai-quick-tip">再次点击可切回普通问询</span>
      </div>
      <div class="ai-input-shell">
        <el-input
          ref="inputRef"
          v-model="input"
          type="textarea"
          :rows="2"
          resize="none"
          :placeholder="inputPlaceholder"
          :disabled="sending"
          @keydown.enter.exact.prevent="send"
        />
        <el-button
          class="ai-send-btn"
          type="primary"
          round
          :icon="'Promotion'"
          :loading="sending"
          :disabled="!input.trim() || sending"
          @click="send"
        >
          发送
        </el-button>
      </div>
      <div class="ai-tip">
        {{ activeKind ? '当前为资讯提取模式：AI 只返回候选条目，勾选后才写入数据库' : '由 DeepSeek 提供支持 · 回复仅供参考' }}
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.ai-page {
  gap: 14px;
  padding: 2px;
  background: linear-gradient(180deg, rgba(233, 240, 255, 0.55) 0%, rgba(247, 249, 253, 0.15) 45%, rgba(238, 243, 251, 0.4) 100%);
}

.ai-header {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-radius: 14px;
  background-color: #fff;
  box-shadow: 0 2px 14px rgba(31, 45, 75, 0.06);
}

.ai-header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ai-header-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: linear-gradient(135deg, #8199ff, #4f6ef7);
  color: #fff;
  box-shadow: 0 6px 14px rgba(79, 110, 247, 0.28);
}

.ai-title-wrap {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.ai-title {
  color: #1f2d4d;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.2px;
}

.ai-subtitle {
  color: #9aa4b6;
  font-size: 12px;
}

.ai-mode-tag {
  border-color: transparent;
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 500;
}

.ai-chat {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 20px;
  min-height: 0;
  overflow-y: auto;
  padding: 24px 20px;
  border-radius: 14px;
  background-color: #fff;
  box-shadow: 0 2px 16px rgba(31, 45, 75, 0.06);
  scrollbar-width: thin;
  scrollbar-color: #dbe2ee transparent;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-thumb {
    border-radius: 3px;
    background-color: #dbe2ee;
  }
}

.ai-msg {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  max-width: 86%;

  &.is-user {
    flex-direction: row-reverse;
    align-self: flex-end;

    .ai-avatar {
      background: linear-gradient(135deg, #6d8bff, #4264e8);
      box-shadow: 0 4px 10px rgba(66, 100, 232, 0.25);
    }

    .ai-bubble {
      border-color: transparent;
      border-radius: 14px 14px 4px 14px;
      background: linear-gradient(135deg, #5b78f0, #4264e8);
      box-shadow: 0 6px 16px rgba(66, 100, 232, 0.22);
      color: #fff;
    }
  }
}

.ai-avatar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, #9b8cff, #6f61e8);
  box-shadow: 0 4px 10px rgba(111, 97, 232, 0.22);
  color: #fff;
}

.ai-bubble {
  padding: 12px 16px;
  border: 1px solid #edf0f6;
  border-radius: 14px 14px 14px 4px;
  background-color: #f7f9fc;
  box-shadow: 0 2px 8px rgba(31, 45, 75, 0.04);
  color: #303133;
  line-height: 1.75;
  word-break: break-word;
}

.ai-bubble-text {
  white-space: pre-wrap;
}

.ai-bubble.is-thinking {
  display: flex;
  align-items: center;
  gap: 5px;
  min-height: 42px;
  padding: 12px 18px;
}

.ai-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background-color: #9aa4b6;
  animation: ai-dot-pulse 1.2s infinite ease-in-out;

  &:nth-child(2) {
    animation-delay: 0.15s;
  }

  &:nth-child(3) {
    animation-delay: 0.3s;
  }
}

@keyframes ai-dot-pulse {
  0%, 60%, 100% {
    opacity: 0.35;
    transform: translateY(0);
  }
  30% {
    opacity: 1;
    transform: translateY(-4px);
  }
}

.ai-candidates {
  margin-top: 12px;
  padding: 14px;
  border: 1px solid #e8edf7;
  border-radius: 12px;
  background: linear-gradient(180deg, #fbfcff, #f6f9fd);
}

.ai-candidates-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
  color: #2b3550;
  font-size: 13px;
  font-weight: 700;
}

.ai-candidate-list {
  display: block;
}

.ai-candidate {
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid #edf1f8;
  border-radius: 10px;
  background-color: #fff;
  transition: border-color 0.2s, box-shadow 0.2s, transform 0.2s;

  &:hover {
    border-color: #c9d6ff;
    box-shadow: 0 4px 12px rgba(79, 110, 247, 0.08);
    transform: translateX(2px);
  }

  &:last-child {
    margin-bottom: 0;
  }
}

.ai-candidate-title {
  font-size: 13px;
  font-weight: 600;
}

.ai-candidate-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 10px;
  padding-left: 24px;
  color: #8b95a9;
  font-size: 12px;
  line-height: 1.5;
}

.ai-candidate-source {
  padding: 1px 6px;
  border-radius: 4px;
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.ai-candidate-url {
  word-break: break-all;
}

.ai-candidate-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
}

.ai-candidate-hint {
  color: #98a2b6;
  font-size: 12px;
}

.ai-save-result {
  margin-top: 12px;
  padding: 8px 10px;
  border-radius: 8px;
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary-dark-2);
  font-size: 12px;

  ul {
    margin: 6px 0 0;
    padding-left: 18px;
  }
}

.ai-composer {
  flex-shrink: 0;
  padding: 14px 18px 12px;
  border-radius: 14px;
  background-color: #fff;
  box-shadow: 0 2px 16px rgba(31, 45, 75, 0.06);
}

.ai-quick {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.ai-quick-label {
  color: #606b80;
  font-size: 12px;
  font-weight: 600;
}

.ai-quick-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 13px;
  border: 1px solid #e4e9f2;
  border-radius: 999px;
  background-color: #f7f9fc;
  color: #5d6779;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover:not(.active):not(:disabled) {
    border-color: #b9c8ff;
    color: var(--el-color-primary);
  }

  &.active {
    border-color: transparent;
    background: linear-gradient(135deg, #5b78f0, #4264e8);
    box-shadow: 0 4px 10px rgba(66, 100, 232, 0.25);
    color: #fff;
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }
}

.ai-quick-tip {
  color: #b0b8c6;
  font-size: 12px;
}

.ai-input-shell {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  padding: 10px 10px 10px 12px;
  border: 1px solid #e4e9f2;
  border-radius: 12px;
  background-color: #fbfcff;
  transition: border-color 0.2s, box-shadow 0.2s, background-color 0.2s;

  &:focus-within {
    border-color: #8ba3ff;
    background-color: #fff;
    box-shadow: 0 0 0 3px rgba(91, 120, 240, 0.1);
  }

  :deep(.el-textarea__inner) {
    padding: 6px 0;
    background-color: transparent;
    box-shadow: none !important;
    line-height: 1.6;
  }
}

.ai-send-btn {
  min-width: 92px;
  box-shadow: 0 4px 12px rgba(66, 100, 232, 0.22);
}

.ai-tip {
  margin-top: 10px;
  color: #a0a9ba;
  font-size: 12px;
  text-align: center;
}
</style>
