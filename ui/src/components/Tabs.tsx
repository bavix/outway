interface Tab {
  id: string;
  label: string;
  href?: string;
}

interface TabsProps {
  tabs: Tab[];
  activeTab: string;
  onTabChange: (tabId: string) => void;
  className?: string;
}

export function Tabs({ tabs, activeTab, onTabChange, className = '' }: TabsProps) {
  return (
    <nav className={`nav ${className}`}>
      {tabs.map((tab) => (
        <a
          key={tab.id}
          href={tab.href || `#${tab.id}`}
          className={`nav-link ${activeTab === tab.id ? 'active' : ''}`}
          onClick={(e) => {
            e.preventDefault();
            onTabChange(tab.id);
          }}
        >
          {tab.label}
        </a>
      ))}
    </nav>
  );
}
