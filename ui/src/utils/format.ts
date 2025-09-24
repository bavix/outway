import { QTYPE_NAMES } from '../providers/types.js';

/**
 * Format DNS query type number to human readable name
 */
export function formatQType(qtype: number): string {
  return QTYPE_NAMES[qtype] || `TYPE${qtype}`;
}

/**
 * Format upstream string for display
 */
export function formatUpstream(upstream: string): string {
  // Truncate very long upstream strings
  if (upstream.length > 50) {
    return upstream.substring(0, 47) + '...';
  }
  return upstream;
}

/**
 * Format pattern for display (handle wildcards)
 */
export function formatPattern(pattern: string): string {
  if (pattern.length > 40) {
    return pattern.substring(0, 37) + '...';
  }
  return pattern;
}

/**
 * Format interface name for display
 */
export function formatInterface(iface: string): string {
  return iface || 'default';
}

/**
 * Format status for display
 */
export function formatStatus(status: 'ok' | 'error'): string {
  return status === 'ok' ? 'OK' : 'Error';
}

/**
 * Format timestamp to ISO string for display
 */
export function formatTimestamp(timestamp: string | number): string {
  const date = new Date(typeof timestamp === 'string' ? timestamp : timestamp * 1000);
  return date.toLocaleTimeString('en-US', {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  });
}

/**
 * Truncate text to specified length with ellipsis
 */
export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) {
    return text;
  }
  return text.substring(0, maxLength - 3) + '...';
}

/**
 * Validate and format DNS pattern
 */
export function validatePattern(pattern: string): { valid: boolean; error?: string } {
  if (!pattern.trim()) {
    return { valid: false, error: 'Pattern is required' };
  }
  
  if (pattern.length > 255) {
    return { valid: false, error: 'Pattern too long' };
  }
  
  // Basic DNS name validation
  const dnsRegex = /^(\*\.)?[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$/;
  if (!dnsRegex.test(pattern)) {
    return { valid: false, error: 'Invalid DNS pattern' };
  }
  
  return { valid: true };
}

/**
 * Validate and format upstream URL
 */
export function validateUpstream(upstream: string): { valid: boolean; error?: string } {
  if (!upstream.trim()) {
    return { valid: false, error: 'Upstream is required' };
  }
  
  // Check for common upstream formats
  const formats = [
    /^udp:[a-zA-Z0-9.-]+:\d+$/,                    // udp:host:port
    /^tcp:[a-zA-Z0-9.-]+:\d+$/,                    // tcp:host:port
    /^https?:\/\/[a-zA-Z0-9.-]+(\/.*)?$/,          // http(s)://host/path
    /^[a-zA-Z0-9.-]+:\d+$/,                        // host:port
    /^[a-zA-Z0-9.-]+$/                             // host only
  ];
  
  const isValid = formats.some(format => format.test(upstream));
  
  if (!isValid) {
    return { 
      valid: false, 
      error: 'Invalid upstream format. Use: udp:host:port, tcp:host:port, https://host/path, or host:port' 
    };
  }
  
  return { valid: true };
}
