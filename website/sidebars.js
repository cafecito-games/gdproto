/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docsSidebar: [
    'overview',
    'quickstart',
    'installation',
    {
      type: 'category',
      label: 'Generation',
      items: ['buf', 'protoc-plugin', 'direct-cli'],
    },
    {
      type: 'category',
      label: 'Godot Usage',
      items: ['generated-code', 'feature-support', 'troubleshooting'],
    },
    'development',
    'releases',
  ],
};

module.exports = sidebars;
