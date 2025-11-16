import { useRef, useEffect, useCallback } from 'preact/hooks';
import { Chart, ChartConfiguration, registerables } from 'chart.js';

// Register Chart.js components
Chart.register(...registerables);

interface UseChartOptions {
  type: 'line' | 'bar';
  data: {
    labels: string[];
    datasets: Array<{
      label: string;
      data: number[];
      borderColor?: string;
      backgroundColor?: string;
      fill?: boolean;
    }>;
  };
  options?: Partial<ChartConfiguration['options']>;
}

export function useChart(options: UseChartOptions) {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const chartRef = useRef<Chart | null>(null);

  const createChart = useCallback(() => {
    if (!canvasRef.current) return;

    const ctx = canvasRef.current.getContext('2d');
    if (!ctx) return;

    // Destroy existing chart
    if (chartRef.current) {
      chartRef.current.destroy();
    }

    // Default options for Outway charts
    const defaultOptions: ChartConfiguration['options'] = {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 250, easing: 'easeOutQuart' },
      interaction: { intersect: false, mode: 'index' },
      layout: { padding: { left: 6, right: 6, top: 4, bottom: 4 } },
      scales: {
        x: {
          grid: { color: 'rgba(0,0,0,0.06)' },
          ticks: { 
            maxTicksLimit: 6, 
            color: '#6b7280', 
            autoSkip: true 
          }
        },
        y: {
          beginAtZero: true,
          grid: { color: 'rgba(0,0,0,0.06)' },
          ticks: { color: '#6b7280' }
        }
      },
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            label: (context) => {
              const value = context.parsed.y;
              if (value === null || value === undefined) {
                return 'N/A';
              }
              const isRPS = context.dataset.label === 'RPS';
              return isRPS 
                ? `${value.toFixed(2)} rps` 
                : `${value.toFixed(1)} ms`;
            }
          }
        }
      },
      ...options.options
    };

    // Default dataset styling
    const datasets = options.data.datasets.map(dataset => ({
      ...dataset,
      borderColor: dataset.borderColor || '#111827',
      backgroundColor: dataset.backgroundColor || 'rgba(17, 24, 39, 0.1)',
      fill: dataset.fill ?? true,
      tension: 0.4,
      pointRadius: 0,
      pointHoverRadius: 4,
      borderWidth: 2
    }));

    const config: ChartConfiguration = {
      type: options.type,
      data: {
        ...options.data,
        datasets
      },
      options: defaultOptions
    };

    chartRef.current = new Chart(ctx, config);
  }, [options]);

  const updateChart = useCallback(() => {
    if (chartRef.current) {
      chartRef.current.data.labels = options.data.labels;
      chartRef.current.data.datasets.forEach((dataset, index) => {
        if (options.data.datasets[index]) {
          dataset.data = options.data.datasets[index].data;
        }
      });
      chartRef.current.update('none');
    }
  }, [options.data]);

  // Debounced update function to reduce chart updates
  const debouncedUpdate = useCallback((() => {
    let timeoutId: number;
    return () => {
      clearTimeout(timeoutId);
      timeoutId = window.setTimeout(() => updateChart(), 200); // 200ms debounce
    };
  })(), [updateChart]);

  useEffect(() => {
    createChart();
    
    return () => {
      if (chartRef.current) {
        chartRef.current.destroy();
        chartRef.current = null;
      }
    };
  }, [createChart]);

  useEffect(() => {
    debouncedUpdate();
  }, [debouncedUpdate]);

  return canvasRef;
}
