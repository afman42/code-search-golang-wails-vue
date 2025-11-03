<template>
  <div v-if="isVisible" class="modal-overlay" @click="closeModal">
    <div class="modal-container" @click.stop>
      <div class="modal-header">
        <h3 class="modal-title">File Preview: {{ truncatePath(filePath) }}</h3>
        <button class="modal-close-button" @click="closeModal">
          <span>&times;</span>
        </button>
      </div>
      
      <div class="modal-content">
        <div class="code-container" ref="codeContainerRef">
          <pre class="code-block"><code ref="codeBlock" v-html="highlightedCode"></code></pre>
        </div>
      </div>
      
      <div class="modal-footer">
        <div class="modal-footer-info">
          Lines: {{ totalLines }} | Language: {{ detectedLanguage }}
          <span v-if="totalMatches > 0"> | Matches: {{ totalMatches }}</span>
        </div>
        <button class="copy-button" @click="copyToClipboard">
          <span v-if="copied">Copied!</span>
          <span v-else>Copy to Clipboard</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed, onMounted, nextTick } from 'vue';
import hljs from 'highlight.js';

export default defineComponent({
  name: 'CodeModal',
  props: {
    isVisible: {
      type: Boolean,
      required: true
    },
    filePath: {
      type: String,
      required: true
    },
    fileContent: {
      type: String,
      required: true
    },
    query: {
      type: String,
      default: ''
    }
  },
  emits: ['close', 'copy'],
  setup(props, { emit }) {
    const codeBlock = ref<HTMLElement | null>(null);
    const codeContainerRef = ref<HTMLElement | null>(null);
    const copied = ref(false);
    
    const closeModal = () => {
      emit('close');
    };
    
    // Truncate long file paths
    const truncatePath = (path: string): string => {
      if (!path) return '';
      const maxLength = 50;
      if (path.length <= maxLength) return path;
      const parts = path.split('/');
      if (parts.length > 1) {
        return '...' + parts.slice(-2).join('/');
      }
      return path.substring(path.length - maxLength);
    };
    
    // Detect programming language from file extension
    const detectedLanguage = computed(() => {
      if (!props.filePath) return 'text';
      const ext = props.filePath.split('.').pop()?.toLowerCase() || '';
      const languages: Record<string, string> = {
        'go': 'go',
        'js': 'javascript',
        'ts': 'typescript',
        'java': 'java',
        'py': 'python',
        'rb': 'ruby',
        'php': 'php',
        'cpp': 'cpp',
        'hpp': 'cpp',
        'h': 'c',
        'c': 'c',
        'html': 'html',
        'htm': 'html',
        'xml': 'xml',
        'css': 'css',
        'scss': 'scss',
        'sass': 'sass',
        'json': 'json',
        'yaml': 'yaml',
        'yml': 'yaml',
        'md': 'markdown',
        'sql': 'sql',
        'sh': 'bash',
        'bash': 'bash',
        'rs': 'rust',
        'swift': 'swift',
        'kt': 'kotlin',
        'scala': 'scala',
        'dart': 'dart',
        'lua': 'lua',
        'pl': 'perl',
        'r': 'r',
        'coffee': 'coffeescript',
        'vue': 'vue',
        'jsx': 'jsx',
        'tsx': 'tsx'
      };
      return languages[ext] || 'text';
    });
    
    // Get total number of lines in file
    const totalLines = computed(() => {
      if (!props.fileContent) return 0;
      return props.fileContent.split('\n').length;
    });
    
    // Highlight code with syntax highlighting and line numbers
    const highlightedCode = computed(() => {
      if (!props.fileContent) return '';
      
      // First, apply syntax highlighting
      const language = detectedLanguage.value;
      let highlightedCode = hljs.highlight(props.fileContent, { language: language }).value;
      
      // Split code into lines
      const lines = highlightedCode.split(/\r?\n/);
      let html = '';
      
      lines.forEach((line, index) => {
        const lineNumber = index + 1;
        let highlightedLine = line;
        
        // Highlight query matches if query exists
        if (props.query) {
          try {
            // Use a different approach to highlight query matches while preserving existing syntax highlighting
            const regex = new RegExp(`(${props.query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
            highlightedLine = highlightedLine.replace(regex, '<mark class="highlight-match">$1</mark>');
          } catch (e) {
            // If regex fails, don't highlight
          }
        }
        
        // Add line with number - make sure to handle empty lines properly
        const lineContent = highlightedLine || '<span class="hljs-comment"> </span>';
        html += `<span class="line-number" data-line="${lineNumber}">${lineNumber}</span><span class="code-line">${lineContent}</span>\n`;
      });
      
      return html;
    });
    
    // Total number of matches
    const totalMatches = computed(() => {
      if (!props.query || !props.fileContent) return 0;
      
      try {
        const regex = new RegExp(props.query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
        const matches = props.fileContent.match(regex);
        return matches ? matches.length : 0;
      } catch (e) {
        // If regex fails, return 0
        return 0;
      }
    });
    
    // Copy file content to clipboard
    const copyToClipboard = () => {
      navigator.clipboard.writeText(props.fileContent)
        .then(() => {
          copied.value = true;
          // Reset copied status after 2 seconds
          setTimeout(() => {
            copied.value = false;
          }, 2000);
          
          // Emit copy event
          emit('copy');
        })
        .catch(err => {
          console.error('Failed to copy:', err);
        });
    };
    
    return {
      codeBlock,
      codeContainerRef,
      copied,
      closeModal,
      truncatePath,
      detectedLanguage,
      totalLines,
      highlightedCode,
      totalMatches,
      copyToClipboard
    };
  }
});
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-container {
  background-color: #333;
  border-radius: 8px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  width: 90%;
  max-width: 1200px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #555;
  background-color: #2d2d2d;
}

.modal-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: calc(100% - 40px);
}

.modal-close-button {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #ccc;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background-color 0.2s;
}

.modal-close-button:hover {
  background-color: #555;
  color: #fff;
}

.modal-content {
  flex: 1;
  overflow: auto;
  padding: 0;
  background-color: #333;
}

.code-container {
  overflow: auto;
  max-height: calc(70vh - 60px);
}

.code-block {
  margin: 0;
  padding: 0;
  background-color: #333;
  border-radius: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.4;
}

.code-block code {
  display: block;
  padding: 0;
  background-color: #333 !important;
  color: #fff;
}

/* Line numbers styling */
.line-number {
  display: inline-block;
  width: 50px;
  padding: 0 12px;
  text-align: right;
  color: #888;
  background-color: #222;
  border-right: 1px solid #555;
  user-select: none;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  position: relative;
  vertical-align: top;
  line-height: 1.4;
}

.code-line {
  display: inline-block;
  padding: 0 12px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  white-space: pre;
  vertical-align: top;
  line-height: 1.4;
}

/* Highlight matches - ensure they stand out against the Agate theme */
.highlight-match {
  background-color: #ffeb3b;
  color: #000 !important;
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

.modal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-top: 1px solid #555;
  background-color: #2d2d2d;
  color: #fff;
}

.modal-footer-info {
  color: #ccc;
  font-size: 14px;
}

.copy-button {
  background-color: #4caf50;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.copy-button:hover {
  background-color: #45a049;
}

/* Scrollbar styling */
.modal-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.modal-content::-webkit-scrollbar-track {
  background: #222;
}

.modal-content::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.modal-content::-webkit-scrollbar-thumb:hover {
  background: #666;
}
</style>