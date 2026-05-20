const {themes: prismThemes} = require('prism-react-renderer');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'gdproto',
  tagline: 'Protocol Buffers v3 to GDScript for Godot 4.6+',
  url: 'https://cafecito-games.github.io',
  baseUrl: '/gdproto/',
  organizationName: 'cafecito-games',
  projectName: 'gdproto',
  onBrokenLinks: 'throw',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },
  trailingSlash: false,
  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },
  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          routeBasePath: 'docs',
          editUrl: 'https://github.com/cafecito-games/gdproto/tree/main/website/',
          showLastUpdateAuthor: false,
          showLastUpdateTime: false,
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],
  themeConfig: {
    metadata: [
      {
        name: 'description',
        content:
          'Documentation for gdproto, a Protocol Buffers v3 to GDScript compiler for Godot.',
      },
    ],
    navbar: {
      title: 'gdproto',
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Docs',
        },
        {
          type: 'docsVersionDropdown',
          position: 'right',
        },
        {
          href: 'https://github.com/cafecito-games/gdproto',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {label: 'Quickstart', to: '/docs/quickstart'},
            {label: 'Using buf', to: '/docs/buf'},
            {label: 'Generated GDScript', to: '/docs/generated-code'},
          ],
        },
        {
          title: 'Project',
          items: [
            {
              label: 'Cafecito Games',
              href: 'https://www.cafecito.games/',
            },
            {label: 'GitHub', href: 'https://github.com/cafecito-games/gdproto'},
            {
              label: 'Releases',
              href: 'https://github.com/cafecito-games/gdproto/releases',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} <a href="https://www.cafecito.games/">Cafecito Games LLC</a>.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'gdscript', 'protobuf', 'yaml'],
    },
  },
};

module.exports = config;
