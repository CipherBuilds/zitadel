"use strict";
const fs = require("node:fs");
const report = JSON.parse(fs.readFileSync("/opt/copperholt-login-tests/login-tests.json", "utf8"));
const tests = report.testResults.flatMap(file => file.assertionResults.map(test => ({
  name: test.fullName, status: test.status
})));
console.log(JSON.stringify({kind: "LOGIN_AFFECTED_TESTS", total: report.numTotalTests,
  passed: report.numPassedTests, failed: report.numFailedTests,
  pending: report.numPendingTests, tests}));
if (!report.success || report.numFailedTests !== 0 || report.numPendingTests !== 0 ||
    tests.length !== report.numTotalTests || tests.some(test => test.status !== "passed")) {
  process.exitCode = 1;
}
