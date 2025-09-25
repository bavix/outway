import { useState, useEffect } from 'preact/hooks';
import { Card, Button, Input, Badge } from '../components/index.js';
import { useRuleGroups, useRuleGroupsActions } from '../store/store.js';
import { FailoverProvider } from '../providers/failoverProvider.js';
import { RuleGroup } from '../providers/types.js';

interface RulesProps {
  provider: FailoverProvider;
}

export function Rules({ provider }: RulesProps) {
  const ruleGroups = useRuleGroups();
  const { setRuleGroups, setError } = useRuleGroupsActions();
  
  const [groupName, setGroupName] = useState('');
  const [description, setDescription] = useState('');
  const [via, setVia] = useState('');
  const [patterns, setPatterns] = useState<string[]>(['']);
  const [pinTTL, setPinTTL] = useState(true);
  const [filter, setFilter] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [editing, setEditing] = useState<Record<string, RuleGroup | null>>({});

  // Subscribe to rule groups updates
  useEffect(() => {
    const unsubscribe = provider.onRuleGroups((newGroups: RuleGroup[]) => {
      setRuleGroups(newGroups);
    });

    // Load initial rule groups
    provider.fetchRuleGroups()
      .then(setRuleGroups)
      .catch((error: any) => {
        console.error('Failed to fetch rule groups:', error);
        setError(error.message);
      });

    return unsubscribe;
  }, [provider, setRuleGroups, setError]);

  // Filter rule groups based on search
  const filteredGroups = ruleGroups.filter(group => 
    group.name.toLowerCase().includes(filter.toLowerCase()) ||
    group.description?.toLowerCase().includes(filter.toLowerCase()) ||
    group.patterns.some(pattern => pattern.toLowerCase().includes(filter.toLowerCase()))
  );

  const handleAddPattern = () => {
    setPatterns([...patterns, '']);
  };

  const handleRemovePattern = (index: number) => {
    setPatterns(patterns.filter((_, i) => i !== index));
  };

  const handlePatternChange = (index: number, value: string) => {
    const newPatterns = [...patterns];
    newPatterns[index] = value;
    setPatterns(newPatterns);
  };

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    
    if (!groupName.trim() || !via.trim() || patterns.some(p => !p.trim())) {
      setError('Please fill in all required fields');
      return;
    }

    setIsSubmitting(true);
    
    try {
      const newGroup: RuleGroup = {
        name: groupName.trim(),
        description: description.trim() || '',
        via: via.trim(),
        patterns: patterns.filter(p => p.trim()),
        pin_ttl: pinTTL
      };

      await provider.createRuleGroup(newGroup);
      
      // Reset form
      setGroupName('');
      setDescription('');
      setVia('');
      setPatterns(['']);
      setPinTTL(true);
    } catch (error) {
      console.error('Failed to create rule group:', error);
      setError(error instanceof Error ? error.message : 'Failed to create rule group');
    } finally {
      setIsSubmitting(false);
    }
  };

  const startEdit = (g: RuleGroup) => {
    setEditing(prev => ({ ...prev, [g.name]: { ...g } }));
  };

  const cancelEdit = (name: string) => {
    setEditing(prev => { const n = { ...prev }; delete n[name]; return n; });
  };

  const updateEditField = (name: string, field: keyof RuleGroup, value: any) => {
    setEditing(prev => ({
      ...prev,
      [name]: { ...(prev[name] as RuleGroup), [field]: value }
    }));
  };

  const updateEditPattern = (name: string, idx: number, value: string) => {
    setEditing(prev => {
      const g = prev[name] as RuleGroup; if (!g) return prev;
      const patterns = g.patterns.slice(); patterns[idx] = value;
      return { ...prev, [name]: { ...g, patterns } };
    });
  };

  const addEditPattern = (name: string) => {
    setEditing(prev => {
      const g = prev[name] as RuleGroup; if (!g) return prev;
      return { ...prev, [name]: { ...g, patterns: [...g.patterns, ''] } };
    });
  };

  const removeEditPattern = (name: string, idx: number) => {
    setEditing(prev => {
      const g = prev[name] as RuleGroup; if (!g) return prev;
      const patterns = g.patterns.filter((_, i) => i !== idx);
      return { ...prev, [name]: { ...g, patterns } };
    });
  };

  const saveEdit = async (name: string) => {
    const g = editing[name]; if (!g) return;
    try {
      await provider.updateRuleGroup(name, {
        name: g.name,
        description: g.description || '',
        via: g.via,
        patterns: g.patterns.filter(p => p.trim()),
        pin_ttl: !!g.pin_ttl,
      });
      cancelEdit(name);
      // Refresh groups to reflect any server-side normalization
      const groups = await provider.fetchRuleGroups();
      setRuleGroups(groups);
    } catch (error) {
      console.error('Failed to update rule group:', error);
      setError(error instanceof Error ? error.message : 'Failed to update rule group');
    }
  };

  const handleDelete = async (groupName: string) => {
    try {
      await provider.deleteRuleGroup(groupName);
      setDeleteConfirm(null);
    } catch (error) {
      console.error('Failed to delete rule group:', error);
      setError(error instanceof Error ? error.message : 'Failed to delete rule group');
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">Rule Groups</h1>
          <p className="text-gray-600 dark:text-gray-400 mt-1">
            Manage DNS routing rules and patterns
          </p>
        </div>
      </div>

      {/* Add Rule Group Form */}
      <Card title="Add New Rule Group" subtitle="Create a new routing rule group">
        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Input
              label="Group Name"
              value={groupName}
              onInput={(e) => setGroupName((e.target as HTMLInputElement).value)}
              placeholder="e.g., YouTube"
              required
            />
            
            <Input
              label="Via Interface"
              value={via}
              onInput={(e) => setVia((e.target as HTMLInputElement).value)}
              placeholder="e.g., utun4"
              required
            />
          </div>

          <Input
            label="Description"
            value={description}
            onInput={(e) => setDescription((e.target as HTMLInputElement).value)}
            placeholder="Optional description"
          />

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
              DNS Patterns *
            </label>
            {patterns.map((pattern, index) => (
              <div key={index} className="flex gap-3 mb-3">
                <div className="flex-1">
                  <Input
                    value={pattern}
                    onInput={(e) => handlePatternChange(index, (e.target as HTMLInputElement).value)}
                    placeholder="e.g., *.youtube.com"
                    required
                  />
                </div>
                {patterns.length > 1 && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => handleRemovePattern(index)}
                  >
                    Remove
                  </Button>
                )}
              </div>
            ))}
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={handleAddPattern}
            >
              Add Pattern
            </Button>
          </div>

          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              id="pin-ttl"
              checked={pinTTL}
              onChange={(e) => setPinTTL((e.target as HTMLInputElement).checked)}
              className="rounded border-gray-300 dark:border-gray-600 text-black dark:text-white focus:ring-black dark:focus:ring-white"
            />
            <label htmlFor="pin-ttl" className="text-sm font-medium text-gray-700 dark:text-gray-300">
              <span className="input-label">Pin TTL (keep IP/route until TTL expires)</span>
            </label>
          </div>

          <div className="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
            <Button
              type="submit"
              variant="primary"
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Creating...' : 'Create Rule Group'}
            </Button>
          </div>
        </form>
      </Card>

      {/* Search */}
      <div>
        <Input
          label="Search Rule Groups"
          value={filter}
          onInput={(e) => setFilter((e.target as HTMLInputElement).value)}
          placeholder="Search by name, description, or pattern..."
          hint="Filter rule groups by name, description, or DNS patterns"
        />
      </div>

      {/* Rule Groups List */}
      <div className="grid gap-4">
        {filteredGroups.map((group) => (
          <Card key={group.name} title={group.name} subtitle={group.description}>
            <div className="flex justify-between items-start mb-4">
              <div className="flex items-center gap-3">
                <Badge variant="primary">{group.patterns.length}</Badge>
                <span className="text-sm text-gray-600 dark:text-gray-400">
                  {group.patterns.length === 1 ? 'pattern' : 'patterns'}
                </span>
              </div>
              <div className="flex gap-2">
                {editing[group.name] ? (
                  <>
                    <Button variant="secondary" size="sm" onClick={() => cancelEdit(group.name)}>Cancel</Button>
                    <Button variant="primary" size="sm" onClick={() => saveEdit(group.name)}>Save</Button>
                  </>
                ) : (
                  <Button variant="secondary" size="sm" onClick={() => startEdit(group)}>Edit</Button>
                )}
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setDeleteConfirm(group.name)}
                >
                  Delete
                </Button>
              </div>
            </div>

            {editing[group.name] ? (
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-700 dark:text-gray-300">Via Interface:</span>
                    <Input value={editing[group.name]!.via} onInput={(e) => updateEditField(group.name, 'via', (e.target as HTMLInputElement).value)} />
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-700 dark:text-gray-300">Pin TTL:</span>
                    <input type="checkbox" checked={!!editing[group.name]!.pin_ttl} onChange={(e) => updateEditField(group.name, 'pin_ttl', (e.target as HTMLInputElement).checked)} />
                  </div>
                </div>
                <div>
                  <span className="font-medium text-sm text-gray-700 dark:text-gray-300">Description:</span>
                  <Input value={editing[group.name]!.description || ''} onInput={(e) => updateEditField(group.name, 'description', (e.target as HTMLInputElement).value)} placeholder="Optional description" />
                </div>
                <div className="mt-2 pt-2 border-t border-gray-200 dark:border-gray-700">
                  <span className="font-medium text-sm text-gray-700 dark:text-gray-300">DNS Patterns:</span>
                  {editing[group.name]!.patterns.map((p, idx) => (
                    <div key={idx} className="flex gap-2 mt-2">
                      <Input value={p} onInput={(e) => updateEditPattern(group.name, idx, (e.target as HTMLInputElement).value)} />
                      <Button variant="outline" size="sm" onClick={() => removeEditPattern(group.name, idx)}>Remove</Button>
                    </div>
                  ))}
                  <Button className="mt-2" variant="secondary" size="sm" onClick={() => addEditPattern(group.name)}>Add Pattern</Button>
                </div>
              </div>
            ) : (
              <>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-700 dark:text-gray-300">Via Interface:</span>
                    <Badge variant="secondary">{group.via}</Badge>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-gray-700 dark:text-gray-300">Pin TTL:</span>
                    <Badge variant={group.pin_ttl ? 'primary' : 'secondary'}>
                      {group.pin_ttl ? 'Yes' : 'No'}
                    </Badge>
                  </div>
                </div>
                <div className="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                  <span className="font-medium text-sm text-gray-700 dark:text-gray-300">DNS Patterns:</span>
                  <div className="flex flex-wrap gap-2 mt-2">
                    {group.patterns.map((pattern, index) => (
                      <span
                        key={index}
                        className="bg-gray-100 dark:bg-gray-700 text-gray-800 dark:text-gray-200 px-2 py-1 rounded text-xs font-mono"
                      >
                        {pattern}
                      </span>
                    ))}
                  </div>
                </div>
              </>
            )}
          </Card>
        ))}
      </div>

      {filteredGroups.length === 0 && (
        <Card className="text-center py-8">
          <p className="text-secondary">
            {filter ? 'No rule groups match your search.' : 'No rule groups configured.'}
          </p>
        </Card>
      )}

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <Card className="max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold mb-4">Confirm Delete</h3>
            <p className="text-secondary mb-6">
              Are you sure you want to delete the rule group "{deleteConfirm}"? This action cannot be undone.
            </p>
            <div className="flex gap-3 justify-end">
              <Button
                variant="secondary"
                onClick={() => setDeleteConfirm(null)}
              >
                Cancel
              </Button>
              <Button
                variant="danger"
                onClick={() => handleDelete(deleteConfirm)}
              >
                Delete
              </Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}