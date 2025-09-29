/**
 * Format timestamp to mm:ss format for chart labels
 */
export function formatTimeLabel(timestamp: number, windowSeconds: number): string {
  const now = Date.now();
  const timeDiff = now - timestamp;
  const seconds = Math.floor(timeDiff / 1000);
  
  if (seconds < 60) {
    return `${seconds}s`;
  }
  
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  
  if (windowSeconds <= 300) { // 5 minutes or less
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  }
  
  return `${minutes}m`;
}

/**
 * Generate time labels for chart based on window length
 */
export function generateTimeLabels(windowSeconds: number, dataPoints: number): string[] {
  const now = Date.now();
  const interval = (windowSeconds * 1000) / Math.max(dataPoints - 1, 1);
  
  return Array.from({ length: dataPoints }, (_, i) => {
    const timestamp = now - (dataPoints - 1 - i) * interval;
    return formatTimeLabel(timestamp, windowSeconds);
  });
}


/**
 * Format RPS (requests per second) value
 */
export function formatRPS(rps: number): string {
  if (rps < 1) {
    return `${(rps * 1000).toFixed(0)}/s`;
  }
  
  if (rps < 1000) {
    return `${rps.toFixed(2)}/s`;
  }
  
  return `${(rps / 1000).toFixed(1)}k/s`;
}

/**
 * Format timestamp to relative time (e.g., "2m ago", "1h ago")
 */
export function formatRelativeTime(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp;
  
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  
  if (seconds < 60) {
    return `${seconds}s ago`;
  }
  
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  
  if (hours < 24) {
    return `${hours}h ago`;
  }
  
  return `${days}d ago`;
}
