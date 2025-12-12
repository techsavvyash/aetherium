const { chromium } = require('playwright');

const BASE_URL = 'http://127.0.0.1:3000';

async function testDashboard() {
  console.log('Launching browser...');
  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const context = await browser.newContext();
  const page = await context.newPage();
  page.setDefaultTimeout(60000);

  const results = [];

  // Test 1: Dashboard page
  console.log('\n=== Test 1: Dashboard Page ===');
  try {
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(2000);

    const title = await page.title();
    console.log(`Title: ${title}`);
    results.push({ test: 'Dashboard - Title', pass: title === 'Aetherium Dashboard' });

    // Check for health status
    const healthBadge = await page.locator('text=System Status').first();
    const healthVisible = await healthBadge.isVisible();
    console.log(`Health status visible: ${healthVisible}`);
    results.push({ test: 'Dashboard - Health Status', pass: healthVisible });

    // Check for stats cards
    const statsCards = await page.locator('.text-2xl.font-bold').count();
    console.log(`Stats cards found: ${statsCards}`);
    results.push({ test: 'Dashboard - Stats Cards', pass: statsCards >= 4 });

    await page.screenshot({ path: 'test-screenshots/dashboard.png', fullPage: true });
    console.log('Screenshot saved: dashboard.png');
  } catch (e) {
    console.log(`Error: ${e.message}`);
    results.push({ test: 'Dashboard Page', pass: false, error: e.message });
  }

  // Test 2: VMs page
  console.log('\n=== Test 2: VMs Page ===');
  try {
    await page.goto(`${BASE_URL}/vms`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(2000);

    // Check for Create VM button
    const createBtn = await page.locator('text=Create VM').first();
    const createBtnVisible = await createBtn.isVisible();
    console.log(`Create VM button visible: ${createBtnVisible}`);
    results.push({ test: 'VMs - Create Button', pass: createBtnVisible });

    // Check for VM table
    const vmTable = await page.locator('table').first();
    const vmTableVisible = await vmTable.isVisible();
    console.log(`VM table visible: ${vmTableVisible}`);
    results.push({ test: 'VMs - Table', pass: vmTableVisible });

    await page.screenshot({ path: 'test-screenshots/vms.png', fullPage: true });
    console.log('Screenshot saved: vms.png');
  } catch (e) {
    console.log(`Error: ${e.message}`);
    results.push({ test: 'VMs Page', pass: false, error: e.message });
  }

  // Test 3: Workers page
  console.log('\n=== Test 3: Workers Page ===');
  try {
    await page.goto(`${BASE_URL}/workers`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(2000);

    // Check for Queue Statistics card
    const queueStats = await page.locator('text=Queue Statistics').first();
    const queueStatsVisible = await queueStats.isVisible();
    console.log(`Queue Statistics visible: ${queueStatsVisible}`);
    results.push({ test: 'Workers - Queue Stats', pass: queueStatsVisible });

    await page.screenshot({ path: 'test-screenshots/workers.png', fullPage: true });
    console.log('Screenshot saved: workers.png');
  } catch (e) {
    console.log(`Error: ${e.message}`);
    results.push({ test: 'Workers Page', pass: false, error: e.message });
  }

  // Test 4: Tasks page
  console.log('\n=== Test 4: Tasks Page ===');
  try {
    await page.goto(`${BASE_URL}/tasks`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(2000);

    // Check for filter dropdowns
    const filterDropdowns = await page.locator('button[role="combobox"]').count();
    console.log(`Filter dropdowns found: ${filterDropdowns}`);
    results.push({ test: 'Tasks - Filters', pass: filterDropdowns >= 2 });

    await page.screenshot({ path: 'test-screenshots/tasks.png', fullPage: true });
    console.log('Screenshot saved: tasks.png');
  } catch (e) {
    console.log(`Error: ${e.message}`);
    results.push({ test: 'Tasks Page', pass: false, error: e.message });
  }

  // Test 5: API Explorer page
  console.log('\n=== Test 5: API Explorer Page ===');
  try {
    await page.goto(`${BASE_URL}/api-explorer`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(2000);

    // Check for Quick Actions
    const quickActions = await page.locator('text=Quick Actions').first();
    const quickActionsVisible = await quickActions.isVisible();
    console.log(`Quick Actions visible: ${quickActionsVisible}`);
    results.push({ test: 'API Explorer - Quick Actions', pass: quickActionsVisible });

    // Check for Send button
    const sendBtn = await page.locator('text=Send').first();
    const sendBtnVisible = await sendBtn.isVisible();
    console.log(`Send button visible: ${sendBtnVisible}`);
    results.push({ test: 'API Explorer - Send Button', pass: sendBtnVisible });

    await page.screenshot({ path: 'test-screenshots/api-explorer.png', fullPage: true });
    console.log('Screenshot saved: api-explorer.png');
  } catch (e) {
    console.log(`Error: ${e.message}`);
    results.push({ test: 'API Explorer Page', pass: false, error: e.message });
  }

  // Test 6: Settings page
  console.log('\n=== Test 6: Settings Page ===');
  try {
    await page.goto(`${BASE_URL}/settings`, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(2000);

    // Check for API Connection card
    const apiConnection = await page.locator('text=API Connection').first();
    const apiConnectionVisible = await apiConnection.isVisible();
    console.log(`API Connection visible: ${apiConnectionVisible}`);
    results.push({ test: 'Settings - API Connection', pass: apiConnectionVisible });

    await page.screenshot({ path: 'test-screenshots/settings.png', fullPage: true });
    console.log('Screenshot saved: settings.png');
  } catch (e) {
    console.log(`Error: ${e.message}`);
    results.push({ test: 'Settings Page', pass: false, error: e.message });
  }

  // Test 7: Navigation
  console.log('\n=== Test 7: Navigation ===');
  try {
    await page.goto(BASE_URL, { waitUntil: 'domcontentloaded', timeout: 60000 });
    await page.waitForTimeout(1000);

    // Check sidebar links
    const sidebarLinks = ['Dashboard', 'Virtual Machines', 'Workers', 'Tasks', 'API Explorer', 'Settings'];
    let linksFound = 0;
    for (const link of sidebarLinks) {
      const el = await page.locator(`text=${link}`).first();
      if (await el.isVisible()) {
        linksFound++;
      }
    }
    console.log(`Sidebar links found: ${linksFound}/${sidebarLinks.length}`);
    results.push({ test: 'Navigation - Sidebar Links', pass: linksFound >= 5 });

  } catch (e) {
    console.log(`Error: ${e.message}`);
    results.push({ test: 'Navigation', pass: false, error: e.message });
  }

  await browser.close();

  // Print summary
  console.log('\n========== TEST SUMMARY ==========');
  const passed = results.filter(r => r.pass).length;
  const failed = results.filter(r => !r.pass).length;
  console.log(`Total: ${results.length} | Passed: ${passed} | Failed: ${failed}`);
  console.log('');

  for (const r of results) {
    const status = r.pass ? '✓' : '✗';
    console.log(`${status} ${r.test}${r.error ? ` - ${r.error}` : ''}`);
  }

  return failed === 0;
}

testDashboard().then(success => {
  process.exit(success ? 0 : 1);
}).catch(e => {
  console.error('Test failed:', e);
  process.exit(1);
});
