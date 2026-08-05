import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import EditorStatusDisplay from '@/components/ui/EditorStatusDisplay.vue';

describe('EditorStatusDisplay.vue', () => {
  it('renders nothing when neither detecting nor complete', () => {
    const wrapper = mount(EditorStatusDisplay, {
      props: {
        editorDetectionStatus: {
          detectingEditors: false,
          detectionComplete: false,
          message: '',
          detectionProgress: 0,
          detectedEditors: [],
        },
      },
    });

    expect(wrapper.find('.editor-detection-status').exists()).toBe(false);
  });

  it('shows detection animation when detecting', () => {
    const wrapper = mount(EditorStatusDisplay, {
      props: {
        editorDetectionStatus: {
          detectingEditors: true,
          detectionComplete: false,
          message: 'Detecting editors...',
          detectionProgress: 50,
          detectedEditors: [],
        },
      },
    });

    expect(wrapper.find('.editor-detection-status').exists()).toBe(true);
    expect(wrapper.find('.detection-animation').exists()).toBe(true);
    expect(wrapper.find('.spinner').exists()).toBe(true);
    expect(wrapper.text()).toContain('Detecting editors...');
    expect(wrapper.find('.progress-text').text()).toContain('50%');
  });

  it('shows detection complete message', () => {
    const wrapper = mount(EditorStatusDisplay, {
      props: {
        editorDetectionStatus: {
          detectingEditors: false,
          detectionComplete: true,
          message: 'Detection complete',
          detectionProgress: 100,
          detectedEditors: [],
        },
      },
    });

    expect(wrapper.find('.completed').exists()).toBe(true);
    expect(wrapper.find('.status-icon').exists()).toBe(true);
    expect(wrapper.text()).toContain('Detection complete');
  });

  it('shows detected editors list', () => {
    const editors = ['VS Code', 'Sublime Text', 'Vim'];
    const wrapper = mount(EditorStatusDisplay, {
      props: {
        editorDetectionStatus: {
          detectingEditors: false,
          detectionComplete: true,
          message: 'Found installed editors',
          detectionProgress: 100,
          detectedEditors: editors,
        },
      },
    });

    expect(wrapper.text()).toContain('Found editors: VS Code, Sublime Text, Vim');
  });

  it('handles progress updates correctly', async () => {
    const wrapper = mount(EditorStatusDisplay, {
      props: {
        editorDetectionStatus: {
          detectingEditors: true,
          detectionComplete: false,
          message: 'Scanning for editors',
          detectionProgress: 25,
          detectedEditors: [],
        },
      },
    });

    const progressBar = wrapper.find('.progress-fill');
    expect(progressBar.element.getAttribute('style')).toContain('width: 25%');

    await wrapper.setProps({
      editorDetectionStatus: {
        ...wrapper.props().editorDetectionStatus,
        detectionProgress: 75,
      },
    });

    const updatedBar = wrapper.find('.progress-fill');
    expect(updatedBar.element.getAttribute('style')).toContain('width: 75%');
  });

  it('renders empty status when no data provided', () => {
    const wrapper = mount(EditorStatusDisplay, {});

    expect(wrapper.find('.editor-detection-status').exists()).toBe(false);
  });

  it('applies correct styling for completed state', () => {
    const wrapper = mount(EditorStatusDisplay, {
      props: {
        editorDetectionStatus: {
          detectingEditors: false,
          detectionComplete: true,
          message: 'Done',
          detectionProgress: 100,
          detectedEditors: [],
        },
      },
    });

    expect(wrapper.find('.editor-detection-status').classes()).toContain('completed');
  });
});
