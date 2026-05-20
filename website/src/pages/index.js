import Heading from '@theme/Heading';
import Link from '@docusaurus/Link';
import Layout from '@theme/Layout';
import styles from './index.module.css';

const sections = [
  {
    num: '01',
    tag: 'workflow',
    title: 'Generate with buf',
    to: '/docs/buf',
    detail: 'Wire protoc-gen-gdscript into a buf.gen.yaml pipeline.',
  },
  {
    num: '02',
    tag: 'runtime',
    title: 'Use in Godot',
    to: '/docs/generated-code',
    detail: 'Construct messages, serialize bytes, decode responses.',
  },
  {
    num: '03',
    tag: 'support',
    title: 'Feature matrix',
    to: '/docs/feature-support',
    detail: 'Which proto3 features are wired up today.',
  },
];

const stats = [
  {label: 'TARGET', value: 'Godot 4.6+'},
  {label: 'SOURCE', value: 'proto3'},
  {label: 'OUTPUT', value: 'GDScript'},
  {label: 'RUNTIME', value: 'static'},
];

export default function Home() {
  return (
    <Layout
      title="Protocol Buffers to GDScript"
      description="gdproto — a Protocol Buffers v3 to GDScript compiler for Godot 4.6+">
      <main className={styles.main}>
        <div className={styles.gridOverlay} aria-hidden />

        <section className={styles.hero}>
          <div className={styles.heroContainer}>
            <div className={styles.statusBar}>
              <span className={styles.statusItem}>
                <span className={styles.pulseDot} />
                SYSTEM ONLINE
              </span>
              <span className={styles.statusItem}>v1.0 / STABLE</span>
              <span className={styles.statusItem}>BUILD 2026.05</span>
            </div>

            <div className={styles.heroBody}>
              <Heading as="h1" className={styles.title}>
                <span className={styles.titleLine}>
                  <span className={styles.titleBracket}>[</span>
                  gdproto
                  <span className={styles.titleBracket}>]</span>
                </span>
                <span className={styles.titleSub}>
                  proto<span className={styles.slash}>/</span>
                  <span className={styles.accent}>gdscript</span> compiler
                </span>
              </Heading>

              <p className={styles.lede}>
                Generate Godot 4.6+ GDScript wrappers from Protocol Buffers v3
                schemas. Built for game engineers who treat their network
                layer as carefully as their physics.
              </p>

              <div className={styles.heroActions}>
                <Link className="button button--primary" to="/docs/quickstart">
                  <span className={styles.btnContent}>
                    <span>Quickstart</span>
                    <span className={styles.btnArrow}>→</span>
                  </span>
                </Link>
                <Link className="button button--secondary" to="/docs/buf">
                  <span className={styles.btnContent}>
                    <span>Read buf setup</span>
                  </span>
                </Link>
              </div>
            </div>

            <div className={styles.heroAside}>
              <div className={styles.terminal}>
                <div className={styles.terminalBar}>
                  <div className={styles.terminalDots}>
                    <span />
                    <span />
                    <span />
                  </div>
                  <span className={styles.terminalTitle}>~/game · zsh</span>
                  <span className={styles.terminalBadge}>LIVE</span>
                </div>
                <pre className={styles.terminalBody}>
                  <code>
                    <span className={styles.line}>
                      <span className={styles.lineNo}>01</span>
                      <span className={styles.prompt}>$</span>{' '}
                      <span className={styles.cmd}>buf</span> generate
                    </span>
                    <span className={styles.lineCmt}>
                      <span className={styles.lineNo}>02</span>
                      → generated/player.pb.gd
                    </span>
                    <span className={styles.lineCmt}>
                      <span className={styles.lineNo}>03</span>
                      → generated/world.pb.gd
                    </span>
                    <span className={styles.lineCmt}>
                      <span className={styles.lineNo}>04</span>
                      → generated/proto_core_utils.gd
                    </span>
                    <span className={styles.line}>
                      <span className={styles.lineNo}>05</span>
                      <span className={styles.prompt}>$</span>{' '}
                      <span className={styles.cmd}>godot</span>{' '}
                      <span className={styles.flag}>--headless</span>{' '}
                      <span className={styles.flag}>--import</span>
                    </span>
                    <span className={styles.line}>
                      <span className={styles.lineNo}>06</span>
                      <span className={styles.cursor}>▍</span>
                    </span>
                  </code>
                </pre>
              </div>
            </div>
          </div>

          <dl className={styles.stats}>
            {stats.map((s) => (
              <div key={s.label} className={styles.stat}>
                <dt>{s.label}</dt>
                <dd>{s.value}</dd>
              </div>
            ))}
          </dl>
        </section>

        <section className={styles.docsSection}>
          <div className={styles.docsHead}>
            <span className={styles.kicker}>
              <span className={styles.kickerBar} />
              SECTIONS / 03
            </span>
            <h2 className={styles.docsTitle}>
              Three vectors{' '}
              <span className={styles.accent}>through the docs.</span>
            </h2>
          </div>

          <div className={styles.cardGrid}>
            {sections.map((s) => (
              <Link to={s.to} className={styles.card} key={s.to}>
                <div className={styles.cardTop}>
                  <span className={styles.cardNum}>{s.num}</span>
                  <span className={styles.cardTag}>{s.tag}</span>
                </div>
                <div className={styles.cardBody}>
                  <h3 className={styles.cardTitle}>{s.title}</h3>
                  <p className={styles.cardDetail}>{s.detail}</p>
                </div>
                <div className={styles.cardArrow}>
                  <span>OPEN</span>
                  <span className={styles.cardArrowIcon}>→</span>
                </div>
                <span className={styles.cardCorner} aria-hidden />
              </Link>
            ))}
          </div>
        </section>

        <section className={styles.colophon}>
          <div className={styles.colophonContent}>
            <span className={styles.kicker}>
              <span className={styles.kickerBar} />
              ABOUT
            </span>
            <p>
              Built and maintained by{' '}
              <a href="https://www.cafecito.games/">Cafecito Games</a>{' '}
              for engineers who would rather ship than handwrite wire formats.
              Open source.
            </p>
          </div>
        </section>
      </main>
    </Layout>
  );
}
