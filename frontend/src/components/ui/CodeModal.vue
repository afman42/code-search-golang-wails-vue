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
        <div class="code-container">
          <pre class="code-block"><code ref="codeBlock" v-html="highlightedCode"></code></pre>
        </div>
      </div>
      
      <div class="modal-footer">
        <div class="modal-footer-info">
          Lines: {{ totalLines }} | Language: {{ detectedLanguage }}
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
import { defineComponent, ref, computed, onMounted, watch } from 'vue';
import type { PropType } from 'vue';

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
    const copied = ref(false);
    
    // Close modal when escape key is pressed
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && props.isVisible) {
        closeModal();
      }
    };
    
    onMounted(() => {
      document.addEventListener('keydown', handleEscape);
    });
    
    const closeModal = () => {
      document.removeEventListener('keydown', handleEscape);
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
        'vue': 'vue'
      };
      return languages[ext] || 'text';
    });
    
    // Get total number of lines in file
    const totalLines = computed(() => {
      if (!props.fileContent) return 0;
      return props.fileContent.split('\n').length;
    });
    
    // Highlight code with line numbers and query matches
    const highlightedCode = computed(() => {
      if (!props.fileContent) return '';
      
      // Split content into lines
      const lines = props.fileContent.split('\n');
      
      // Create HTML with line numbers
      let html = '';
      
      lines.forEach((line, index) => {
        const lineNumber = index + 1;
        let highlightedLine = escapeHtml(line);
        
        // Highlight query matches if query exists
        if (props.query) {
          try {
            const escapedQuery = escapeRegExp(props.query);
            const regex = new RegExp(`(${escapedQuery})`, 'gi');
            highlightedLine = highlightedLine.replace(regex, '<mark class="highlight-match">$1</mark>');
          } catch (e) {
            // If regex fails, don't highlight (e.g. invalid regex)
          }
        }
        
        // Add line with number
        html += `<span class="line-number">${lineNumber}</span><span class="code-line">${highlightedLine || ' '}</span>\n`;
      });
      
      return html;
    });
    
    // Utility function to escape HTML
    const escapeHtml = (unsafe: string): string => {
      if (!unsafe) return '';
      return unsafe
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
    };
    
    // Utility function to escape regex special characters
    const escapeRegExp = (string: string): string => {
      return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    };
    
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
    
    // Watch for visibility changes to reset copied status
    watch(() => props.isVisible, (newVal) => {
      if (!newVal) {
        copied.value = false;
      }
    });
    
    return {
      codeBlock,
      copied,
      closeModal,
      truncatePath,
      detectedLanguage,
      totalLines,
      highlightedCode,
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
  background-color: #ffffff;
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
  border-bottom: 1px solid #e0e0e0;
  background-color: #f5f5f5;
}

.modal-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
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
  color: #666;
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
  background-color: #e0e0e0;
  color: #333;
}

.modal-content {
  flex: 1;
  overflow: auto;
  padding: 0;
  background-color: #f8f8f8;
}

.code-container {
  overflow: auto;
  max-height: 70vh;
}

.code-block {
  margin: 0;
  padding: 0;
  background-color: #f8f8f8;
  border-radius: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  line-height: 1.4;
}

.code-block code {
  display: block;
  padding: 0;
}

/* Line numbers styling */
.line-number {
  display: inline-block;
  width: 50px;
  padding: 0 12px;
  text-align: right;
  color: #999;
  background-color: #f0f0f0;
  border-right: 1px solid #e0e0e0;
  user-select: none;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
}

.code-line {
  display: inline-block;
  padding: 0 12px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 14px;
  white-space: pre;
}

/* Highlight matches */
.highlight-match {
  background-color: #ffeb3b;
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

.mark {
  background-color: #ffeb3b;
  padding: 1px 2px;
  border-radius: 2px;
  font-weight: bold;
}

.modal-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-top: 1px solid #e0e0e0;
  background-color: #f5f5f5;
}

.modal-footer-info {
  color: #666;
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

.copy-button:active {
  background-color: #3d8b40;
}

/* Scrollbar styling */
.modal-content::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.modal-content::-webkit-scrollbar-track {
  background: #f1f1f1;
}

.modal-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 4px;
}

.modal-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>