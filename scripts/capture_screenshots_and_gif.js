const puppeteer = require('puppeteer-core');
const path = require('path');
const fs = require('fs');
const { execSync } = require('child_process');

const CHROME_PATH = 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const OUTPUT_DIR = path.join(__dirname, '..', 'docs', 'images');
const FRAMES_DIR = path.join(__dirname, '..', 'docs', 'images', 'temp_frames');

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function capture() {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  if (fs.existsSync(FRAMES_DIR)) {
    fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
  }
  fs.mkdirSync(FRAMES_DIR, { recursive: true });

  console.log('Launching headless Chrome via puppeteer-core...');
  const browser = await puppeteer.launch({
    executablePath: CHROME_PATH,
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--window-size=1440,900'],
  });

  const page = await browser.newPage();
  page.on('pageerror', (err) => console.error('PAGE_ERROR:', err.message));
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });

  // 1. Dashboard Desktop Screenshot
  console.log('1. Navigating to Dashboard...');
  await page.goto('http://localhost:8085', { waitUntil: 'domcontentloaded' });
  await sleep(2500);

  const dashboardPath = path.join(OUTPUT_DIR, 'dashboard_preview.png');
  await page.screenshot({ path: dashboardPath, fullPage: false });
  console.log('✓ Captured desktop dashboard to', dashboardPath);

  // 2. Mobile Responsive Viewport Screenshot (iPhone 14/15 size: 390x844)
  console.log('2. Capturing Mobile Responsive Dashboard...');
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 2 });
  await sleep(1000);
  const mobilePath = path.join(OUTPUT_DIR, 'mobile_dashboard_preview.png');
  await page.screenshot({ path: mobilePath, fullPage: false });
  console.log('✓ Captured mobile dashboard to', mobilePath);

  // Reset to Desktop Viewport
  await page.setViewport({ width: 1440, height: 900, deviceScaleFactor: 2 });
  await sleep(500);

  // 3. Breakpoint Debugger / Workflow DAG view
  console.log('3. Navigating to Breakpoint Debugger / Workflow DAG view...');
  // Click inspect button for the execution paused at breakpoint
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const inspectBtn = buttons.find((b) => b.textContent && b.textContent.includes('Inspect'));
    if (inspectBtn) inspectBtn.click();
  });
  await sleep(2000);

  const workflowDAGPath = path.join(OUTPUT_DIR, 'workflow_dag_preview.png');
  await page.screenshot({ path: workflowDAGPath, fullPage: false });
  console.log('✓ Captured Workflow DAG / Breakpoint Debugger to', workflowDAGPath);

  // 4. DLQ Review & Side-by-Side Visual State Diff View
  console.log('4. Navigating to DLQ Review & Visual State Diff...');
  // Click DLQ Review in navigation
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const dlqNav = buttons.find((b) => b.textContent && b.textContent.includes('DLQ Review'));
    if (dlqNav) dlqNav.click();
  });
  await sleep(1500);

  // Click Review & Resume on DLQ incident
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const reviewBtn = buttons.find((b) => b.textContent && (b.textContent.includes('Review & Resume') || b.textContent.includes('Inspect')));
    if (reviewBtn) reviewBtn.click();
  });
  await sleep(1800);

  // Switch to Visual Diff tab
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const diffBtn = buttons.find((b) => b.textContent && b.textContent.includes('Visual Diff'));
    if (diffBtn) diffBtn.click();
  });
  await sleep(1500);

  const dlqDiffPath = path.join(OUTPUT_DIR, 'dlq_diff_preview.png');
  await page.screenshot({ path: dlqDiffPath, fullPage: false });
  console.log('✓ Captured DLQ Visual State Diff to', dlqDiffPath);

  // 5. Capture Sequential Animation Frames for GIF
  console.log('5. Capturing animation frames for demo GIF...');
  let frameIndex = 1;

  async function snapFrame() {
    const num = String(frameIndex).padStart(3, '0');
    const fPath = path.join(FRAMES_DIR, `frame_${num}.png`);
    await page.screenshot({ path: fPath, fullPage: false });
    frameIndex++;
  }

  // Scene A: DLQ Visual Diff & Patch View (8 frames)
  for (let i = 0; i < 8; i++) {
    await snapFrame();
    await sleep(250);
  }

  // Scene B: Back to Overview Dashboard (8 frames)
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const dashBtn = buttons.find((b) => b.textContent && b.textContent.includes('Dashboard'));
    if (dashBtn) dashBtn.click();
  });
  await sleep(800);

  for (let i = 0; i < 8; i++) {
    await snapFrame();
    await sleep(250);
  }

  // Scene C: Click Breakpoint Execution & Step Debugger (10 frames)
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const inspectBtn = buttons.find((b) => b.textContent && b.textContent.includes('Inspect'));
    if (inspectBtn) inspectBtn.click();
  });
  await sleep(800);

  for (let i = 0; i < 10; i++) {
    await snapFrame();
    await sleep(250);
  }

  // Scene D: Switch to Compliance Replay tab (8 frames)
  await page.evaluate(() => {
    const buttons = Array.from(document.querySelectorAll('button'));
    const replayBtn = buttons.find((b) => b.textContent && b.textContent.includes('Compliance Replay'));
    if (replayBtn) replayBtn.click();
  });
  await sleep(800);

  for (let i = 0; i < 8; i++) {
    await snapFrame();
    await sleep(250);
  }

  console.log(`✓ Captured ${frameIndex - 1} animation frames.`);

  // Clean close to avoid SSE lock
  const pages = await browser.pages();
  await Promise.all(pages.map((p) => p.close().catch(() => {})));
  await browser.close();

  // 6. Generate GIF using FFmpeg
  console.log('6. Compiling demo GIF using ffmpeg...');
  const gifPath = path.join(OUTPUT_DIR, 'agentloop_demo.gif');
  const inputPattern = path.join(FRAMES_DIR, 'frame_%03d.png');

  // Palette generation for ultra-crisp GIF without color banding
  const palettePath = path.join(FRAMES_DIR, 'palette.png');
  execSync(`ffmpeg -y -i "${inputPattern}" -vf "fps=5,scale=1020:-1:flags=lanczos,palettegen" "${palettePath}"`);
  execSync(`ffmpeg -y -i "${inputPattern}" -i "${palettePath}" -lavfi "fps=5,scale=1020:-1:flags=lanczos [x]; [x][1:v] paletteuse=dither=bayer:bayer_scale=3" "${gifPath}"`);

  console.log('✓ Successfully generated animated demo GIF at', gifPath);

  // Clean up frames directory
  fs.rmSync(FRAMES_DIR, { recursive: true, force: true });
  console.log('✓ Cleaned up temporary frames.');
}

capture().catch((err) => {
  console.error('Capture failed:', err);
  process.exit(1);
});
