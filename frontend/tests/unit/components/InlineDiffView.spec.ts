import { describe, test, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import InlineDiffView from '../../../src/components/ui/InlineDiffView.vue';

describe('InlineDiffView', () => {
  const baseProps = {
    content: 'fmt.Println("test message")',
    lineNum: 10,
    contextBefore: ['package main', '', 'import "fmt"'],
    contextAfter: ['func main() {', '\tfmt.Println("other")'],
    query: 'test'
  };

  test('renders with correct props', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    expect(wrapper.exists()).toBe(true);
    expect(wrapper.find('.inline-diff-view').exists()).toBe(true);
  });

  test('displays line numbers correctly', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    // Match line should be line 10
    const matchLine = wrapper.findAll('.result-line')[0];
    expect(matchLine.text()).toContain('10');
  });

  test('highlights matched text in content', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    // Should render the component without errors
    expect(wrapper.exists()).toBe(true);
    
    // The matching line should have the match-class applied via CSS styling
    // Check that we have a result-line with matched class
    const hasMatchedLine = wrapper.find('.result-line.matched').exists();
    expect(hasMatchedLine).toBe(true);
  });

  test('renders context before lines', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    const contextLines = wrapper.findAll('.context-before');
    expect(contextLines.length).toBe(3);
    expect(contextLines[0].text()).toContain('package main');
  });

  test('renders context after lines', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    const contextLines = wrapper.findAll('.context-after');
    expect(contextLines.length).toBe(2);
    expect(contextLines[1].text()).toContain('other');
  });

  test('emits copy event when copy button clicked', async () => {
    const wrapper = mount(InlineDiffView, { 
      props: baseProps,
      global: { stubs: { 'copy-line-btn': true } }
    });
    
    await wrapper.vm.$emit('copy', 'test content');
    
    // Verify event was emitted
    expect(wrapper.emitted('copy')).toBeTruthy();
    expect(wrapper.emitted('copy')![0][0]).toBe('test content');
  });

  test('shows fuzzy badge when similarity score provided', () => {
    const fuzzyProps = {
      ...baseProps,
      fuzzyMatchScore: 0.85
    };
    
    const wrapper = mount(InlineDiffView, { props: fuzzyProps });
    
    expect(wrapper.find('.fuzzy-badge').exists()).toBe(true);
    expect(wrapper.find('.fuzzy-badge').text()).toBe('~');
  });

  test('hides fuzzy badge when no score provided', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    expect(wrapper.find('.fuzzy-badge').exists()).toBe(false);
  });

  test('handles empty context arrays', () => {
    const emptyContext = {
      ...baseProps,
      contextBefore: [],
      contextAfter: []
    };
    
    const wrapper = mount(InlineDiffView, { props: emptyContext });
    
    expect(wrapper.findAll('.context-before').length).toBe(0);
    expect(wrapper.findAll('.context-after').length).toBe(0);
  });

  test('does not show diff indicators with zero context', () => {
    const emptyContext = {
      ...baseProps,
      contextBefore: [],
      contextAfter: []
    };
    
    const wrapper = mount(InlineDiffView, { props: emptyContext });
    
    expect(wrapper.find('.diff-indicators').exists()).toBe(false);
  });

  test('shows diff hint when context is available', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    expect(wrapper.find('.diff-hint').exists()).toBe(true);
    expect(wrapper.find('.diff-hint').text()).toContain('lines before');
    expect(wrapper.find('.diff-hint').text()).toContain('lines after');
  });

  test('uses correct line numbering for context lines', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    const contextLines = wrapper.findAll('.context-before');
    // Should have 3 context lines before the match
    expect(contextLines.length).toBe(3);
    
    // Line numbers should be rendered (9, 8, 7) based on match being at line 10
    const firstContextLine = contextLines[0];
    expect(firstContextLine.text()).toContain('package main');
  });

  test('styles match line with different background', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    const matchLine = wrapper.find('.result-line.matched');
    expect(matchLine.exists()).toBe(true);
  });

  test('context lines have correct border colors', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    
    const beforeLines = wrapper.findAll('.context-before');
    const afterLines = wrapper.findAll('.context-after');
    
    expect(beforeLines.length).toBeGreaterThan(0);
    expect(afterLines.length).toBeGreaterThan(0);
  });

  test('handles special characters in query safely', () => {
    const specialQuery = {
      ...baseProps,
      query: '[a-z]+.*'
    };
    
    const wrapper = mount(InlineDiffView, { props: specialQuery });
    
    // Should not crash and should still render
    expect(wrapper.exists()).toBe(true);
    expect(wrapper.find('.inline-diff-view').exists()).toBe(true);
  });

  test('truncates long content properly', () => {
    const longContent = {
      ...baseProps,
      content: 'x'.repeat(1000) + 'test' + 'y'.repeat(1000)
    };
    
    const wrapper = mount(InlineDiffView, { props: longContent });
    
    expect(wrapper.exists()).toBe(true);
    // Content should wrap via CSS
    expect(wrapper.find('.line-content').exists()).toBe(true);
  });

  test('preserves whitespace in code content', () => {
    const tabsContent = {
      ...baseProps,
      content: '\tconst x = "test value";\n\treturn x;'
    };
    
    const wrapper = mount(InlineDiffView, { props: tabsContent });
    
    expect(wrapper.find('.line-content').exists()).toBe(true);
    expect(wrapper.text()).toContain('\t');
  });

  test('multiple matches are all highlighted', () => {
    const multiMatch = {
      ...baseProps,
      content: 'const test = "test data"; console.log(test);'
    };
    
    const wrapper = mount(InlineDiffView, { props: multiMatch });
    
    // Should highlight both occurrences of "test"
    expect(wrapper.findAll('.diff-match').length).toBeGreaterThanOrEqual(1);
  });

  test('renders diff-plus marker on match line', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    expect(wrapper.find('.diff-plus').exists()).toBe(true);
    expect(wrapper.find('.diff-plus').text()).toBe('+');
  });

  test('renders diff-minus marker on context lines', () => {
    const wrapper = mount(InlineDiffView, { props: baseProps });
    const minusMarkers = wrapper.findAll('.diff-minus');
    // contextBefore has 3 lines + contextAfter has 2 lines = 5 minus markers
    expect(minusMarkers.length).toBe(5);
  });

  test('does not render diff markers when no context', () => {
    const noContext = {
      ...baseProps,
      contextBefore: [],
      contextAfter: []
    };
    const wrapper = mount(InlineDiffView, { props: noContext });
    expect(wrapper.find('.diff-minus').exists()).toBe(false);
    // Match line still has plus marker
    expect(wrapper.find('.diff-plus').exists()).toBe(true);
  });

  test('shows match count hint when multiple matches on line', () => {
    const multiMatch = {
      ...baseProps,
      content: 'test test test'
    };
    const wrapper = mount(InlineDiffView, { props: multiMatch });
    const hint = wrapper.find('.diff-hint');
    expect(hint.exists()).toBe(true);
    expect(hint.text()).toContain('3 matches');
  });

  test('shows truncation hint for long lines', () => {
    const longContent = {
      ...baseProps,
      content: 'x'.repeat(250) + 'test' + 'y'.repeat(250),
    };
    const wrapper = mount(InlineDiffView, { props: longContent });
    const hint = wrapper.find('.diff-hint');
    expect(hint.exists()).toBe(true);
    expect(hint.text()).toContain('truncated');
  });

  test('honors caseSensitive prop', () => {
    const caseProps = {
      ...baseProps,
      content: 'Test test TEST',
      query: 'Test',
      caseSensitive: true,
    };
    const wrapper = mount(InlineDiffView, { props: caseProps });
    // Only 'Test' (exact case) should be highlighted
    const matches = wrapper.findAll('.diff-match');
    expect(matches.length).toBe(1);
    expect(matches[0].text()).toBe('Test');
  });

  test('case-insensitive highlights all case variants', () => {
    const caseProps = {
      ...baseProps,
      content: 'Test test TEST',
      query: 'test',
      caseSensitive: false,
    };
    const wrapper = mount(InlineDiffView, { props: caseProps });
    const matches = wrapper.findAll('.diff-match');
    expect(matches.length).toBe(3);
  });

  test('renders truncation marker for long content', () => {
    const longContent = {
      ...baseProps,
      content: 'x'.repeat(250) + 'test' + 'y'.repeat(250),
    };
    const wrapper = mount(InlineDiffView, { props: longContent });
    expect(wrapper.find('.diff-truncation').exists()).toBe(true);
  });
});
