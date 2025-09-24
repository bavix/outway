import { useState } from 'preact/hooks';
import { Card } from '../components/Card.js';
import { Button } from '../components/Button.js';
import { Input } from '../components/Input.js';
import { FailoverProvider } from '../providers/failoverProvider.js';

interface ResolveProps {
  provider: FailoverProvider;
}

export function Resolve({ provider }: ResolveProps) {
  const [name, setName] = useState('');
  const [type, setType] = useState<string>('A');
  const [result, setResult] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (e: Event) => {
    e.preventDefault();
    if (!name.trim()) return;
    setLoading(true);
    setResult('');
    try {
      const res = await provider.testResolve(name.trim(), type);
      const header = `Upstream: ${res.upstream} | RCODE: ${res.rcode} | Answers: ${res.answers}`;
      const body = (res.records || []).join('\n');
      setResult(`${header}\n${body}`);
    } catch (err) {
      setResult((err as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Resolve</h1>
        <p className="text-gray-600 dark:text-gray-400 mt-1">Run a test DNS resolve via the active pipeline</p>
      </div>

      <Card>
        <form onSubmit={submit} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <Input
              label="Domain"
              value={name}
              onInput={(e) => setName((e.target as HTMLInputElement).value)}
              placeholder="example.com"
              required
            />

            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type</label>
              <select
                className="w-full px-3 py-2 h-[42px] border border-gray-300 rounded-md bg-white text-gray-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-800 dark:text-gray-100 dark:border-gray-600 dark:focus:border-blue-400 dark:focus:ring-blue-400"
                value={type}
                onChange={(e) => setType(e.currentTarget.value)}
              >
                <option value="A">A</option>
                <option value="AAAA">AAAA</option>
                <option value="CNAME">CNAME</option>
                <option value="MX">MX</option>
                <option value="NS">NS</option>
                <option value="TXT">TXT</option>
                <option value="SRV">SRV</option>
                <option value="PTR">PTR</option>
              </select>
            </div>

            <div className="flex items-end">
              <Button type="submit" variant="primary" fullWidth loading={loading} disabled={loading}>
                {loading ? 'Resolving…' : 'Resolve'}
              </Button>
            </div>
          </div>

          {result && (
            <div>
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm text-gray-600 dark:text-gray-400">Result</span>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => navigator.clipboard.writeText(result)}
                  >
                    Copy
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => setResult('')}
                  >
                    Clear
                  </Button>
                </div>
              </div>
              <pre className="mt-1 p-3 rounded-md bg-gray-50 text-gray-800 dark:bg-gray-900 dark:text-gray-200 whitespace-pre-wrap text-sm border border-gray-200 dark:border-gray-700 font-mono leading-6 max-h-96 overflow-auto">
                {result}
              </pre>
            </div>
          )}
        </form>
      </Card>
    </div>
  );
}


