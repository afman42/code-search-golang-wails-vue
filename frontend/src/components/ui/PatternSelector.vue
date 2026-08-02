<template>
  <div class="control-group pattern-group">
    <label>Select Patterns</label>
    
    <!-- Exclude patterns -->
    <div class="pattern-section">
      <span class="section-title">Exclude:</span>
      <div class="pattern-list" v-if="excludePatterns.length > 0">
        <span
          v-for="(pattern, index) in excludePatterns"
          :key="'exclude-' + index"
          class="pattern-tag"
        >
          {{ pattern }}
          <button
            type="button"
            class="remove-btn"
            @click="$emit('removePattern', 'exclude', index)"
          >
            ×
          </button>
        </span>
      </div>
      <select v-else @change="addPatternFromSelect('exclude')" class="pattern-select">
        <option value="">Add exclusion...</option>
        <option v-for="opt in availableExcludeOptions" :key="opt" :value="opt">
          {{ opt }}
        </option>
      </select>
      <input
        v-model="customExcludePattern"
        @keyup.enter="addCustomPattern('exclude')"
        placeholder="Custom pattern..."
        class="custom-pattern-input"
      />
      <button @click="addCustomPattern('exclude')" class="btn-add">Add</button>
    </div>

    <!-- Allowed file types -->
    <div class="pattern-section">
      <span class="section-title">Allowed files:</span>
      <div class="pattern-list" v-if="allowedFileTypes.length > 0">
        <span
          v-for="(type, index) in allowedFileTypes"
          :key="'allow-' + index"
          class="pattern-tag"
        >
          {{ type }}
          <button
            type="button"
            class="remove-btn"
            @click="$emit('removePattern', 'allow', index)"
          >
            ×
          </button>
        </span>
      </div>
      <select v-else @change="addPatternFromSelect('allow')" class="pattern-select">
        <option value="">Add file type...</option>
        <option v-for="opt in availableAllowOptions" :key="opt" :value="opt">
          {{ opt }}
        </option>
      </select>
      <input
        v-model="customAllowType"
        @keyup.enter="addCustomPattern('allow')"
        placeholder="Custom type (e.g., .js)..."
        class="custom-pattern-input"
      />
      <button @click="addCustomPattern('allow')" class="btn-add">Add</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, defineOptions } from 'vue';

defineOptions({
  name: 'PatternSelector',
});


interface Props {
  excludePatterns?: string[];
  allowedFileTypes?: string[];
}

const props = withDefaults(defineProps<Props>(), {
  excludePatterns: () => [],
  allowedFileTypes: () => [],
});

const emit = defineEmits<{
  update: [patterns: { exclude: string[]; allow: string[] }];
  removePattern: [type: 'exclude' | 'allow', index: number];
}>();

const customExcludePattern = ref('');
const customAllowType = ref('');

const availableExcludeOptions = ['node_modules', '.git', 'vendor', 'dist', 'build', 'bin'];
const availableAllowOptions = ['.go', '.ts', '.tsx', '.js', '.vue', '.py', '.java', '.css', '.html'];

watch(() => props.excludePatterns, (newVal) => {
  if (!newVal || newVal.length === 0) {
    customExcludePattern.value = '';
  }
});

watch(() => props.allowedFileTypes, (newVal) => {
  if (!newVal || newVal.length === 0) {
    customAllowType.value = '';
  }
});

const addPatternFromSelect = (type: 'exclude' | 'allow') => {
  const selectElement = event?.target as HTMLSelectElement;
  if (!selectElement) return;
  
  const pattern = selectElement.value;
  selectElement.selectedIndex = 0;
  
  if (!pattern) return;

  if (type === 'exclude') {
    const newExclude = [...props.excludePatterns, pattern];
    emit('update', { exclude: newExclude, allow: props.allowedFileTypes });
  } else {
    const newAllow = [...props.allowedFileTypes, pattern];
    emit('update', { exclude: props.excludePatterns, allow: newAllow });
  }
};

const addCustomPattern = (type: 'exclude' | 'allow') => {
  const inputKey = type === 'exclude' ? customExcludePattern : customAllowType;
  
  if (!inputKey.value) return;
  
  const isExclude = type === 'exclude';
  const currentList = isExclude ? props.excludePatterns : props.allowedFileTypes;
  const newList = [...currentList, inputKey.value];
  
  emit('update', {
    exclude: isExclude ? newList : props.excludePatterns,
    allow: isExclude ? props.allowedFileTypes : newList,
  });
  
  if (type === 'exclude') {
    customExcludePattern.value = '';
  } else {
    customAllowType.value = '';
  }
};
</script>

<style scoped>
.pattern-group {
  margin-top: 1rem;
  padding: 1rem;
  background-color: #f8f9fa;
  border-radius: 0.25rem;
}

.control-group label {
  display: block;
  font-weight: 600;
  margin-bottom: 0.75rem;
  color: #495057;
}

.pattern-section {
  margin-bottom: 0.75rem;
}

.section-title {
  display: block;
  font-size: 0.8rem;
  color: #6c757d;
  margin-bottom: 0.5rem;
}

.pattern-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.pattern-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  background-color: #e9ecef;
  border-radius: 0.25rem;
  font-size: 0.8rem;
}

.remove-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1rem;
  line-height: 1;
  padding: 0;
  opacity: 0.5;
  transition: opacity 0.2s;
}

.remove-btn:hover {
  opacity: 1;
}

.pattern-select {
  padding: 0.375rem;
  border: 1px solid #ced4da;
  border-radius: 0.25rem;
  font-size: 0.8rem;
  background-color: white;
}

.custom-pattern-input {
  width: 100%;
  max-width: 200px;
  padding: 0.375rem;
  border: 1px solid #ced4da;
  border-radius: 0.25rem;
  font-size: 0.8rem;
  margin-right: 0.5rem;
}

.btn-add {
  padding: 0.375rem 0.75rem;
  background-color: #6c757d;
  color: white;
  border: none;
  border-radius: 0.25rem;
  cursor: pointer;
  font-size: 0.8rem;
}

.btn-add:hover {
  background-color: #5a6268;
}
</style>
