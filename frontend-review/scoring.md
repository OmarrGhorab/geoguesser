# Scoring

## Scale

Use 0-100. Start each category at 100, subtract for issues, then calibrate against production risk.

## Category Weights

- Architecture: 12
- Next.js: 12
- TypeScript: 8
- Performance: 10
- Accessibility: 10
- Security: 12
- Maintainability: 10
- Developer Experience: 6
- SEO: 5
- Testing: 7
- Localization: 5
- State Management: 3

## Deductions

- Critical: subtract 25-40 from relevant categories; overall score cannot exceed 60 with any Critical issue.
- High: subtract 10-20; overall score usually cannot exceed 80 with any unresolved High issue.
- Medium: subtract 4-9.
- Low: subtract 1-3.
- Suggestion: subtract 0-1 only if it affects production quality.

## Score Bands

- 90-100: production-ready with minor notes.
- 80-89: good but needs targeted fixes.
- 70-79: risky; several meaningful issues.
- 60-69: not ready; significant rework needed.
- 0-59: block merge.

## Calibration

Be strict but fair. A small diff can score low if it touches auth, payments, forms, or app architecture incorrectly. A large diff can score high if it follows the stack contract, has tests, and handles edge cases.

