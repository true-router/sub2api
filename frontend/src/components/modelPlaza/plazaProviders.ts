// Identify the model's manufacturer before considering its transport platform.
// Compatible gateways may serve models from several manufacturers.
const providers = [
  { id: 'anthropic', name: 'Anthropic', icon: 'Anthropic', model: /(?:^|[/.\s:_-])claude(?:$|[\s._-]|\d)/i, platforms: ['anthropic'] },
  { id: 'openai', name: 'OpenAI', icon: 'OpenAI', model: /(?:^|[/.\s:_-])(?:gpt(?:[\s._-]|\d)|chatgpt|o[1-9](?:$|[\s._-])|codex|dall-e|whisper|tts-1|text-embedding|text-moderation|omni-moderation|sora)/i, platforms: [] },
  { id: 'google', name: 'Google', icon: 'Google', model: /(?:^|[/.\s:_-])(?:gemini|gemma|learnlm|imagen|veo)(?:$|[\s._-]|\d)/i, platforms: ['gemini'] },
  { id: 'deepseek', name: 'DeepSeek', icon: 'DeepSeek', model: /(?:^|[/.\s:_-])deepseek/i, platforms: ['deepseek'] },
  { id: 'alibaba', name: 'Alibaba', icon: 'Alibaba', model: /(?:^|[/.\s:_-])(?:qwen|qwq|wan)(?:$|[\s._-]|\d)/i, platforms: ['qwen'] },
  { id: 'xai', name: 'xAI', icon: 'XAI', model: /(?:^|[/.\s:_-])grok(?:$|[\s._-]|\d)/i, platforms: ['grok'] },
  { id: 'moonshot', name: 'Moonshot AI', icon: 'Moonshot', model: /(?:^|[/.\s:_-])(?:kimi|moonshot)(?:$|[\s._-]|\d)/i, platforms: ['kimi'] },
  { id: 'zhipu', name: 'Zhipu AI', icon: 'Zhipu', model: /(?:^|[/.\s:_-])(?:glm|chatglm|cogview|cogvideo)(?:$|[\s._-]|\d)/i, platforms: ['zhipu'] },
  { id: 'minimax', name: 'MiniMax', icon: 'Minimax', model: /(?:^|[/.\s:_-])(?:minimax|abab|hailuo)(?:$|[\s._-]|\d)/i, platforms: ['minimax'] },
  { id: 'bytedance', name: 'ByteDance', icon: 'ByteDance', model: /(?:^|[/.\s:_-])(?:doubao|seed|seedance|seedream)(?:$|[\s._-]|\d)/i, platforms: [] },
  { id: 'meta', name: 'Meta', icon: 'Meta', model: /(?:^|[/.\s:_-])llama(?:$|[\s._-]|\d)/i, platforms: [] },
  { id: 'mistral', name: 'Mistral AI', icon: 'Mistral', model: /(?:^|[/.\s:_-])(?:mistral|mixtral|codestral|pixtral|voxtral|magistral|devstral|ministral)(?:$|[\s._-]|\d)/i, platforms: [] },
  { id: 'cohere', name: 'Cohere', icon: 'Cohere', model: /(?:^|[/.\s:_-])(?:command-r|command-a|c4ai|embed-(?:english|multilingual))/i, platforms: [] },
  { id: 'baidu', name: 'Baidu', icon: 'Baidu', model: /(?:^|[/.\s:_-])(?:ernie|wenxin)(?:$|[\s._-]|\d)/i, platforms: [] }
]
const other = { id: 'other', name: '', icon: '' }

export function modelProvider(name: string, platform: string) {
  return providers.find(provider => provider.model.test(name))
    ?? providers.find(provider => provider.platforms.includes(platform.toLowerCase()))
    ?? other
}
